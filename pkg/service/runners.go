package service

import (
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-log"
)

type Runner interface {
	Start() error
}

func (s *Service) StartRunner(data *eventline.RunnerData) (Runner, error) {
	name := data.JobExecution.JobSpec.Runner.Name

	def, found := s.runnerDefs[name]
	if !found {
		return nil, fmt.Errorf("unknown runner %q", name)
	}

	logger := s.Log.Child("runner", log.Data{
		"runner":        "local",
		"job_execution": data.JobExecution.Id.String(),
	})

	initData := eventline.RunnerInitData{
		Log:    logger,
		Daemon: s.Daemon,

		Def:  def,
		Cfg:  def.Cfg,
		Data: data,

		StopChan: s.runnerStopChan,
		Wg:       &s.runnerWg,
	}

	runner := eventline.NewRunner(initData)

	if err := runner.Start(); err != nil {
		return nil, err
	}

	return runner, nil
}
