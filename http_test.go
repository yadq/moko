package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"io"
	"net/http"
	"net/http/httptest"
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

	Convey("parse cfg file", t, func() {
		So(s.Port, ShouldEqual, defaultHTTPPort)
		uris := getUris(s.Routes)
		So(uris, ShouldContain, "/hello")
	})

	Convey("mock GET static uri", t, func() {
		req, _ := http.NewRequest("GET", "/hello", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, 201)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)

		So(resp.Header.Get("Content-Type"), ShouldEqual, "text/plain")
		So(string(body), ShouldEqual, "hello world")
	})

	Convey("mock POST static uri", t, func() {
		req, _ := http.NewRequest("POST", "/hello", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusOK)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)

		So(resp.Header.Get("Content-Type"), ShouldEqual, "application/json")
		So(string(body), ShouldEqual, "{\"another\":{\"sub\":\"subvalue\"},\"hello\":\"world\"}")
	})

	Convey("mock GET dynamic uri", t, func() {
		req, _ := http.NewRequest("GET", "/hello/world", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusOK)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)

		So(resp.Header.Get("trace-id"), ShouldEqual, "12345")
		So(string(body), ShouldEqual, "hello world")
	})

	Convey("mock GET dynamic uri with parameters", t, func() {
		req, _ := http.NewRequest("GET", "/user/20?name=world", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusOK)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)

		So(string(body), ShouldEqual, "user world 20")
	})

	Convey("mock POST form-data dynamic uri", t, func() {
		req, _ := http.NewRequest("POST", "/hello/world", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusOK)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)

		So(resp.Header.Get("trace-id"), ShouldEqual, "12345")
		So(string(body), ShouldEqual, "hello world")
	})

	Convey("mock POST json dynamic uri", t, func() {
		// TODO
	})
}
