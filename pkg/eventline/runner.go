package eventline

import (
	"fmt"
	"sync"
	"time"

	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/daemon"
	"github.com/exograd/go-daemon/pg"
	"github.com/exograd/go-log"
)

var RunnerDefs = map[string]*RunnerDef{}

type RunnerCfg interface {
	check.Object
}

type RunnerDef struct {
	Name        string
	Cfg         RunnerCfg
	Instantiate func(*Runner) RunnerBehaviour
}

type RunnerInitData struct {
	Log    *log.Logger
	Daemon *daemon.Daemon

	Def  *RunnerDef
	Cfg  RunnerCfg
	Data *RunnerData

	StopChan <-chan struct{}
	Wg       *sync.WaitGroup
}

type RunnerData struct {
	JobExecution     *JobExecution
	StepExecutions   StepExecutions
	ExecutionContext *ExecutionContext
	Project          *Project
	ProjectSettings  *ProjectSettings
}

type RunnerBehaviour interface {
	Start() error
}

type Runner struct {
	log       *log.Logger
	daemon    *daemon.Daemon
	cfg       RunnerCfg
	behaviour RunnerBehaviour

	jobExecution     *JobExecution
	stepExecutions   StepExecutions
	executionContext *ExecutionContext
	project          *Project
	projectSettings  *ProjectSettings

	environment map[string]string
	scope       Scope

	stopChan <-chan struct{}
	wg       *sync.WaitGroup
}

func NewRunner(data RunnerInitData) *Runner {
	r := &Runner{
		log:    data.Log,
		daemon: data.Daemon,
		cfg:    data.Cfg,

		jobExecution:     data.Data.JobExecution,
		stepExecutions:   data.Data.StepExecutions,
		executionContext: data.Data.ExecutionContext,
		project:          data.Data.Project,
		projectSettings:  data.Data.ProjectSettings,

		environment: data.Data.Environment(),
		scope:       NewProjectScope(data.Data.Project.Id),

		stopChan: data.StopChan,
		wg:       data.Wg,
	}

	r.behaviour = data.Def.Instantiate(r)

	return r
}

func (r *Runner) Start() error {
	return r.behaviour.Start()
}

func (rd *RunnerData) Environment() map[string]string {
	env := map[string]string{
		"EVENTLINE":                  "true",
		"EVENTLINE_PROJECT_ID":       rd.Project.Id.String(),
		"EVENTLINE_PROJECT_NAME":     rd.Project.Name,
		"EVENTLINE_JOB_ID":           rd.JobExecution.JobId.String(),
		"EVENTLINE_JOB_NAME":         rd.JobExecution.JobSpec.Name,
		"EVENTLINE_JOB_EXECUTION_ID": rd.JobExecution.Id.String(),
		"EVENTLINE_CONTEXT_PATH":     "/eventline/context.json",
	}

	for _, i := range rd.ExecutionContext.Identities {
		for name, value := range i.Environment() {
			env[name] = value
		}
	}

	for name, value := range rd.JobExecution.JobSpec.Environment {
		env[name] = value
	}

	return env
}

