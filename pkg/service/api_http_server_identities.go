package service

import (
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/galdor/go-service/pkg/pg"
)

func (s *APIHTTPServer) setupIdentityRoutes() {
	s.route("/identities", "GET", s.hIdentitiesGET,
		HTTPRouteOptions{Project: true})
	s.route("/identities/id/:id", "GET", s.hIdentitiesIdGET,
		HTTPRouteOptions{Project: true})
}

func (s *APIHTTPServer) hIdentitiesGET(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	cursor, err := h.ParseCursor(eventline.IdentitySorts)
	if err != nil {
		return
	}

	var page *eventline.Page

	err = s.Pg.WithConn(func(conn pg.Conn) (err error) {
		page, err = eventline.LoadIdentityPage(conn, cursor, scope)
		if err != nil {
			err = fmt.Errorf("cannot load identities: %w", err)
		}
		return
	})
	if err != nil {
		h.ReplyInternalError(500, "%v", err)
		return
	}

	h.ReplyJSON(200, page)
}

func (s *APIHTTPServer) hIdentitiesIdGET(h *HTTPHandler) {
	identityId, err := h.IdPathVariable("id")
	if err != nil {
		return
	}

	identity, err := s.LoadIdentity(h, identityId)
	if err != nil {
		return
	}

	h.ReplyJSON(200, identity)
}
