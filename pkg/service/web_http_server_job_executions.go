package service

import (
	"errors"
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/web"
	"github.com/exograd/go-daemon/pg"
)

func (s *WebHTTPServer) setupJobExecutionRoutes() {
	s.route("/job_executions/id/{id}", "GET",
		s.hJobExecutionsIdGET,
		HTTPRouteOptions{Project: true})

	s.route("/job_executions/id/{id}/content", "GET",
		s.hJobExecutionsIdContentGET,
		HTTPRouteOptions{Project: true})

	s.route("/job_executions/id/{id}/abort", "POST",
		s.hJobExecutionsIdAbortPOST,
		HTTPRouteOptions{Project: true})

	s.route("/job_executions/id/{id}/restart", "POST",
		s.hJobExecutionsIdRestartPOST,
		HTTPRouteOptions{Project: true})
}

func (s *WebHTTPServer) hJobExecutionsIdGET(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	jeId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	var job eventline.Job
	var jobExecution eventline.JobExecution

	err = s.Pg.WithConn(func(conn pg.Conn) error {
		if err := jobExecution.Load(conn, jeId, scope); err != nil {
			return fmt.Errorf("cannot load job execution: %w", err)
		}

		err := job.Load(conn, jobExecution.JobId, scope)
		if err != nil {
			return fmt.Errorf("cannot load job: %w", err)
		}

		return nil
	})
	if err != nil {
		var unknownJobExecutionErr *eventline.UnknownJobExecutionError

		if errors.As(err, &unknownJobExecutionErr) {
			h.ReplyError(404, "unknown_job_execution", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return
	}

	bodyData := struct {
		JobExecution *eventline.JobExecution
	}{
		JobExecution: &jobExecution,
	}

	h.ReplyView(200, &web.View{
		Title:      "Job execution",
		Menu:       NewMainMenu("jobs"),
		Breadcrumb: jobExecutionBreadcrumb(&job, &jobExecution),
		Body:       s.NewTemplate("job_execution_view.html", bodyData),
	})
}

func (s *WebHTTPServer) hJobExecutionsIdContentGET(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	jeId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	var job eventline.Job
	var jobExecution eventline.JobExecution
	var stepExecutions eventline.StepExecutions
	var event *eventline.Event

	err = s.Pg.WithConn(func(conn pg.Conn) error {
		if err := jobExecution.Load(conn, jeId, scope); err != nil {
			return fmt.Errorf("cannot load job execution: %w", err)
		}

		err := job.Load(conn, jobExecution.JobId, scope)
		if err != nil {
			return fmt.Errorf("cannot load job: %w", err)
		}

		if err = stepExecutions.LoadByJobExecutionId(conn, jeId); err != nil {
			return fmt.Errorf("cannot load step executions: %w", err)
		}

		if eventId := jobExecution.EventId; eventId != nil {
			event = new(eventline.Event)
			if err := event.Load(conn, *eventId, scope); err != nil {
				return fmt.Errorf("cannot load event: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		var unknownJobExecutionErr *eventline.UnknownJobExecutionError

		if errors.As(err, &unknownJobExecutionErr) {
			h.ReplyError(404, "unknown_job_execution", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return
	}

	contentData := struct {
		JobExecution   *eventline.JobExecution
		StepExecutions eventline.StepExecutions
		Event          *eventline.Event
	}{
		JobExecution:   &jobExecution,
		StepExecutions: stepExecutions,
		Event:          event,
	}

	content := s.NewTemplate("job_execution_view_content.html", contentData)

	h.ReplyContent(200, content)
}

func (s *WebHTTPServer) hJobExecutionsIdAbortPOST(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	jeId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	if _, err := s.Service.AbortJobExecution(jeId, scope); err != nil {
		var unknownJobExecutionErr *eventline.UnknownJobExecutionError
		var jobExecutionFinishedErr *eventline.JobExecutionFinishedError

		if errors.As(err, &unknownJobExecutionErr) {
			h.ReplyError(404, "unknown_job_execution", "%v", err)
		} else if errors.As(err, &jobExecutionFinishedErr) {
			h.ReplyError(400, "job_execution_finished", "%v", err)
		} else {
			h.ReplyInternalError(500, "cannot restart job execution: %v", err)
		}

		return
	}

	h.ReplyEmpty(204)
}

func (s *WebHTTPServer) hJobExecutionsIdRestartPOST(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	jeId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	if _, err := s.Service.RestartJobExecution(jeId, scope); err != nil {
		var unknownJobExecutionErr *eventline.UnknownJobExecutionError
		var jobExecutionNotFinishedErr *eventline.JobExecutionNotFinishedError

		if errors.As(err, &unknownJobExecutionErr) {
			h.ReplyError(404, "unknown_job_execution", "%v", err)
		} else if errors.As(err, &jobExecutionNotFinishedErr) {
			h.ReplyError(400, "job_execution_not_finished", "%v", err)
		} else {
			h.ReplyInternalError(500, "cannot restart job execution: %v", err)
		}

		return
	}

	h.ReplyEmpty(204)
}

func jobExecutionBreadcrumb(job *eventline.Job, jobExecution *eventline.JobExecution) *web.Breadcrumb {
	breadcrumb := jobBreadcrumb(job)

	breadcrumb.AddEntry(&web.BreadcrumbEntry{
		Label:    jobExecution.Id.String(),
		Verbatim: true,
		URI:      "/job_executions/id/" + jobExecution.Id.String(),
	})

	return breadcrumb
}
