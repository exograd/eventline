package service

import (
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/utils"
)

type RunnersCfg struct {
	Local LocalRunnerCfg `json:"local"`
}

type Runner interface {
	Start() error
}

type RunnerData struct {
	JobExecution     *eventline.JobExecution
	StepExecutions   eventline.StepExecutions
	ExecutionContext *eventline.ExecutionContext
	Project          *eventline.Project
	ProjectSettings  *eventline.ProjectSettings
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

func (s *Service) StartRunner(data *RunnerData) (Runner, error) {
	runtimeName := data.JobExecution.JobSpec.Runtime.Name

	var runner Runner

	switch runtimeName {
	case eventline.RuntimeNameLocal:
		runner = NewLocalRunner(s, data)

	default:
		utils.Panicf("unhandled runtime %q", runtimeName)
	}

	if err := runner.Start(); err != nil {
		return nil, err
	}

	return runner, nil
}
