package time

import (
	"fmt"

	"github.com/exograd/evgo/pkg/eventline"
	"github.com/exograd/go-daemon/daemon"
	"github.com/exograd/go-daemon/pg"
	"github.com/exograd/go-log"
)

type Worker struct {
	Log    *log.Logger
	Daemon *daemon.Daemon

	worker *eventline.Worker
}

func NewWorker() *Worker {
	return &Worker{}
}

func (w *Worker) Init(ew *eventline.Worker) {
	w.Log = ew.Log
	w.Daemon = ew.Daemon

	w.worker = ew
}

func (w *Worker) Start() error {
	return nil
}

func (w *Worker) Stop() {
}

func (w *Worker) ProcessJob() (bool, error) {
	var event *eventline.Event

	err := w.Daemon.Pg.WithTx(func(conn pg.Conn) error {
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
	// The most important value here is expectedTick, which is the time the
	// event was supposed to be generated; the actual time can be higher since
	// there is always a delay (the time needed for the worker to pick up the
	// subscription).
	//
	// The next tick is computed based on this tick to avoid potential
	// drifting.

	w.Log.Info("processing subscription %q", s.Id)

	params := es.Parameters.(*Parameters)

	expectedTick := s.NextTick

	s.LastTick = &expectedTick
	s.NextTick = params.NextTick(expectedTick)

	event := es.NewEvent("time", "tick", &expectedTick, &TickEvent{})
	if err := event.Insert(conn); err != nil {
		return nil, fmt.Errorf("cannot insert event: %w", err)
	}

	if err := s.Update(conn); err != nil {
		return nil, fmt.Errorf("cannot update subscription: %w", err)
	}

	return event, nil
}
