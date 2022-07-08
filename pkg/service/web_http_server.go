package service

import (
	"fmt"
	"html/template"
	"path"
	"strings"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/web"
	"github.com/exograd/go-daemon/dhttp"
)

type WebHTTPServer struct {
	*HTTPServer

	HTMLTemplate *template.Template
}

func NewWebHTTPServer(service *Service) (*WebHTTPServer, error) {
	s := &WebHTTPServer{
		HTTPServer: &HTTPServer{
			Log: service.Log.Child("web", nil),

			Pg: service.Daemon.Pg,

			Service: service,
		},
	}

	if err := s.initHTMLTemplate(); err != nil {
		return nil, err
	}

	s.initHTTPServer()

	return s, nil
}

func (s *WebHTTPServer) initHTMLTemplate() error {
	dirPath := path.Join(s.Service.Cfg.DataDirectory, "templates")

	template, err := eventline.LoadHTMLTemplates(dirPath)
	if err != nil {
		return fmt.Errorf("cannot load html templates: %w", err)
	}

	s.HTMLTemplate = template

	return nil
}

func (s *WebHTTPServer) NewTemplate(name string, data interface{}) *web.Template {
	return web.NewTemplate(s.HTMLTemplate, name, data)
}

func (s *WebHTTPServer) route(path, method string, fn HTTPRouteFunc, options HTTPRouteOptions) {
	s.Server.Route(path, method, s.Service.WrapRoute(fn, options,
		WebHTTPInterface))
}

func (s *WebHTTPServer) initHTTPServer() {
	s.Server = s.Service.Daemon.HTTPServers["web"]

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

func (s *Service) webHTTPErrorHandler(dh *dhttp.Handler, status int, code, msg string, data dhttp.APIErrorData) {
	// In this handler, we have access to the go-daemon handler. If the error
	// is happening before entering one of our routes, we do not have a
	// HTTPHandler, and replying with a page is a bit more complicated. We
	// handle both cases.

	var h *HTTPHandler

	ctx := dh.Request.Context()
	v := ctx.Value(contextKeyHandler)
	if v != nil {
		h = v.(*HTTPHandler)
	}

	// Check that the client actually asked for HTML content. Fetch requests
	// sent from Javascript code expect JSON.
	//
	// TODO Proper media type negociation (add it to go-daemon).
	acceptHTML := false
	accept := dh.Request.Header.Get("Accept")
	if accept != "" {
		mediaTypes := strings.Split(accept, ",")
		for _, mediaType := range mediaTypes {
			mediaType = strings.TrimSpace(mediaType)
			if strings.HasPrefix(mediaType, "text/html") {
				acceptHTML = true
				break
			} else if strings.HasPrefix(mediaType, "text/*") {
				acceptHTML = true
				break
			}
		}
	}

	if !acceptHTML {
		dh.ReplyJSON(status, dhttp.APIError{
			Message: msg,
			Code:    code,
			Data:    data,
		})

		return
	}

	// At this point, if we do not have a HTTPHandler, we have to create one
	// to be able to reply with an error page.
	if h == nil {
		h = &HTTPHandler{
			Handler: dh,

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
}
