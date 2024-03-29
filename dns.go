package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/gookit/slog"
	"github.com/miekg/dns"
	"gopkg.in/yaml.v3"
)

// dns-mock.yaml example
//

const (
	defaultDNSPort     = 53
	defaultDNSProtocol = "udp4"
	defaultParentDNS   = "223.5.5.5:53" // aliyun public DNS
)

var dnsClient = &dns.Client{Net: "udp"}

type dnsMap map[uint16]map[string][]dns.RR // {rrtype: {fqdn: [{ip, ttl}]}}

func (c dnsMap) Get(dnsType uint16, record string) ([]dns.RR, error) {
	typeMap, exists := c[dnsType]
	if !exists {
		return nil, fmt.Errorf("%s 404 not found", record)
	}
	result, exists := typeMap[record]
	if !exists {
		return nil, fmt.Errorf("%s 404 not found", record)
	}
	return result, nil
}

func (c dnsMap) Set(dnsType uint16, key string, value []dns.RR) {
	typeMap, exists := c[dnsType]
	if !exists {
		c[dnsType] = map[string][]dns.RR{}
		typeMap = c[dnsType]
	}

	typeMap[key] = value
}

type DNSServer struct {
	Protocol  string    `yaml:"protocol"`
	Port      int       `yaml:"port"`
	ParentDNS string    `yaml:"parent"`
	Routes    []*Record `yaml:"routes"`

	server *dns.Server
	m      dnsMap
	w      *FileWatcher
}

type Record struct {
	Rrtype string `yaml:"rrtype"`
	Fqdn   string `yaml:"fqdn"`
	Ip     string `yaml:"ip"`
	Ttl    uint32 `yaml:"ttl"`
}

func newDNSServer() *DNSServer {
	return &DNSServer{m: dnsMap{}}
}

func (s *DNSServer) Init(cfgFile string) error {
	if err := s.loadConfig(cfgFile); err != nil {
		return err
	}
	s.server = &dns.Server{Addr: fmt.Sprintf(":%d", s.Port), Net: s.Protocol}

	// init routes (in memory)
	s.initRoutes()

	// add config watcher and hot reload
	s.w = NewFileWatcher()
	s.w.Watch(cfgFile, func() error {
		err := s.loadConfig(cfgFile)
		if err != nil {
			return err
		}
		s.m = dnsMap{}
		s.initRoutes()
		slog.Warn("only routes and parentdns will be auto reloaded when config update")

		return nil
	})

	return nil
}

func (s *DNSServer) loadConfig(cfgFile string) error {
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, s); err != nil {
		return err
	}

	if s.Protocol == "" {
		slog.Warnf("protocol is not set, use default protocol: %s", defaultDNSProtocol)
		s.Protocol = defaultDNSProtocol
	}
	if s.Port == 0 {
		slog.Warnf("port is not set, use default port: %d", defaultDNSPort)
		s.Port = defaultDNSPort
	}
	if s.ParentDNS == "" {
		slog.Warnf("parentdns is not set, use default parent: %s", defaultParentDNS)
		s.ParentDNS = defaultParentDNS
	}

	return nil
}

func (s *DNSServer) initRoutes() {
	for _, r := range s.Routes {
		// add "." as suffix of FQDN
		if !strings.HasSuffix(r.Fqdn, ".") {
			r.Fqdn += "."
		}
		slog.Infof("add mock DNS: %s %s", r.Rrtype, r.Fqdn)
		switch r.Rrtype {
		case "A":
			ips := strings.Split(r.Ip, ",")
			rrs := make([]dns.RR, len(ips))
			for idx, ip := range ips {
				realIp := net.ParseIP(ip)
				if realIp == nil {
					slog.Fatalf("invalid ip addr: %s", ip)
				}
				rrs[idx] = &dns.A{
					Hdr: dns.RR_Header{
						Name:   r.Fqdn,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    r.Ttl,
					},
					A: realIp,
				}
			}
			s.m.Set(dns.TypeA, r.Fqdn, rrs)
		case "CNAME":
			slog.Fatal("CNAME is not supported yet")
		default:
			slog.Fatalf("unsupported DNS type: %s", r.Rrtype)
		}
	}
}

func (s *DNSServer) Serve(wg *sync.WaitGroup) error {
	defer wg.Done()

	// hijack all dns requests
	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		rrs, err := s.m.Get(r.Question[0].Qtype, r.Question[0].Name)
		if err != nil {
			slog.Warnf("handle request %v error: %v", r.Question[0], err)
			slog.Warnf("forward request %s to parent DNS", r.Question[0].Name)
			resp, _, err := dnsClient.Exchange(r, s.ParentDNS)
			if err != nil {
				slog.Errorf("forward client %s request %s to parent DNS error: %v", w.RemoteAddr(), r.Question[0].Name, err)
				dns.HandleFailed(w, r)
				return
			}
			if err = w.WriteMsg(resp); err != nil {
				slog.Errorf("write response msg error: %v", err)
			}
			return
		}

		m := new(dns.Msg)
		m.Authoritative = true
		m.SetReply(r)
		m.Answer = rrs
		w.WriteMsg(m)
	})

	return s.server.ListenAndServe()
}

func (s *DNSServer) Shutdown() error {
	return s.server.Shutdown()
}

func init() {
	ServerMap.Add("dns", newDNSServer())
}
