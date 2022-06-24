package service

import (
	"fmt"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/pg"
)

func (s *Service) updateStepExecution(jeId, seId eventline.Id, fn func(*eventline.StepExecution), scope eventline.Scope) (*eventline.JobExecution, *eventline.StepExecution, error) {
	var je eventline.JobExecution
	var se eventline.StepExecution

	err := s.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		if err := je.LoadForUpdate(conn, jeId, scope); err != nil {
			return fmt.Errorf("cannot load job execution: %w", err)
		}

		if je.Status == eventline.JobExecutionStatusAborted {
			return &JobExecutionAbortedError{Id: jeId}
		}

		if err := se.Load(conn, seId, scope); err != nil {
			return fmt.Errorf("cannot load step execution: %w", err)
		}

		fn(&se)

		if err := se.Update(conn); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return &je, &se, nil
}

func (s *Service) UpdateStepExecutionStart(jeId, seId eventline.Id, scope eventline.Scope) (*eventline.JobExecution, *eventline.StepExecution, error) {
	return s.updateStepExecution(jeId, seId, func(se *eventline.StepExecution) {
		now := time.Now().UTC()

		se.Status = eventline.StepExecutionStatusStarted
		se.StartTime = &now
		se.FailureMessage = ""
		se.Output = ""
	}, scope)
}

func (s *Service) UpdateStepExecutionAborted(jeId, seId eventline.Id, scope eventline.Scope) (*eventline.JobExecution, *eventline.StepExecution, error) {
	return s.updateStepExecution(jeId, seId, func(se *eventline.StepExecution) {
		now := time.Now().UTC()

		se.Status = eventline.StepExecutionStatusAborted
		se.EndTime = &now
	}, scope)
}

func (s *Service) UpdateStepExecutionSuccess(jeId, seId eventline.Id, scope eventline.Scope) (*eventline.JobExecution, *eventline.StepExecution, error) {
	return s.updateStepExecution(jeId, seId, func(se *eventline.StepExecution) {
		now := time.Now().UTC()

		se.Status = eventline.StepExecutionStatusSuccessful
		se.EndTime = &now
	}, scope)
}

func (s *Service) UpdateStepExecutionFailure(jeId, seId eventline.Id, err error, scope eventline.Scope) (*eventline.JobExecution, *eventline.StepExecution, error) {
	return s.updateStepExecution(jeId, seId, func(se *eventline.StepExecution) {
		now := time.Now().UTC()

		se.Status = eventline.StepExecutionStatusFailed
		se.EndTime = &now
		se.FailureMessage = err.Error()
	}, scope)
}
