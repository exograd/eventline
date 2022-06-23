package service

import (
	"errors"
	"fmt"

	"github.com/exograd/evgo/pkg/eventline"
	"github.com/exograd/go-daemon/pg"
)

func (s *APIHTTPServer) setupEventRoutes() {
	s.route("/events", "GET",
		s.hEventsGET,
		HTTPRouteOptions{Project: true})

	s.route("/events/id/{id}", "GET",
		s.hEventsIdGET,
		HTTPRouteOptions{Project: true})

	s.route("/events/id/{id}/replay", "POST",
		s.hEventsIdReplayPOST,
		HTTPRouteOptions{Project: true})
}

func (s *APIHTTPServer) hEventsGET(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	cursor, err := h.ParseCursor(eventline.EventSorts)
	if err != nil {
		return
	}

	var page *eventline.Page

	err = s.Pg.WithConn(func(conn pg.Conn) (err error) {
		page, err = eventline.LoadEventPage(conn, cursor, scope)
		if err != nil {
			err = fmt.Errorf("cannot load events: %w", err)
		}
		return
	})
	if err != nil {
		h.ReplyInternalError(500, "%v", err)
		return
	}

	h.ReplyJSON(200, page)
}

func (s *APIHTTPServer) hEventsIdGET(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	eventId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	var event eventline.Event

	err = s.Pg.WithConn(func(conn pg.Conn) (err error) {
		if err = event.Load(conn, eventId, scope); err != nil {
			err = fmt.Errorf("cannot load event: %w", err)
		}
		return
	})
	if err != nil {
		var unknownEventErr *eventline.UnknownEventError

		if errors.As(err, &unknownEventErr) {
			h.ReplyError(404, "unknown_event", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return
	}

	h.ReplyJSON(200, &event)
}

func (s *APIHTTPServer) hEventsIdReplayPOST(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	eventId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	event, err := s.Service.ReplayEvent(eventId, scope)
	if err != nil {
		var unknownEventErr *eventline.UnknownEventError

		if errors.As(err, &unknownEventErr) {
			h.ReplyError(404, "unknown_event", "%v", err)
		} else {
			h.ReplyInternalError(500, "cannot replay event: %v", err)
		}

		return
	}

	h.ReplyJSON(200, event)
}
