package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/galdor/go-service/pkg/pg"
)

func (s *WebHTTPServer) setupStepExecutionRoutes() {
	s.route("/step_executions/id/:id/log_file", "GET",
		s.hStepExecutionsIdLogFileGET,
		HTTPRouteOptions{Project: true})
}

func (s *WebHTTPServer) hStepExecutionsIdLogFileGET(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	seId, err := h.IdPathVariable("id")
	if err != nil {
		return
	}

	var stepExecution eventline.StepExecution

	err = s.Pg.WithConn(func(conn pg.Conn) error {
		err := stepExecution.Load(conn, seId, scope)
		if err != nil {
			return fmt.Errorf("cannot load step execution: %w", err)
		}

		return nil
	})
	if err != nil {
		var unknownStepExecutionErr *eventline.UnknownStepExecutionError

		if errors.As(err, &unknownStepExecutionErr) {
			h.ReplyError(404, "unknown_step_execution", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return
	}

	filename := fmt.Sprintf("job-execution-%s-step-%03d.log",
		stepExecution.JobExecutionId.String(), stepExecution.Position)

	header := h.ResponseWriter.Header()
	header.Set("Content-Type", "text/plain")
	header.Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	h.Reply(200, strings.NewReader(stepExecution.Output))
}
