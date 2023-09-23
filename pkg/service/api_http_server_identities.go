package service

import (
	"errors"
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/galdor/go-service/pkg/pg"
)

func (s *APIHTTPServer) setupIdentityRoutes() {
	s.route("/identities", "GET", s.hIdentitiesGET,
		HTTPRouteOptions{Project: true})
	s.route("/identities", "POST",
		s.hIdentitiesPOST,
		HTTPRouteOptions{Project: true})
	s.route("/identities/id/:id", "GET", s.hIdentitiesIdGET,
		HTTPRouteOptions{Project: true})
	s.route("/identities/id/:id", "DELETE", s.hIdentitiesIdDELETE,
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

func (s *APIHTTPServer) hIdentitiesPOST(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	var newIdentity eventline.NewIdentity
	if err := h.JSONRequestData(&newIdentity); err != nil {
		return
	}

	identity, err := s.Service.CreateIdentity(&newIdentity, scope)
	if err != nil {
		var duplicateIdentityNameErr *DuplicateIdentityNameError

		if errors.As(err, &duplicateIdentityNameErr) {
			h.ReplyError(400, "duplicate_identity_name", "%v", err)
		} else {
			h.ReplyInternalError(500, "cannot create identity: %v", err)
		}

		return
	}

	extra := map[string]interface{}{
		"id":     identity.Id.String(),
		"status": identity.Status,
	}

	h.ReplyJSON(201, extra)
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

func (s *APIHTTPServer) hIdentitiesIdDELETE(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	identityId, err := h.IdPathVariable("id")
	if err != nil {
		return
	}

	if err := s.Service.DeleteIdentity(identityId, scope); err != nil {
		var unknownIdentityErr *eventline.UnknownIdentityError
		var identityInUseErr *IdentityInUseError

		if errors.As(err, &unknownIdentityErr) {
			h.ReplyError(404, "unknown_identity", "%v", err)
		} else if errors.As(err, &identityInUseErr) {
			h.ReplyError(400, "identity_in_use", "%v", err)
		} else {
			h.ReplyInternalError(500, "cannot delete identity: %v", err)
		}

		return
	}

	h.ReplyEmpty(204)
}
