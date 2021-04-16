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
		log.Printf("add mock API: %v %v\n", r.Method, r.Uri)
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
			log.Printf("Unsupported method %v", r.Method)
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

		var bodyString string

		// json response
		switch reflect.TypeOf(response.Body).Kind() {
		case reflect.String: // text
			bodyString = response.Body.(string)
		default: // json
			w.Header().Set("Content-Type", "application/json")
			jsonBytes, err := MarshalJSON(response.Body)
			if err == nil {
				bodyString = string(jsonBytes)
			} else {
				bodyString = err.Error()
			}
		}

		// write status code
		w.WriteHeader(response.Code)
		// render template: ${...} => {...}, then render as go template
		if strings.ContainsRune(bodyString, '$') {
			// render path variable
			for _, p := range ps {
				bodyString = strings.ReplaceAll(bodyString, fmt.Sprintf("${%s}", p.Key), p.Value)
			}
			// render URL query and form-data
			if err := r.ParseForm(); err == nil {
				for qk := range r.Form {
					bodyString = strings.ReplaceAll(bodyString, fmt.Sprintf("${%s}", qk), r.Form.Get(qk))
				}
			}
			// render json data
			if r.Header.Get("Content-Type") == "application/json" {
				// TODO
			}
		}
		fmt.Fprint(w, bodyString)
	}
}

func (s *HttpServer) Serve() error {
	log.Printf("start HTTP server on :%d\n", s.Port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.Port), s.router)
}

func init() {
	ServerMap.Add("http", newHttpServer())
}
