package service

import (
	"errors"
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/pg"
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