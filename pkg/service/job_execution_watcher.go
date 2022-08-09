package service

import (
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/dlog"
	"github.com/exograd/go-daemon/pg"
)

type JobExecutionWatcher struct {
	Log     *dlog.Logger
	Service *Service
}

func NewJobExecutionWatcher(s *Service) *JobExecutionWatcher {
	return &JobExecutionWatcher{
		Service: s,
	}
}

func (w *JobExecutionWatcher) Init(worker *eventline.Worker) {
	w.Log = worker.Log
}

func (w *JobExecutionWatcher) Start() error {
	return nil
}

func (w *JobExecutionWatcher) Stop() {
}

func (w *JobExecutionWatcher) ProcessJob() (bool, error) {
	var deleted bool

	err := w.Service.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		timeout := w.Service.Cfg.JobExecutionTimeout

		je, err := eventline.LoadDeadJobExecution(conn, timeout)
		if err != nil {
			return fmt.Errorf("cannot load job execution: %w", err)
		} else if je == nil {
			return nil
		}

		w.Log.Info("stopping dead job execution %q", je.Id)

		err = w.Service.UpdateJobExecutionFailure(conn, je,
			"execution timeout")
		if err != nil {
			return fmt.Errorf("cannot update job execution %q: %w", je.Id, err)
		}

		deleted = true
		return nil
	})
	if err != nil {
		return false, err
	}

	return deleted, nil
}
