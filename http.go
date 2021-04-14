package main

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strings"
)

// http-mock.yaml example
//
// port: 8181
// routes:
//   - uri: /api
//     method: GET
//     response:
//       headers:
//         Content-Type: plain/text
//       body: hello world

const (
	defaultHTTPPort   = 8181
	defaultHTTPMethod = "GET"
	defaultHTTPCode   = 200
)

type HttpServer struct {
	Port   int          `yaml:"port"`
	Routes []*httpRoute `yaml:"routes"`
	router *httprouter.Router
}

type httpRoute struct {
	Uri      string        `yaml:"uri"`
	Method   string        `yaml:"method"`
	Response *httpResponse `yaml:"response"`
}

type httpResponse struct {
	Code    int               `yaml:"code"`
	Headers map[string]string `yaml:"headers"`
	Body    interface{}       `yaml:"body"`
}

func newHttpServer() *HttpServer {
	return &HttpServer{router: httprouter.New()}
}

func (s *HttpServer) Init(cfgFile string) error {
	data, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, s); err != nil {
		return err
	}

	if s.Port == 0 {
		s.Port = defaultHTTPPort
	}
	// normalize route
	for _, r := range s.Routes {
		r.Method = strings.ToUpper(r.Method)
		if r.Method == "" {
			r.Method = defaultHTTPMethod
		}
		if r.Response.Code == 0 {
			r.Response.Code = defaultHTTPCode
		}
	}

	// init routes
	s.initRoutes()

	return nil
}

func (s *HttpServer) initRoutes() {
	for _, r := range s.Routes {
		log.Printf("try to mock: %v %v\n", r.Method, r.Uri)
		switch r.Method {
		case "GET":
			s.router.GET(r.Uri, uriHandler(r.Response))
		case "POST":
			s.router.POST(r.Uri, uriHandler(r.Response))
		case "HEAD":
			s.router.HEAD(r.Uri, uriHandler(r.Response))
		case "DELETE":
			s.router.DELETE(r.Uri, uriHandler(r.Response))
		case "PUT":
			s.router.PUT(r.Uri, uriHandler(r.Response))
		case "OPTIONS":
			s.router.OPTIONS(r.Uri, uriHandler(r.Response))
		default:
			log.Printf("Unknown method %v", r.Method)
		}
	}
}

func uriHandler(response *httpResponse) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// TODO: handle:
		// * dynamic response data
		// * delayed response
		// * chunked response
		for k, v := range response.Headers {
			w.Header().Set(k, v)
		}
		// support json string
		if reflect.TypeOf(response.Body).Kind() != reflect.String {
			if jsonBytes, err := MarshalJSON(response.Body); err == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(response.Code)
				w.Write(jsonBytes)
				return
			}
		}
		// raw string
		w.WriteHeader(response.Code)
		fmt.Fprint(w, response.Body)
	}
}

func (s *HttpServer) Serve() error {
	log.Printf("start HTTP server on :%d\n", s.Port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.Port), s.router)
}

func init() {
	ServerMap.Add("http", newHttpServer())
}
