package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strings"
	"text/template"
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
			if err != nil {
				bodyString = string(jsonBytes)
				log.Printf("marshal response json error: %v\n", err)
				fmt.Fprint(w, err.Error())
				return
			}
			bodyString = string(jsonBytes)
		}

		// write status code
		w.WriteHeader(response.Code)
		// render template
		renderedBody, err := renderResponseBody(bodyString, r, ps)
		if err != nil {
			log.Printf("render response template error: %v\n", err)
			fmt.Fprint(w, bodyString)
			return
		}
		fmt.Fprint(w, renderedBody)
	}
}

func renderResponseBody(body string, r *http.Request, ps httprouter.Params) (string, error) {
	if !strings.ContainsRune(body, '{') {
		return body, nil
	}

	params := make(map[string]interface{})
	// render path variable
	for _, p := range ps {
		params[p.Key] = p.Value
	}
	// parse URL query and form-data
	if err := r.ParseForm(); err != nil {
		return body, err
	}
	for qk := range r.Form {
		params[qk] = r.Form.Get(qk)
	}
	// parse json data
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			return body, err
		}
	}
	// render template
	tpl, err := template.New("response").Parse(body)
	if err != nil {
		return body, err
	}
	buf := bytes.NewBuffer(nil)
	if err := tpl.Execute(buf, params); err != nil {
		return body, err
	}

	return buf.String(), nil
}

func (s *HttpServer) Serve() error {
	log.Printf("start HTTP server on :%d\n", s.Port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.Port), s.router)
}

func init() {
	ServerMap.Add("http", newHttpServer())
}
