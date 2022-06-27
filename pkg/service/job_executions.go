package service

import (
	"fmt"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/pg"
)

func (s *Service) LoadJobExecutionContext(conn pg.Conn, je *eventline.JobExecution) (*eventline.ExecutionContext, error) {
	var ctx eventline.ExecutionContext

	err := s.Daemon.Pg.WithConn(func(conn pg.Conn) (err error) {
		err = ctx.Load(conn, je)
		return
	})
	if err != nil {
		return nil, err
	}

	return &ctx, nil
}

func (s *Service) StartJobExecution(conn pg.Conn, je *eventline.JobExecution, scope eventline.Scope) error {
	now := time.Now().UTC()

	// Mark the job execution as started and update it
	je.Status = eventline.JobExecutionStatusStarted
	je.StartTime = &now
	je.RefreshTime = &now
	je.FailureMessage = ""

	if err := je.Update(conn); err != nil {
		return fmt.Errorf("cannot update job execution %q: %w", je.Id, err)
	}

	// Load step executions
	var ses eventline.StepExecutions
	if err := ses.LoadByJobExecutionId(conn, je.Id); err != nil {
		return fmt.Errorf("cannot load step executions: %w", err)
	}

	// Load the execution context
	ectx, err := s.LoadJobExecutionContext(conn, je)
	if err != nil {
		return fmt.Errorf("cannot load execution context: %w", err)
	}

	// Load the project and its settings
	projectId := scope.(*eventline.ProjectScope).ProjectId

	var project eventline.Project
	if err := project.Load(conn, projectId); err != nil {
		return fmt.Errorf("cannot load project: %w", err)
	}

	var projectSettings eventline.ProjectSettings
	if err := projectSettings.Load(conn, projectId); err != nil {
		return fmt.Errorf("cannot load project settings: %w", err)
	}

	// Create and start a runner
	runnerData := eventline.RunnerData{
		JobExecution:     je,
		StepExecutions:   ses,
		ExecutionContext: ectx,
		Project:          &project,
		ProjectSettings:  &projectSettings,
	}

	if _, err := s.StartRunner(&runnerData); err != nil {
		return fmt.Errorf("cannot start runner: %w", err)
	}

	return nil
}

func (s *Service) AbortJobExecution(jeId eventline.Id, scope eventline.Scope) (*eventline.JobExecution, error) {
	var je eventline.JobExecution

	now := time.Now().UTC()

	err := s.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		if err := je.LoadForUpdate(conn, jeId, scope); err != nil {
			return fmt.Errorf("cannot load job execution: %w", err)
		}

		if je.Finished() {
			return &eventline.JobExecutionFinishedError{Id: jeId}
		}

		je.Status = eventline.JobExecutionStatusAborted
		if je.StartTime != nil {
			je.EndTime = &now
		}
		je.RefreshTime = nil

		if err := je.Update(conn); err != nil {
			return fmt.Errorf("cannot update job execution: %w", err)
		}

		var ses eventline.StepExecutions
		err := ses.LoadByJobExecutionIdForUpdate(conn, jeId)
		if err != nil {
			return fmt.Errorf("cannot load step executions: %w", err)
		}

		for _, se := range ses {
			if !se.Finished() {
				se.Status = eventline.StepExecutionStatusAborted
				if se.StartTime != nil {
					se.EndTime = &now
				}
			}

			if err := se.Update(conn); err != nil {
				return fmt.Errorf("cannot update step %d: %w",
					se.Position, err)
			}

			if err := se.Update(conn); err != nil {
				return fmt.Errorf("cannot update step execution: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &je, nil
}

func (s *Service) RestartJobExecution(jeId eventline.Id, scope eventline.Scope) (*eventline.JobExecution, error) {
	var je eventline.JobExecution

	now := time.Now().UTC()

	err := s.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		if err := je.LoadForUpdate(conn, jeId, scope); err != nil {
			return fmt.Errorf("cannot load job execution: %w", err)
		}

		if !je.Finished() {
			return &eventline.JobExecutionNotFinishedError{Id: jeId}
		}

		je.Status = eventline.JobExecutionStatusCreated
		je.StartTime = nil
		je.EndTime = nil
		je.UpdateTime = now
		je.RefreshTime = nil
		je.FailureMessage = ""

		if err := je.Update(conn); err != nil {
			return fmt.Errorf("cannot update job execution: %w", err)
		}

		var ses eventline.StepExecutions
		err := ses.LoadByJobExecutionIdForUpdate(conn, jeId)
		if err != nil {
			return fmt.Errorf("cannot load step executions: %w", err)
		}

		for _, se := range ses {
			se.Status = eventline.StepExecutionStatusCreated
			se.StartTime = nil
			se.EndTime = nil
			se.FailureMessage = ""
			se.Output = ""

			if err := se.Update(conn); err != nil {
				return fmt.Errorf("cannot update step execution: %w", err)
			}

			// StepExecution.Update does not update the output column on
			// purpose, so we have to clear it separately.
			if err := se.ClearOutput(conn); err != nil {
				return fmt.Errorf("cannot update step execution: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &je, nil
}
