package service

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
