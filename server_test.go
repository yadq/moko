package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"sort"
	"testing"
)

type mockServer struct{}

func (s *mockServer) Init(cfgFile string) error {
	return nil
}

func (s *mockServer) Serve() error{
	return nil
}

func TestServerMap(t *testing.T) {
	Convey("get exist server", t, func() {
		server, err := ServerMap.Get("http")
		So(err, ShouldEqual, nil)
		So(server, ShouldHaveSameTypeAs, &HttpServer{})
	})

	Convey("get unknown server", t, func() {
		server, err := ServerMap.Get("does-not-exist")
		So(err, ShouldNotEqual, nil)
		So(server, ShouldEqual, nil)
	})

	Convey("add new server", t, func() {
		server, err := ServerMap.Get("new-not-exist")
		So(err, ShouldNotEqual, nil)
		So(server, ShouldEqual, nil)
		ServerMap.Add("new-not-exist", &mockServer{})
		server, err = ServerMap.Get("new-not-exist")
		So(err, ShouldBeNil)
		So(server, ShouldNotBeNil)
	})

	Convey("list server names", t, func() {
		m := serverMap{}
		So(len(m.List()), ShouldEqual, 0)
		m.Add("new-not-exist", &mockServer{})
		So(m.List(), ShouldResemble, []string{"new-not-exist"})
		m.Add("new-not-exist2", &mockServer{})
		mlist := m.List()
		sort.SliceStable(mlist, func(i, j int) bool {
			return mlist[i] < mlist[j]
		})
		So(mlist, ShouldResemble, []string{"new-not-exist", "new-not-exist2"})
	})
}
