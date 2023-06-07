package service

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/galdor/go-ejson"
	"github.com/galdor/go-service/pkg/pg"
)

func (s *APIHTTPServer) setupJobRoutes() {
	s.route("/jobs", "GET", s.hJobsGET,
		HTTPRouteOptions{Project: true})

	s.route("/jobs", "PUT", s.hJobsPUT,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/:id", "GET", s.hJobsIdGET,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/:id", "DELETE", s.hJobsIdDELETE,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/name/:name", "GET", s.hJobsNameGET,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/name/:name", "PUT", s.hJobsNamePUT,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/:id/rename", "POST", s.hJobsIdRenamePOST,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/:id/enable", "POST", s.hJobsIdEnablePOST,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/:id/disable", "POST", s.hJobsIdDisablePOST,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/:id/execute", "POST", s.hJobsIdExecutePOST,
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

func (s *APIHTTPServer) hJobsPUT(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	var specs eventline.JobSpecs
	if err := h.JSONRequestData(&specs); err != nil {
		return
	}

	dryRun := h.HasQueryParameter("dry-run")

	jobs := make(eventline.Jobs, len(specs))
	var subscriptionsCreatedOrUpdated bool

	err := s.Service.Pg.WithTx(func(conn pg.Conn) error {
		id1 := PgAdvisoryLockId1
		id2 := PgAdvisoryLockId2JobDeployment

		if err := pg.TakeAdvisoryTxLock(conn, id1, id2); err != nil {
			return fmt.Errorf("cannot take advisory lock: %w", err)
		}

		var validationErrors ejson.ValidationErrors

		for i, spec := range specs {
			if err := s.Service.ValidateJobSpec(conn, spec, scope); err != nil {
				var verrs ejson.ValidationErrors

				if errors.As(err, &verrs) {
					for _, verr := range verrs {
						verr.Pointer.Prepend(strconv.Itoa(i))
					}

					validationErrors = append(validationErrors, verrs...)
				} else {
					return fmt.Errorf("invalid job specification %d: %w",
						i+1, err)
				}
			}
		}

		if validationErrors != nil {
			return fmt.Errorf("invalid job specifications: %w",
				validationErrors)
		}

		if dryRun {
			return nil
		}

		for i, spec := range specs {
			var err error
			job, subscriptionCreatedOrUpdated, err :=
				s.Service.CreateOrUpdateJob(conn, spec, scope)
			if err != nil {
				return fmt.Errorf("cannot create or update job: %w", err)
			}

			jobs[i] = job

			if subscriptionCreatedOrUpdated {
				subscriptionsCreatedOrUpdated = true
			}
		}

		return nil
	})
	if err != nil {
		var validationErrors ejson.ValidationErrors

		if errors.As(err, &validationErrors) {
			h.ReplyValidationErrors(validationErrors)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return
	}

	if dryRun {
		h.ReplyEmpty(204)
		return
	}

	if subscriptionsCreatedOrUpdated {
		if w := s.Service.FindWorker("subscription-worker"); w != nil {
			w.WakeUp()
		}
	}

	h.ReplyJSON(200, jobs)
}

func (s *APIHTTPServer) hJobsIdGET(h *HTTPHandler) {
	jobId, err := h.IdPathVariable("id")
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
	jobId, err := h.IdPathVariable("id")
	if err != nil {
		return
	}

	if err := s.DeleteJob(h, jobId); err != nil {
		return
	}

	h.ReplyEmpty(204)
}

func (s *APIHTTPServer) hJobsNameGET(h *HTTPHandler) {
	jobName := h.PathVariable("name")

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

	err := s.Service.Pg.WithTx(func(conn pg.Conn) error {
		id1 := PgAdvisoryLockId1
		id2 := PgAdvisoryLockId2JobDeployment

		if err := pg.TakeAdvisoryTxLock(conn, id1, id2); err != nil {
			return fmt.Errorf("cannot take advisory lock: %w", err)
		}

		if err := s.Service.ValidateJobSpec(conn, &spec, scope); err != nil {
			return fmt.Errorf("invalid job specification: %w", err)
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
		var validationErrors ejson.ValidationErrors

		if errors.As(err, &validationErrors) {
			h.ReplyValidationErrors(validationErrors)
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

func (s *APIHTTPServer) hJobsIdRenamePOST(h *HTTPHandler) {
	jobId, err := h.IdPathVariable("id")
	if err != nil {
		return
	}

	var data eventline.JobRenamingData
	if err := h.JSONRequestData(&data); err != nil {
		return
	}

	if err := s.RenameJob(h, jobId, &data); err != nil {
		return
	}

	h.ReplyEmpty(204)
}

func (s *APIHTTPServer) hJobsIdEnablePOST(h *HTTPHandler) {
	jobId, err := h.IdPathVariable("id")
	if err != nil {
		return
	}

	if err := s.EnableJob(h, jobId); err != nil {
		return
	}

	h.ReplyEmpty(204)
}

func (s *APIHTTPServer) hJobsIdDisablePOST(h *HTTPHandler) {
	jobId, err := h.IdPathVariable("id")
	if err != nil {
		return
	}

	if err := s.DisableJob(h, jobId); err != nil {
		return
	}

	h.ReplyEmpty(204)
}

func (s *APIHTTPServer) hJobsIdExecutePOST(h *HTTPHandler) {
	jobId, err := h.IdPathVariable("id")
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
