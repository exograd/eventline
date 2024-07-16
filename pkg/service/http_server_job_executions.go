package service

import (
	"errors"
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/service/pkg/pg"
)

func (s *HTTPServer) LoadJobExecution(h *HTTPHandler, jeId eventline.Id) (*eventline.JobExecution, error) {
	scope := h.Context.ProjectScope()

	var je eventline.JobExecution

	err := s.Pg.WithConn(func(conn pg.Conn) error {
		if err := je.Load(conn, jeId, scope); err != nil {
			return fmt.Errorf("cannot load job execution: %w", err)
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

		return nil, err
	}

	return &je, nil
}

func (s *HTTPServer) AbortJobExecution(h *HTTPHandler, jeId eventline.Id) error {
	scope := h.Context.ProjectScope()

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

		return err
	}

	return nil
}

func (s *HTTPServer) RestartJobExecution(h *HTTPHandler, jeId eventline.Id) error {
	scope := h.Context.ProjectScope()

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

		return err
	}

	return nil
}
