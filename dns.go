package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strings"

	"github.com/miekg/dns"
	"gopkg.in/yaml.v2"
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
	server    *dns.Server
	m         dnsMap
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
	data, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, s); err != nil {
		return err
	}

	if s.Protocol == "" {
		log.Println("protocol is not set, use default protocol:", defaultDNSProtocol)
		s.Protocol = defaultDNSProtocol
	}
	if s.Port == 0 {
		log.Println("port is not set, use default port:", defaultDNSPort)
		s.Port = defaultDNSPort
	}
	if s.ParentDNS == "" {
		log.Println("parentdns is not set, use default parent:", defaultParentDNS)
		s.ParentDNS = defaultParentDNS
	}
	s.server = &dns.Server{Addr: fmt.Sprintf(":%d", s.Port), Net: s.Protocol}

	// init routes (in memory)
	s.initRoutes()

	return nil
}

func (s *DNSServer) initRoutes() {
	for _, r := range s.Routes {
		if !strings.HasSuffix(r.Fqdn, ".") {
			r.Fqdn += "."
		}
		log.Println("add mock DNS:", r.Rrtype, r.Fqdn)
		switch r.Rrtype {
		case "A":
			ips := strings.Split(r.Ip, ",")
			rrs := make([]dns.RR, len(ips))
			for idx, ip := range ips {
				realIp := net.ParseIP(ip)
				if realIp == nil {
					log.Fatalln("invalid ip addr:", ip)
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
			log.Fatalln("CNAME is not supported yet")
		default:
			log.Fatalln("unsupported DNS type: ", r.Rrtype)
		}
	}
}

func (s *DNSServer) Serve() error {
	// hijack all dns requests
	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		rrs, err := s.m.Get(r.Question[0].Qtype, r.Question[0].Name)
		if err != nil {
			log.Println("handle request error:", r, err)
			log.Printf("forward request %s to parent DNS\n", r.Question[0].Name)
			resp, _, err := dnsClient.Exchange(r, s.ParentDNS)
			if err != nil {
				log.Println("forward parent DNS", w.RemoteAddr(), r.Question[0].Name, err)
				dns.HandleFailed(w, r)
				return
			}
			if err = w.WriteMsg(resp); err != nil {
				log.Println("write response msg error:", err)
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

func init() {
	ServerMap.Add("dns", newDNSServer())
}
