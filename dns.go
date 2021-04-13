package main

// dns-mock.yaml example
//

type DNSServer struct {
	Port int
}

func newDNSServer() *DNSServer {
	return &DNSServer{}
}

func (s *DNSServer) Init(cfgFile string) error{
	// TODO
	return nil
}

func (s *DNSServer) Serve() error {
	// TODO
	return nil
}

func init() {
	ServerMap.Add("dns", newDNSServer())
}
