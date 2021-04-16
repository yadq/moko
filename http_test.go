package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func getUris(routes []*httpRoute) []string {
	uris := make([]string, len(routes))
	for idx := range routes {
		uris[idx] = routes[idx].Uri
	}

	return uris
}

func TestHTTPServer(t *testing.T) {
	s := newHttpServer()
	s.Init("examples/http-mock.yml")

	doHTTPRequest := func(method string, uri string, data io.Reader, headers map[string]string) *http.Response {
		req, _ := http.NewRequest(method, uri, data)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		return w.Result()
	}

	Convey("parse cfg file", t, func() {
		So(s.Port, ShouldEqual, defaultHTTPPort)
		uris := getUris(s.Routes)
		So(uris, ShouldContain, "/hello")
	})

	Convey("mock GET static uri", t, func() {
		resp := doHTTPRequest("GET", "/hello", nil, nil)
		So(resp.StatusCode, ShouldEqual, 201)
		So(resp.Header.Get("Content-Type"), ShouldEqual, "text/plain")
		body, _ := io.ReadAll(resp.Body)
		So(string(body), ShouldEqual, "hello world")
	})

	Convey("mock POST static uri", t, func() {
		resp := doHTTPRequest("POST", "/hello", nil, nil)
		So(resp.StatusCode, ShouldEqual, 200)
		So(resp.Header.Get("Content-Type"), ShouldEqual, "application/json")
		body, _ := io.ReadAll(resp.Body)
		So(string(body), ShouldEqual, "{\"another\":{\"sub\":\"subvalue\"},\"hello\":\"world\"}")
	})

	Convey("mock GET dynamic uri", t, func() {
		resp := doHTTPRequest("GET", "/hello/world", nil, nil)
		So(resp.StatusCode, ShouldEqual, 200)
		So(resp.Header.Get("trace-id"), ShouldEqual, "12345")
		body, _ := io.ReadAll(resp.Body)
		So(string(body), ShouldEqual, "hello world")
	})

	Convey("mock GET dynamic uri with parameters", t, func() {
		resp := doHTTPRequest("GET", "/user/20?name=world", nil, nil)
		So(resp.StatusCode, ShouldEqual, 200)
		body, _ := io.ReadAll(resp.Body)
		So(string(body), ShouldEqual, "user world 20")
	})

	Convey("mock POST form-data dynamic uri", t, func() {
		resp := doHTTPRequest("POST", "/hello/form/world", strings.NewReader("age=20"), map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
		So(resp.StatusCode, ShouldEqual, 200)
		body, _ := io.ReadAll(resp.Body)
		So(string(body), ShouldEqual, "{\"age\":\"20\",\"name\":\"world\"}")
	})

	Convey("mock POST json dynamic uri", t, func() {
		resp := doHTTPRequest("POST", "/hello/json/world", strings.NewReader("{\"age\":20,\"location\":{\"city\":\"hangzhou\"}}"), map[string]string{"Content-Type": "application/json"})
		So(resp.StatusCode, ShouldEqual, 200)
		body, _ := io.ReadAll(resp.Body)
		So(string(body), ShouldEqual, "{\"age\":\"20\",\"location\":{\"city\":\"hangzhou\"},\"name\":\"world\"}")
	})
}
