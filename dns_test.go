package main

import (
	"testing"

	"github.com/miekg/dns"
	. "github.com/smartystreets/goconvey/convey"
)

func TestDNSServer(t *testing.T) {
	s := newDNSServer()
	s.Init("examples/dns-mock.yml")
	go s.Serve()
	client := dns.Client{Net: "udp4"}

	Convey("parse cfg file", t, func() {
		So(s.Port, ShouldEqual, 2053)
		So(s.Protocol, ShouldEqual, "udp4")
		So(s.ParentDNS, ShouldEqual, "114.114.114.114:53")
		So(len(s.Routes), ShouldEqual, 2)
	})

	Convey("query hijacked A record", t, func() {
		m1 := new(dns.Msg)
		m1.SetQuestion("www.my.internal.", dns.TypeA)
		r1, _, err := client.Exchange(m1, "127.0.0.1:2053")
		So(err, ShouldBeNil)
		So(len(r1.Answer), ShouldEqual, 2)

		m2 := new(dns.Msg)
		m2.SetQuestion("host.my.internal.", dns.TypeA)
		r2, _, err := client.Exchange(m2, "127.0.0.1:2053")
		So(err, ShouldBeNil)
		So(len(r2.Answer), ShouldEqual, 1)
	})
}
