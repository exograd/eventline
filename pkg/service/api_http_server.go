package service

import (
	"encoding/json"

	"github.com/galdor/go-service/pkg/shttp"
)

type APIError struct {
	Message string          `json:"error"`
	Code    string          `json:"code,omitempty"`
	RawData json.RawMessage `json:"data,omitempty"`
	Data    shttp.ErrorData `json:"-"`
}

type APIHTTPServer struct {
	*HTTPServer
}

func NewAPIHTTPServer(service *Service) (*APIHTTPServer, error) {
	s := &APIHTTPServer{
		HTTPServer: &HTTPServer{
			Log: service.Log.Child("api", nil),

			Pg: service.Pg,

			Service: service,
		},
	}

	s.initHTTPServer()

	return s, nil
}

func (s *APIHTTPServer) route(path, method string, fn HTTPRouteFunc, options HTTPRouteOptions) {
	s.Server.Route(path, method, s.Service.WrapRoute(fn, options,
		APIHTTPInterface))
}

func (s *APIHTTPServer) initHTTPServer() {
	s.Server = s.Service.Service.HTTPServer("api")

	s.route("/status", "HEAD", s.hStatusHEAD,
		HTTPRouteOptions{Public: true})

	s.setupAccountRoutes()
	s.setupLoginRoute()
	s.setupProjectRoutes()
	s.setupIdentityRoutes()
	s.setupJobRoutes()
	s.setupJobExecutionRoutes()
	s.setupEventRoutes()
}

func (s *APIHTTPServer) hStatusHEAD(h *HTTPHandler) {
	h.ReplyEmpty(204)
}

func (s *Service) jsonErrorHandler(h *shttp.Handler, status int, code string, msg string, data shttp.ErrorData) {
	responseData := APIError{
		Message: msg,
		Code:    code,
		Data:    data,
	}

	h.ReplyJSON(status, &responseData)
}
