package service

import (
	"fmt"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/dlog"
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

	logger := s.Log.Child("runner", dlog.Data{
		"runner":        name,
		"job_execution": data.JobExecution.Id.String(),
	})

	refreshIntervalSeconds := s.Cfg.JobExecutionRefreshInterval
	refreshInterval := time.Duration(refreshIntervalSeconds) * time.Second

	initData := eventline.RunnerInitData{
		Log:    logger,
		Daemon: s.Daemon,

		Def:  def,
		Cfg:  def.Cfg,
		Data: data,

		TerminationChan: s.jobExecutionTerminationChan,

		RefreshInterval: refreshInterval,

		StopChan: s.runnerStopChan,
		Wg:       &s.runnerWg,
	}

	runner, err := eventline.NewRunner(initData)
	if err != nil {
		return nil, err
	}

	if err := runner.Start(); err != nil {
		return nil, err
	}

	return runner, nil
}
