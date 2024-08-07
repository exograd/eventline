package service

import (
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/log"
	"go.n16f.net/service/pkg/pg"
)

type JobExecutionGC struct {
	Log     *log.Logger
	Service *Service
}

func NewJobExecutionGC(s *Service) *JobExecutionGC {
	return &JobExecutionGC{
		Service: s,
	}
}

func (jegc *JobExecutionGC) Init(w *eventline.Worker) {
	jegc.Log = w.Log
}

func (jegc *JobExecutionGC) Start() error {
	return nil
}

func (jegc *JobExecutionGC) Stop() {
}

func (jegc *JobExecutionGC) ProcessJob() (bool, error) {
	var deleted bool

	err := jegc.Service.Pg.WithTx(func(conn pg.Conn) error {
		n, err := eventline.DeleteExpiredJobExecutions(conn)
		if err != nil {
			return fmt.Errorf("cannot delete job executions: %w", err)
		} else if n == 0 {
			return nil
		}

		jegc.Log.Debug(1, "%d job executions deleted", n)

		deleted = true
		return nil
	})
	if err != nil {
		return false, err
	}

	return deleted, nil
}
