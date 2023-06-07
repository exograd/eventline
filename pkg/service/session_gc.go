package service

import (
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/galdor/go-log"
	"github.com/galdor/go-service/pkg/pg"
)

type SessionGC struct {
	Log     *log.Logger
	Service *Service
}

func NewSessionGC(s *Service) *SessionGC {
	return &SessionGC{
		Service: s,
	}
}

func (sgc *SessionGC) Init(w *eventline.Worker) {
	sgc.Log = w.Log
}

func (sgc *SessionGC) Start() error {
	return nil
}

func (sgc *SessionGC) Stop() {
}

func (sgc *SessionGC) ProcessJob() (bool, error) {
	var deleted bool

	retention := sgc.Service.Cfg.SessionRetention

	err := sgc.Service.Pg.WithTx(func(conn pg.Conn) error {
		n, err := eventline.DeleteOldSessions(conn, retention)
		if err != nil {
			return fmt.Errorf("cannot delete sessions: %w", err)
		} else if n == 0 {
			return nil
		}

		sgc.Log.Debug(1, "%d sessions deleted", n)

		deleted = true
		return nil
	})
	if err != nil {
		return false, err
	}

	return deleted, nil
}
