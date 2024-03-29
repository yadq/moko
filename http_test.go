package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func getUris(routes []*httpRoute) []string {
	uris := make([]string, len(routes))
	for idx, r := range routes {
		uris[idx] = strings.Join([]string{r.Method, r.Uri}, " ")
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
		So(uris, ShouldContain, "GET /hello")
		So(uris, ShouldContain, "POST /hello")
		So(uris, ShouldContain, "GET /hello/:name")
		So(uris, ShouldContain, "POST /hello/form/:name")
		So(uris, ShouldContain, "POST /hello/json/:name")
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

	Convey("mock get delay response", t, func() {
		start := time.Now()
		resp := doHTTPRequest("GET", "/delay", nil, nil)
		duration := time.Since(start)
		So(duration.Milliseconds(), ShouldBeGreaterThanOrEqualTo, 1)
		So(resp.StatusCode, ShouldEqual, 200)
		body, _ := io.ReadAll(resp.Body)
		So(string(body), ShouldEqual, "{\"success\":true}")
	})

	Convey("mock return dynamic headers", t, func() {
		resp := doHTTPRequest("GET", "/header/abc", nil, nil)
		So(resp.StatusCode, ShouldEqual, 200)
		So(resp.Header.Get("user-name"), ShouldEqual, "abc")
	})
}

func TestHTTPSServer(t *testing.T) {
	s := newHttpServer()
	s.Init("examples/https-mock.yml")

	Convey("parse cfg file", t, func() {
		So(s.Port, ShouldEqual, defaultHTTPPort)
		uris := getUris(s.Routes)
		So(uris, ShouldContain, "GET /hello")
		So(uris, ShouldContain, "POST /hello")
		So(uris, ShouldContain, "GET /hello/:name")
	})

	Convey("mock GET static uri", t, func() {
		req, _ := http.NewRequest("GET", "/hello", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)
		resp := w.Result()
		So(resp.StatusCode, ShouldEqual, 201)
		So(resp.Header.Get("Content-Type"), ShouldEqual, "text/plain")
		body, _ := io.ReadAll(resp.Body)
		So(string(body), ShouldEqual, "hello world")
	})
}

func TestRenderString(t *testing.T) {
	params := map[string]interface{}{
		"name": "world",
	}

	Convey("no tpl usage", t, func() {
		data, err := renderString("hello world", params)
		So(err, ShouldBeNil)
		So(data, ShouldEqual, "hello world")
		data, err = renderString(`{"name": "hello world"}`, params)
		So(err, ShouldBeNil)
		So(data, ShouldEqual, `{"name": "hello world"}`)
	})

	Convey("go tpl usage", t, func() {
		data, err := renderString("hello {{.name}}", params)
		So(err, ShouldBeNil)
		So(data, ShouldEqual, "hello world")
		data, err = renderString(`{"name": "hello {{.name}}"}`, params)
		So(err, ShouldBeNil)
		So(data, ShouldEqual, `{"name": "hello world"}`)
	})

	Convey("shell tpl usage", t, func() {
		data, err := renderString("hello ${name}", params)
		So(err, ShouldBeNil)
		So(data, ShouldEqual, "hello world")
		data, err = renderString(`{"name": "hello ${name}"}`, params)
		So(err, ShouldBeNil)
		So(data, ShouldEqual, `{"name": "hello world"}`)
	})
}
