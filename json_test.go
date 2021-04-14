package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestMarshalJSON(t *testing.T) {
	Convey("marshal string", t, func() {
		v, err := MarshalJSON("hello world")
		So(err, ShouldBeNil)
		So(string(v), ShouldEqual, "\"hello world\"")
	})

	Convey("marshal simple map", t, func() {
		v, err := MarshalJSON(map[interface{}]interface{}{
			"hello": "world",
		})
		So(err, ShouldBeNil)
		So(string(v), ShouldEqual, "{\"hello\":\"world\"}")
	})

	Convey("marshal nested map", t, func() {
		v, err := MarshalJSON(map[interface{}]interface{}{
			"hello": "world",
			"sub": map[interface{}]interface{}{
				"one":     "value",
				"another": 2,
			},
		})
		So(err, ShouldBeNil)
		So(string(v), ShouldEqual, "{\"hello\":\"world\",\"sub\":{\"another\":2,\"one\":\"value\"}}")
	})

	Convey("marshal array", t, func() {
		v, err := MarshalJSON([]interface{}{
			"abcde",
			map[interface{}]interface{}{
				"hello": "world",
			},
		})
		So(err, ShouldBeNil)
		So(string(v), ShouldEqual, "[\"abcde\",{\"hello\":\"world\"}]")
	})

	Convey("marshal nested array", t, func() {
		v, err := MarshalJSON([]interface{}{
			"abcde",
			map[interface{}]interface{}{
				"hello": []interface{}{
					1, 2, "a",
				},
			},
		})
		So(err, ShouldBeNil)
		So(string(v), ShouldEqual, "[\"abcde\",{\"hello\":[1,2,\"a\"]}]")
	})
}
