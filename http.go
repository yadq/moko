package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/julienschmidt/httprouter"
	"gopkg.in/yaml.v2"
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
	Routes   []*httpRoute `yaml:"routes"`
	Port     int          `yaml:"port"`
	CertFile string       `yaml:"cert"`
	KeyFile  string       `yaml:"key"`

	router *httprouter.Router
}

type httpRoute struct {
	Uri      string        `yaml:"uri"`
	Method   string        `yaml:"method"`
	Response *httpResponse `yaml:"response"`
}

type httpResponse struct {
	Code    int               `yaml:"code"`
	Delay   int               `yaml:"delay"` // delay in milliseconds
	Headers map[string]string `yaml:"headers"`
	Body    interface{}       `yaml:"body"`
}

func newHttpServer() *HttpServer {
	return &HttpServer{router: httprouter.New()}
}

func (s *HttpServer) Init(cfgFile string) error {
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, s); err != nil {
		return err
	}

	if s.CertFile != "" {
		if _, err := os.Stat(s.CertFile); os.IsNotExist(err) {
			return err
		}
	}

	if s.KeyFile != "" {
		if _, err := os.Stat(s.KeyFile); os.IsNotExist(err) {
			return err
		}
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
		log.Printf("add mock HTTP API: %v %v\n", r.Method, r.Uri)
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
		case "PATCH":
			s.router.PATCH(r.Uri, uriHandler(r.Response))
		case "OPTIONS":
			s.router.OPTIONS(r.Uri, uriHandler(r.Response))
		default:
			log.Printf("Unsupported method %v", r.Method)
		}
	}
}

func uriHandler(response *httpResponse) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		params, err := getRequestParams(r, ps)
		if err != nil {
			// XXX: no return but log error
			log.Printf("read request params error: %v\n", err)
		}

		// write response headers
		for k, v := range response.Headers {
			rk, err := renderString(k, params)
			if err != nil {
				log.Printf("render header key %s error: %v\n", k, err)
				continue
			}
			rv, err := renderString(v, params)
			if err != nil {
				log.Printf("render header value %s error: %v\n", v, err)
				continue
			}
			w.Header().Set(rk, rv)
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
				log.Printf("marshal response json error: %v\n", err)
				fmt.Fprint(w, err.Error())
				return
			}
			bodyString = string(jsonBytes)
		}

		// handle delay
		if response.Delay > 0 {
			time.Sleep(time.Duration(response.Delay) * time.Millisecond)
		}

		// write status code
		w.WriteHeader(response.Code)
		// render template
		renderedBody, err := renderString(bodyString, params)
		if err != nil {
			log.Printf("render response template error: %v\n", err)
			fmt.Fprint(w, bodyString)
			return
		}
		fmt.Fprint(w, renderedBody)
	}
}

func getRequestParams(r *http.Request, ps httprouter.Params) (map[string]interface{}, error) {
	params := make(map[string]interface{})
	// render path variable
	for _, p := range ps {
		params[p.Key] = p.Value
	}
	// parse URL query and form-data
	if err := r.ParseForm(); err != nil {
		return params, err
	}
	for qk := range r.Form {
		params[qk] = r.Form.Get(qk)
	}
	// parse json data
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			return params, err
		}
	}

	return params, nil
}

var tplPattern = regexp.MustCompile(`\$\{([^${}]+)\}`) // match ${}

func renderString(body string, params map[string]interface{}) (string, error) {
	if !strings.ContainsRune(body, '{') || !strings.ContainsRune(body, '}') {
		return body, nil
	}
	// render template
	// replace ${var} => {{.var}}
	tpl, err := template.New("response").Parse(tplPattern.ReplaceAllString(body, `{{.$1}}`))
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
	addr := fmt.Sprintf(":%d", s.Port)

	if s.CertFile != "" && s.KeyFile != "" {
		log.Printf("with cert and key file configured, start HTTPS server on :%d\n", s.Port)
		return http.ListenAndServeTLS(addr, s.CertFile, s.KeyFile, s.router)
	}

	log.Printf("start HTTP server on :%d\n", s.Port)
	return http.ListenAndServe(addr, s.router)
}

func init() {
	ServerMap.Add("http", newHttpServer())
}
