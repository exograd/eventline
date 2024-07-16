package service

import (
	"errors"
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/ejson"
	"go.n16f.net/service/pkg/pg"
)

func (s *HTTPServer) LoadJob(h *HTTPHandler, jobId eventline.Id) (*eventline.Job, error) {
	scope := h.Context.ProjectScope()

	var job eventline.Job

	err := s.Pg.WithConn(func(conn pg.Conn) error {
		if err := job.Load(conn, jobId, scope); err != nil {
			return fmt.Errorf("cannot load job: %w", err)
		}

		return nil
	})
	if err != nil {
		var unknownJobErr *eventline.UnknownJobError

		if errors.As(err, &unknownJobErr) {
			h.ReplyError(404, "unknown_job", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return nil, err
	}

	return &job, nil
}

func (s *HTTPServer) LoadJobByName(h *HTTPHandler, jobName string) (*eventline.Job, error) {
	scope := h.Context.ProjectScope()

	var job eventline.Job

	err := s.Pg.WithConn(func(conn pg.Conn) error {
		if err := job.LoadByName(conn, jobName, scope); err != nil {
			return fmt.Errorf("cannot load job: %w", err)
		}

		return nil
	})
	if err != nil {
		var unknownJobNameErr *eventline.UnknownJobNameError

		if errors.As(err, &unknownJobNameErr) {
			h.ReplyError(404, "unknown_job", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return nil, err
	}

	return &job, nil
}

func (s *HTTPServer) DeleteJob(h *HTTPHandler, jobId eventline.Id) error {
	scope := h.Context.ProjectScope()

	var job eventline.Job

	err := s.Pg.WithTx(func(conn pg.Conn) error {
		if err := job.LoadForUpdate(conn, jobId, scope); err != nil {
			return fmt.Errorf("cannot load job: %w", err)
		}

		if err := s.Service.DeleteJob(conn, &job, scope); err != nil {
			return fmt.Errorf("cannot delete job: %w", err)
		}

		return nil
	})
	if err != nil {
		var unknownJobErr *eventline.UnknownJobError

		if errors.As(err, &unknownJobErr) {
			h.ReplyError(404, "unknown_job", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return err
	}

	return nil
}

func (s *HTTPServer) RenameJob(h *HTTPHandler, jobId eventline.Id, data *eventline.JobRenamingData) error {
	scope := h.Context.ProjectScope()

	err := s.Service.Pg.WithTx(func(conn pg.Conn) error {
		_, err := s.Service.RenameJob(conn, jobId, data, scope)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		var unknownJobErr *eventline.UnknownJobError

		if errors.As(err, &unknownJobErr) {
			h.ReplyError(404, "unknown_job", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return err
	}

	return nil
}

func (s *HTTPServer) EnableJob(h *HTTPHandler, jobId eventline.Id) error {
	scope := h.Context.ProjectScope()

	err := s.Service.Pg.WithTx(func(conn pg.Conn) error {
		if _, err := s.Service.EnableJob(conn, jobId, scope); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		var unknownJobErr *eventline.UnknownJobError

		if errors.As(err, &unknownJobErr) {
			h.ReplyError(404, "unknown_job", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return err
	}

	return nil
}

func (s *HTTPServer) DisableJob(h *HTTPHandler, jobId eventline.Id) error {
	scope := h.Context.ProjectScope()

	err := s.Service.Pg.WithTx(func(conn pg.Conn) error {
		if _, err := s.Service.DisableJob(conn, jobId, scope); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		var unknownJobErr *eventline.UnknownJobError

		if errors.As(err, &unknownJobErr) {
			h.ReplyError(404, "unknown_job", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return err
	}

	return nil
}

func (s *HTTPServer) ExecuteJob(h *HTTPHandler, jobId eventline.Id, input *eventline.JobExecutionInput) (*eventline.JobExecution, error) {
	scope := h.Context.ProjectScope()

	var jobExecution *eventline.JobExecution

	err := s.Service.Pg.WithTx(func(conn pg.Conn) error {
		var err error

		jobExecution, err = s.Service.ExecuteJob(conn, jobId, input, scope)
		if err != nil {
			return fmt.Errorf("cannot execute job: %w", err)
		}

		return nil
	})
	if err != nil {
		var unknownJobErr *eventline.UnknownJobError
		var validationErrors ejson.ValidationErrors

		if errors.As(err, &unknownJobErr) {
			h.ReplyError(404, "unknown_job", "%v", err)
		} else if errors.As(err, &validationErrors) {
			h.ReplyValidationErrors(validationErrors)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return nil, err
	}

	return jobExecution, nil
}
