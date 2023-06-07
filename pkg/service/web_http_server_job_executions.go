package service

import (
	"errors"
	"fmt"
	"html/template"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/web"
	"github.com/galdor/go-service/pkg/pg"
)

func (s *WebHTTPServer) setupJobExecutionRoutes() {
	s.route("/job_executions/id/:id", "GET",
		s.hJobExecutionsIdGET,
		HTTPRouteOptions{Project: true})

	s.route("/job_executions/id/:id/content", "GET",
		s.hJobExecutionsIdContentGET,
		HTTPRouteOptions{Project: true})

	s.route("/job_executions/id/:id/abort", "POST",
		s.hJobExecutionsIdAbortPOST,
		HTTPRouteOptions{Project: true})

	s.route("/job_executions/id/:id/restart", "POST",
		s.hJobExecutionsIdRestartPOST,
		HTTPRouteOptions{Project: true})
}

func (s *WebHTTPServer) hJobExecutionsIdGET(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	jeId, err := h.IdPathVariable("id")
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

	jeId, err := h.IdPathVariable("id")
	if err != nil {
		return
	}

	var job eventline.Job
	var jobExecution eventline.JobExecution
	var stepExecutions eventline.StepExecutions
	var stepExecutionOutputs []template.HTML
	var event *eventline.Event

	err = s.Pg.WithConn(func(conn pg.Conn) error {
		if err := jobExecution.Load(conn, jeId, scope); err != nil {
			return fmt.Errorf("cannot load job execution: %w", err)
		}

		err := job.Load(conn, jobExecution.JobId, scope)
		if err != nil {
			return fmt.Errorf("cannot load job: %w", err)
		}

		err = stepExecutions.LoadByJobExecutionIdWithTruncatedOutput(conn,
			jeId, 1_000_000, "\n[truncated]\n")
		if err != nil {
			return fmt.Errorf("cannot load step executions: %w", err)
		}

		stepExecutionOutputs = make([]template.HTML, len(stepExecutions))
		for i, se := range stepExecutions {
			rawOutput := se.Output

			htmlOutput, err := eventline.RenderTermData(rawOutput)
			if err != nil {
				h.Log.Error("cannot render output of step execution %q: %v",
					se.Id, err)
				htmlOutput = rawOutput
			}

			stepExecutionOutputs[i] = template.HTML(htmlOutput)
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
		JobExecution         *eventline.JobExecution
		StepExecutions       eventline.StepExecutions
		StepExecutionOutputs []template.HTML
		Event                *eventline.Event
	}{
		JobExecution:         &jobExecution,
		StepExecutions:       stepExecutions,
		StepExecutionOutputs: stepExecutionOutputs,
		Event:                event,
	}

	content := s.NewTemplate("job_execution_view_content.html", contentData)

	h.ReplyContent(200, content)
}

func (s *WebHTTPServer) hJobExecutionsIdAbortPOST(h *HTTPHandler) {
	jeId, err := h.IdPathVariable("id")
	if err != nil {
		return
	}

	if err := s.AbortJobExecution(h, jeId); err != nil {
		return
	}

	h.ReplyEmpty(204)
}

func (s *WebHTTPServer) hJobExecutionsIdRestartPOST(h *HTTPHandler) {
	jeId, err := h.IdPathVariable("id")
	if err != nil {
		return
	}

	if err := s.RestartJobExecution(h, jeId); err != nil {
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