func (r *Runner) UpdateJobExecutionSuccess(jeId Id, scope Scope) (*JobExecution, error) {
	var je JobExecution

	err := r.daemon.Pg.WithTx(func(conn pg.Conn) error {
		if err := je.LoadForUpdate(conn, jeId, scope); err != nil {
			return fmt.Errorf("cannot load job execution: %w", err)
		}

		if je.Status == JobExecutionStatusAborted {
			return &JobExecutionAbortedError{Id: jeId}
		}

		now := time.Now().UTC()

		je.Status = JobExecutionStatusSuccessful
		je.EndTime = &now
		je.RefreshTime = nil

		if err := je.Update(conn); err != nil {
			return fmt.Errorf("cannot update job execution: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &je, nil
}

func (r *Runner) UpdateJobExecutionAbortion(jeId Id, scope Scope) (*JobExecution, StepExecutions, error) {
	var je JobExecution
	var ses StepExecutions

	err := r.daemon.Pg.WithTx(func(conn pg.Conn) error {
		if err := je.LoadForUpdate(conn, jeId, scope); err != nil {
			return fmt.Errorf("cannot load job execution: %w", err)
		}

		if je.Status == JobExecutionStatusAborted {
			return &JobExecutionAbortedError{Id: jeId}
		}

		now := time.Now().UTC()

		je.Status = JobExecutionStatusAborted
		je.EndTime = &now
		je.RefreshTime = nil

		if err := je.Update(conn); err != nil {
			return fmt.Errorf("cannot update job: %w", err)
		}

		if err := ses.LoadByJobExecutionId(conn, jeId); err != nil {
			return fmt.Errorf("cannot load step executions: %w", err)
		}

		for _, se := range ses {
			if !se.Finished() {
				se.Status = StepExecutionStatusAborted
				if se.StartTime != nil {
					se.EndTime = &now
				}

				if err := se.Update(conn); err != nil {
					return fmt.Errorf("cannot update step %d: %w",
						se.Position, err)
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return &je, ses, nil
}

func (r *Runner) UpdateJobExecutionFailure(jeId Id, jeErr error, scope Scope) (*JobExecution, StepExecutions, error) {
	var je JobExecution
	var ses StepExecutions

	err := r.daemon.Pg.WithTx(func(conn pg.Conn) error {
		if err := je.LoadForUpdate(conn, jeId, scope); err != nil {
			return fmt.Errorf("cannot load job execution: %w", err)
		}

		if je.Status == JobExecutionStatusAborted {
			return &JobExecutionAbortedError{Id: jeId}
		}

		now := time.Now().UTC()

		je.Status = JobExecutionStatusFailed
		je.EndTime = &now
		je.FailureMessage = jeErr.Error()
		je.RefreshTime = nil

		if err := je.Update(conn); err != nil {
			return fmt.Errorf("cannot update job: %w", err)
		}

		if err := ses.LoadByJobExecutionId(conn, jeId); err != nil {
			return fmt.Errorf("cannot load step executions: %w", err)
		}

		for _, se := range ses {
			if !se.Finished() {
				se.Status = StepExecutionStatusAborted
				if se.StartTime != nil {
					se.EndTime = &now
				}

				if err := se.Update(conn); err != nil {
					return fmt.Errorf("cannot update step %d: %w",
						se.Position, err)
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return &je, ses, nil
}

func (r *Runner) UpdateStepExecutionStart(jeId, seId Id, scope Scope) (*JobExecution, *StepExecution, error) {
	return r.updateStepExecution(jeId, seId, func(se *StepExecution) {
		now := time.Now().UTC()

		se.Status = StepExecutionStatusStarted
		se.StartTime = &now
		se.FailureMessage = ""
		se.Output = ""
	}, scope)
}

func (r *Runner) UpdateStepExecutionAborted(jeId, seId Id, scope Scope) (*JobExecution, *StepExecution, error) {
	return r.updateStepExecution(jeId, seId, func(se *StepExecution) {
		now := time.Now().UTC()

		se.Status = StepExecutionStatusAborted
		se.EndTime = &now
	}, scope)
}

func (r *Runner) UpdateStepExecutionSuccess(jeId, seId Id, scope Scope) (*JobExecution, *StepExecution, error) {
	return r.updateStepExecution(jeId, seId, func(se *StepExecution) {
		now := time.Now().UTC()

		se.Status = StepExecutionStatusSuccessful
		se.EndTime = &now
	}, scope)
}

func (r *Runner) UpdateStepExecutionFailure(jeId, seId Id, err error, scope Scope) (*JobExecution, *StepExecution, error) {
	return r.updateStepExecution(jeId, seId, func(se *StepExecution) {
		now := time.Now().UTC()

		se.Status = StepExecutionStatusFailed
		se.EndTime = &now
		se.FailureMessage = err.Error()
	}, scope)
}

func (r *Runner) updateStepExecution(jeId, seId Id, fn func(*StepExecution), scope Scope) (*JobExecution, *StepExecution, error) {
	var je JobExecution
	var se StepExecution

	err := r.daemon.Pg.WithTx(func(conn pg.Conn) error {
		if err := je.LoadForUpdate(conn, jeId, scope); err != nil {
			return fmt.Errorf("cannot load job execution: %w", err)
		}

		if je.Status == JobExecutionStatusAborted {
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
