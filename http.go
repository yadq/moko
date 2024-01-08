package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
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

	server *echo.Echo
}

type httpRoute struct {
	Uri    string `yaml:"uri"`
	Method string `yaml:"method"`
	// TODO: add Request to check
	Response *httpResponse `yaml:"response"`
}

type httpResponse struct {
	Code    int               `yaml:"code"`
	Delay   int               `yaml:"delay"` // delay in milliseconds
	Headers map[string]string `yaml:"headers"`
	Body    interface{}       `yaml:"body"`
}

func newHttpServer() *HttpServer {
	server := echo.New()
	server.Use(middleware.Recover())
	server.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod:   true,
		LogStatus:   true,
		LogURI:      true,
		LogError:    true,
		HandleError: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error == nil {
				log.Infof("server %s %s, status: %d", v.Method, v.URI, v.Status)
			} else {
				log.Errorf("server %s %s, status: %d, error: %v", v.Method, v.URI, v.Status, v.Error)
			}
			return nil
		},
	}))
	server.HideBanner = true
	server.HidePort = true

	return &HttpServer{server: server}
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
		log.Infof("add mock HTTP API: %s %s\n", r.Method, r.Uri)
		switch r.Method {
		case "GET":
			s.server.GET(r.Uri, uriHandler(r.Response))
		case "POST":
			s.server.POST(r.Uri, uriHandler(r.Response))
		case "HEAD":
			s.server.HEAD(r.Uri, uriHandler(r.Response))
		case "DELETE":
			s.server.DELETE(r.Uri, uriHandler(r.Response))
		case "PUT":
			s.server.PUT(r.Uri, uriHandler(r.Response))
		case "PATCH":
			s.server.PATCH(r.Uri, uriHandler(r.Response))
		case "OPTIONS":
			s.server.OPTIONS(r.Uri, uriHandler(r.Response))
		default:
			log.Errorf("Unsupported method %v", r.Method)
		}
	}
}

func uriHandler(response *httpResponse) func(echo.Context) error {
	return func(c echo.Context) error {
		params, err := getRequestParams(c)
		header := c.Response().Header()
		if err != nil {
			// NOTE: write error to header
			errMsg := fmt.Sprintf("read request params error: %v", err)
			header.Set("Moko-Error", errMsg)
			log.Error(errMsg)
		}

		// write response headers
		for k, v := range response.Headers {
			rk, err := renderString(k, params)
			if err != nil {
				log.Errorf("render header key %s error: %v", k, err)
				continue
			}
			rv, err := renderString(v, params)
			if err != nil {
				log.Errorf("render header value %s error: %v", v, err)
				continue
			}
			header.Set(rk, rv)
		}

		var bodyString string

		// json response
		switch reflect.TypeOf(response.Body).Kind() {
		case reflect.String: // text
			bodyString = response.Body.(string)
		default: // json
			header.Set("Content-Type", "application/json")
			jsonBytes, err := MarshalJSON(response.Body)
			if err != nil {
				log.Errorf("marshal response json error: %v", err)
				return c.String(http.StatusInternalServerError, err.Error())
			}
			bodyString = string(jsonBytes)
		}

		// handle delay
		if response.Delay > 0 {
			time.Sleep(time.Duration(response.Delay) * time.Millisecond)
		}

		// render template
		renderedBody, err := renderString(bodyString, params)
		if err != nil {
			log.Errorf("render response template error: %v", err)
			return c.String(response.Code, bodyString)
		}
		return c.String(response.Code, renderedBody)
	}
}

func getRequestParams(c echo.Context) (map[string]interface{}, error) {
	params := make(map[string]interface{})
	// render path variable
	for _, pname := range c.ParamNames() {
		params[pname] = c.Param(pname)
	}
	// render query variable
	for pname, pvalues := range c.QueryParams() {
		if len(pvalues) == 1 {
			params[pname] = pvalues[0]
		} else {
			params[pname] = pvalues
		}
	}
	// render form variable
	formParams, err := c.FormParams()
	if err != nil {
		log.Errorf("parse form params error: %v", err)
	} else {
		for pname, pvalues := range formParams {
			if len(pvalues) == 1 {
				params[pname] = pvalues[0]
			} else {
				params[pname] = pvalues
			}
		}
	}
	// parse json data
	if err := c.Bind(&params); err != nil {
		return params, err
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
		log.Infof("with cert and key file configured, start HTTPS server on :%d", s.Port)
		return s.server.StartTLS(addr, s.CertFile, s.KeyFile)
	}

	log.Infof("start HTTP server on :%d", s.Port)
	return s.server.Start(addr)
}

func init() {
	ServerMap.Add("http", newHttpServer())
}
