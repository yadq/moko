package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestMarshalJSON(t *testing.T) {
	Convey("marshal simple map", t, func() {
		v, err := MarshalJSON(map[interface{}]interface{}{
			"hello": "world",
		})
		So(err, ShouldBeNil)
		So(string(v), ShouldEqual, "{\"hello\":\"world\"}")
	})
}
