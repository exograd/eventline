package service

import "github.com/exograd/go-daemon/dhttp"

type APIHTTPServer struct {
	*HTTPServer
}

func NewAPIHTTPServer(service *Service) (*APIHTTPServer, error) {
	s := &APIHTTPServer{
		HTTPServer: &HTTPServer{
			Log: service.Log.Child("api", nil),

			Pg: service.Daemon.Pg,

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
	s.Server = s.Service.Daemon.HTTPServers["api"]

	s.route("/status", "HEAD", s.hStatusHEAD,
		HTTPRouteOptions{Public: true})

	s.setupAccountRoutes()
	s.setupLoginRoute()
	s.setupProjectRoutes()
	s.setupJobRoutes()
	s.setupEventRoutes()
}

func (s *APIHTTPServer) hStatusHEAD(h *HTTPHandler) {
	h.ReplyEmpty(204)
}

func (s *Service) apiHTTPErrorHandler(dh *dhttp.Handler, status int, code, msg string, data dhttp.APIErrorData) {
	dh.ReplyJSON(status, dhttp.APIError{
		Message: msg,
		Code:    code,
		Data:    data,
	})
}
