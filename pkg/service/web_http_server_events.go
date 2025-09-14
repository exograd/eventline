package service

import (
	"errors"
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/utils"
	"github.com/exograd/eventline/pkg/web"
	"go.n16f.net/service/pkg/pg"
	"go.n16f.net/uuid"
)

func (s *WebHTTPServer) setupEventRoutes() {
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

func (s *WebHTTPServer) hEventsGET(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	cursor, err := h.ParseCursor(eventline.EventSorts)
	if err != nil {
		return
	}
	if cursor.Order == "" {
		cursor.Order = eventline.OrderDesc
	}

	var page *eventline.Page
	var jobNames map[uuid.UUID]string

	err = s.Pg.WithConn(func(conn pg.Conn) (err error) {
		page, err = eventline.LoadEventPage(conn, cursor, scope)
		if err != nil {
			err = fmt.Errorf("cannot load events: %w", err)
			return
		}

		var jobIds []uuid.UUID
		for _, element := range page.Elements {
			event := element.(*eventline.Event)
			jobIds = append(jobIds, event.JobId)
		}

		jobNames, err = eventline.LoadJobNamesById(conn, jobIds)
		if err != nil {
			err = fmt.Errorf("cannot load job names: %w", err)
			return
		}

		return
	})
	if err != nil {
		h.ReplyInternalError(500, "%v", err)
		return
	}

	bodyData := struct {
		Page     *eventline.Page
		JobNames map[uuid.UUID]string
	}{
		Page:     page,
		JobNames: jobNames,
	}

	h.ReplyView(200, &web.View{
		Title:      "Events",
		Menu:       NewMainMenu("events"),
		Breadcrumb: eventsBreadcrumb(),
		Body:       s.NewTemplate("events.html", bodyData),
	})
}

func (s *WebHTTPServer) hEventsIdGET(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	eventId, err := h.IdPathVariable("id")
	if err != nil {
		return
	}

	var event eventline.Event
	var job *eventline.Job
	var jobExecutions eventline.JobExecutions

	err = s.Pg.WithConn(func(conn pg.Conn) error {
		if err := event.Load(conn, eventId, scope); err != nil {
			return fmt.Errorf("cannot load event: %w", err)
		}

		job = new(eventline.Job)
		if err := job.Load(conn, event.JobId, scope); err != nil {
			return fmt.Errorf("cannot load job: %w", err)
		}

		if err := jobExecutions.LoadByEvent(conn, eventId); err != nil {
			return fmt.Errorf("cannot load job executions: %w", err)
		}

		return nil
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

	edata, err := utils.JSONEncode(event.Data)
	if err != nil {
		h.ReplyInternalError(500, "cannot encode event data: %v", err)
		return
	}

	bodyData := struct {
		Event         *eventline.Event
		EventData     string
		Job           *eventline.Job
		JobExecutions eventline.JobExecutions
	}{
		Event:         &event,
		EventData:     string(edata),
		Job:           job,
		JobExecutions: jobExecutions,
	}

	h.ReplyView(200, &web.View{
		Title:      "Event",
		Menu:       NewMainMenu("events"),
		Breadcrumb: eventBreadcrumb(&event),
		Body:       s.NewTemplate("event_view.html", bodyData),
	})
}

func (s *WebHTTPServer) hEventsIdReplayPOST(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	eventId, err := h.IdPathVariable("id")
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

	location := "/events/id/" + event.Id.String()

	h.ReplyJSONLocation(200, location, nil)
}

func eventsBreadcrumb() *web.Breadcrumb {
	breadcrumb := web.NewBreadcrumb()

	breadcrumb.AddEntry(&web.BreadcrumbEntry{
		Label: "Events",
		URI:   "/events",
	})

	return breadcrumb
}

func eventBreadcrumb(event *eventline.Event) *web.Breadcrumb {
	breadcrumb := eventsBreadcrumb()

	breadcrumb.AddEntry(&web.BreadcrumbEntry{
		Label:    event.Id.String(),
		Verbatim: true,
		URI:      "/events/id/" + event.Id.String(),
	})

	return breadcrumb
}
