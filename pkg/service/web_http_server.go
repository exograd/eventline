package service

import (
	"github.com/exograd/eventline/pkg/web"
	"github.com/galdor/go-service/pkg/shttp"
)

type WebHTTPServer struct {
	*HTTPServer
}

func NewWebHTTPServer(service *Service) (*WebHTTPServer, error) {
	s := &WebHTTPServer{
		HTTPServer: &HTTPServer{
			Log: service.Log.Child("web", nil),

			Pg: service.Pg,

			Service: service,
		},
	}

	s.initHTTPServer()

	return s, nil
}

func (s *WebHTTPServer) NewTemplate(name string, data interface{}) *web.Template {
	return web.NewTemplate(s.Service.Service.HTMLTemplate, name, data)
}

func (s *WebHTTPServer) route(path, method string, fn HTTPRouteFunc, options HTTPRouteOptions) {
	s.Server.Route(path, method, s.Service.WrapRoute(fn, options,
		WebHTTPInterface))
}

func (s *WebHTTPServer) initHTTPServer() {
	s.Server = s.Service.Service.HTTPServer("web")

	s.route("/", "GET", s.hGET,
		HTTPRouteOptions{})

	s.route("/status", "HEAD", s.hStatusHEAD,
		HTTPRouteOptions{Public: true}) // TODO do not log

	s.setupAssetRoutes()
	s.setupLoginRoutes()
	s.setupAccountRoutes()
	s.setupAdminRoutes()
	s.setupProjectRoutes()
	s.setupIdentityRoutes()
	s.setupJobRoutes()
	s.setupJobExecutionRoutes()
	s.setupStepExecutionRoutes()
	s.setupEventRoutes()
	s.setupExternalRoutes()
}

func (s *WebHTTPServer) hGET(h *HTTPHandler) {
	if h.Context.ProjectId == nil {
		h.ReplyRedirect(302, "/projects")
	} else {
		h.ReplyRedirect(302, "/jobs")
	}
}

func (s *WebHTTPServer) hStatusHEAD(h *HTTPHandler) {
	h.ReplyEmpty(204)
}

func (s *Service) webErrorHandler(handler *shttp.Handler, status int, code, msg string, data shttp.ErrorData) {
	if shttp.RequestAcceptsText(handler.Request) {
		// In this handler, we have access to the go-service handler. If the
		// error is happening before entering one of our routes, we do not
		// have a HTTPHandler, and replying with a page is a bit more
		// complicated. We handle both cases.

		var h *HTTPHandler

		ctx := handler.Request.Context()
		if v := ctx.Value(contextKeyHandler); v != nil {
			h = v.(*HTTPHandler)
		}

		if h == nil {
			h = &HTTPHandler{
				Handler: handler,

				Service:      s,
				RouteOptions: HTTPRouteOptions{},
				Context:      &HTTPContext{},
			}
		}

		errData := web.ErrorData{
			Message: msg,
		}

		h.ReplyView(status, &web.View{
			Title: "Error",
			Body:  s.WebHTTPServer.NewTemplate("error.html", &errData),
			Menu:  NewMainMenu(""),
		})
	} else {
		shttp.JSONErrorHandler(handler, status, code, msg, data)
	}
}
