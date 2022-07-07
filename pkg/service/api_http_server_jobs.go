package service

import (
	"errors"
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/pg"
)

func (s *APIHTTPServer) setupJobRoutes() {
	s.route("/jobs", "GET", s.hJobsGET,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}", "GET", s.hJobsIdGET,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}", "DELETE", s.hJobsIdDELETE,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}", "DELETE", s.hJobsIdDELETE,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/name/{name}", "GET", s.hJobsNameGET,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/name/{name}", "PUT", s.hJobsNamePUT,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}/enable", "POST", s.hJobsIdEnablePOST,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}/disable", "POST", s.hJobsIdDisablePOST,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}/execute", "POST", s.hJobsIdExecutePOST,
		HTTPRouteOptions{Project: true})
}

func (s *APIHTTPServer) hJobsGET(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	cursor, err := h.ParseCursor(eventline.JobSorts)
	if err != nil {
		return
	}

	var page *eventline.Page

	err = s.Pg.WithConn(func(conn pg.Conn) (err error) {
		pageOptions := eventline.JobPageOptions{}

		page, err = eventline.LoadJobPage(conn, pageOptions, cursor, scope)
		if err != nil {
			err = fmt.Errorf("cannot load jobs: %w", err)
		}
		return
	})
	if err != nil {
		h.ReplyInternalError(500, "%v", err)
		return
	}

	h.ReplyJSON(200, page)
}

func (s *APIHTTPServer) hJobsIdGET(h *HTTPHandler) {
	jobId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	job, err := s.LoadJob(h, jobId)
	if err != nil {
		return
	}

	h.ReplyJSON(200, job)
}

func (s *APIHTTPServer) hJobsIdDELETE(h *HTTPHandler) {
	jobId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	if err := s.DeleteJob(h, jobId); err != nil {
		return
	}

	h.ReplyEmpty(204)
}

func (s *APIHTTPServer) hJobsNameGET(h *HTTPHandler) {
	jobName := h.RouteVariable("name")

	job, err := s.LoadJobByName(h, jobName)
	if err != nil {
		return
	}

	h.ReplyJSON(200, job)
}

func (s *APIHTTPServer) hJobsNamePUT(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	var spec eventline.JobSpec
	if err := h.JSONRequestData(&spec); err != nil {
		return
	}

	dryRun := h.HasQueryParameter("dry-run")

	var job *eventline.Job
	var subscriptionCreatedOrUpdated bool

	err := s.Service.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		if err := s.Service.ValidateJobSpec(conn, &spec, scope); err != nil {
			return fmt.Errorf("invalid job: %w", err)
		}

		if dryRun {
			return nil
		}

		var err error
		job, subscriptionCreatedOrUpdated, err =
			s.Service.CreateOrUpdateJob(conn, &spec, scope)
		if err != nil {
			return fmt.Errorf("cannot create or update job: %w", err)
		}

		return nil
	})
	if err != nil {
		var validationErrors check.ValidationErrors

		if errors.As(err, &validationErrors) {
			h.ReplyRequestBodyValidationErrors(validationErrors)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return
	}

	if dryRun {
		h.ReplyEmpty(204)
		return
	}

	if subscriptionCreatedOrUpdated {
		if w := s.Service.FindWorker("subscription-worker"); w != nil {
			w.WakeUp()
		}
	}

	h.ReplyJSON(200, job)
}

func (s *APIHTTPServer) hJobsIdEnablePOST(h *HTTPHandler) {
	jobId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	if err := s.EnableJob(h, jobId); err != nil {
		return
	}

	h.ReplyEmpty(204)
}

func (s *APIHTTPServer) hJobsIdDisablePOST(h *HTTPHandler) {
	jobId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	if err := s.DisableJob(h, jobId); err != nil {
		return
	}

	h.ReplyEmpty(204)
}

func (s *APIHTTPServer) hJobsIdExecutePOST(h *HTTPHandler) {
	jobId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	var input eventline.JobExecutionInput
	if err := h.JSONRequestData(&input); err != nil {
		return
	}

	jobExecution, err := s.ExecuteJob(h, jobId, &input)
	if err != nil {
		return
	}

	h.ReplyJSON(200, jobExecution)
}
