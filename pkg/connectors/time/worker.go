package time

import (
	"fmt"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/log"
	"go.n16f.net/service/pkg/pg"
)

type Worker struct {
	Log *log.Logger
	Pg  *pg.Client

	worker *eventline.Worker
}

func NewWorker() *Worker {
	return &Worker{}
}

func (w *Worker) Init(ew *eventline.Worker) {
	w.Log = ew.Log
	w.Pg = ew.Pg

	w.worker = ew
}

func (w *Worker) Start() error {
	return nil
}

func (w *Worker) Stop() {
}

func (w *Worker) ProcessJob() (bool, error) {
	var event *eventline.Event

	err := w.Pg.WithTx(func(conn pg.Conn) error {
		s, es, err := LoadSubscriptionForProcessing(conn)
		if err != nil {
			return fmt.Errorf("cannot load subscription: %w", err)
		} else if s == nil {
			return nil
		}

		event, err = w.processSubscription(conn, s, es)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return false, err
	}

	if event != nil {
		w.worker.Cfg.NotificationChan <- event
		return true, nil
	}

	return false, nil
}

func (w *Worker) processSubscription(conn pg.Conn, s *Subscription, es *eventline.Subscription) (*eventline.Event, error) {
	w.Log.Info("processing subscription %q", s.Id)

	params := es.Parameters.(*Parameters)

	var etime time.Time

	if params.Periodic == nil {
		// For non-periodic timers, we want to emit events for all ticks. If a
		// job is supposed to run every day at 2am, we want it executed even
		// if Eventline was down between 1am and 3am.
		etime = s.NextTick
	} else {
		// For periodic timers, we do not want to emit events for ticks which
		// happened while Eventline was down. If the server is not running for
		// 2 hours, the last thing we want is creating 240 events for a 30
		// second timer.
		etime = time.Now().UTC()
	}

	s.LastTick = &etime
	s.NextTick = params.NextTick(etime)

	event := es.NewEvent("time", "tick", &etime, &TickEvent{})
	if err := event.Insert(conn); err != nil {
		return nil, fmt.Errorf("cannot insert event: %w", err)
	}

	if err := s.Update(conn); err != nil {
		return nil, fmt.Errorf("cannot update subscription: %w", err)
	}

	return event, nil
}
