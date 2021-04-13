package main

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
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

var (
	defaultHTTPPort = 8181
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
	Headers map[string]string `yaml:"headers"`
	Body    string            `yaml:"body"`
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
		// * status code
		// * delayed response
		// * dynamic response data
		// * chunked response
		for k, v := range response.Headers {
			w.Header().Set(k, v)
		}
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
