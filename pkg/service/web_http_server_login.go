package service

import (
	"encoding/base64"

	"github.com/exograd/eventline/pkg/web"
)

func (s *WebHTTPServer) setupLoginRoutes() {
	s.route("/login", "GET",
		s.hLoginGET,
		HTTPRouteOptions{Public: true})

	s.route("/login", "POST",
		s.hLoginPOST,
		HTTPRouteOptions{Public: true})

	s.route("/logout", "POST",
		s.hLogoutPOST,
		HTTPRouteOptions{})
}

func (s *WebHTTPServer) hLoginGET(h *HTTPHandler) {
	// If we already are authenticated, we may as well redirect to the
	// required page.
	if h.Context.Session != nil {
		h.ReplyRedirect(302, h.RedirectionTarget())
		return
	}

	encodedErrorMessage := h.QueryParameter("error_message")
	errorMessageData, err :=
		base64.URLEncoding.DecodeString(encodedErrorMessage)
	var errorMessage string
	if err == nil {
		errorMessage = string(errorMessageData)
	}

	breadcrumb := web.NewBreadcrumb()
	breadcrumb.AddEntry(&web.BreadcrumbEntry{
		Label: "Login",
		URI:   "/login",
	})

	bodyData := struct {
		ErrorMessage string
	}{
		ErrorMessage: errorMessage,
	}

	h.ReplyView(200, &web.View{
		Title:      "Login",
		Menu:       NewLoginMenu("login"),
		Breadcrumb: breadcrumb,
		Body:       s.NewTemplate("login.html", bodyData),
	})
}

func (s *WebHTTPServer) hLoginPOST(h *HTTPHandler) {
	var loginData LoginData
	if err := h.JSONRequestData(&loginData); err != nil {
		return
	}

	session, err := s.LogIn(h, &loginData)
	if err != nil {
		return
	}

	h.SetSessionCookie(s.Service.sessionCookie(session.Id))

	h.ReplyJSONLocation(200, h.RedirectionTarget(), nil)
}

func (s *WebHTTPServer) hLogoutPOST(h *HTTPHandler) {
	if h.Context.Session == nil {
		h.ReplyJSONLocation(200, "/", nil)
		return
	}

	if err := s.Service.LogOut(h.Context); err != nil {
		h.ReplyInternalError(500, "cannot log out: %v", err)
		return
	}

	h.SetSessionCookie(s.Service.expiredCookie())

	h.ReplyJSONLocation(201, "/login", nil)
}
