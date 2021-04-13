package main

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
)

// http-mock.yaml example
//
// port: 8181
// routes:
//   - uri: /api
//     method: GET
//     response:
//       contentType: plain/text
//       data: hello world

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
	ContentType string `yaml:"contentType"`
	Data        string `yaml:"data"`
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

	// init routes
	for idx := range s.Routes {
		rt := s.Routes[idx]
		log.Printf("try to mock: %v %v\n", rt.Method, rt.Uri)
		switch rt.Method {
		case "GET":
			s.router.GET(rt.Uri, uriHandler(rt.Response))
		case "POST":
			s.router.POST(rt.Uri, uriHandler(rt.Response))
		case "HEAD":
			s.router.HEAD(rt.Uri, uriHandler(rt.Response))
		case "DELETE":
			s.router.DELETE(rt.Uri, uriHandler(rt.Response))
		case "PUT":
			s.router.PUT(rt.Uri, uriHandler(rt.Response))
		case "OPTIONS":
			s.router.OPTIONS(rt.Uri, uriHandler(rt.Response))
		default:
			log.Printf("Unknown method %v", rt.Method)
		}
	}

	return nil
}

func (s *HttpServer) Serve() error {
	log.Printf("start HTTP server on :%d\n", s.Port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.Port), s.router)
}

func uriHandler(response *httpResponse) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// TODO: handle:
		// * status code
		// * delayed response
		// * dynamic response data
		// * chunked response
		if response.ContentType != "" {
			w.Header().Set("Content-Type", response.ContentType)
		}
		fmt.Fprint(w, response.Data)
	}
}

func init() {
	ServerMap.Add("http", newHttpServer())
}
