package service

import (
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/log"
	"go.n16f.net/service/pkg/pg"
)

type EventWorker struct {
	Log     *log.Logger
	Service *Service
}

func NewEventWorker(s *Service) *EventWorker {
	return &EventWorker{
		Service: s,
	}
}

func (ew *EventWorker) Init(w *eventline.Worker) {
	ew.Log = w.Log
}

func (ew *EventWorker) Start() error {
	return nil
}

func (ew *EventWorker) Stop() {
}

func (ew *EventWorker) ProcessJob() (bool, error) {
	var processed bool
	var jeCreated bool

	err := ew.Service.Pg.WithTx(func(conn pg.Conn) error {
		event, err := eventline.LoadEventForProcessing(conn)
		if err != nil {
			return fmt.Errorf("cannot load event: %w", err)
		} else if event == nil {
			return nil
		}

		ew.Log.Info("processing event %q", event.Id)

		scope := eventline.NewProjectScope(event.ProjectId)

		jeCreated, err = ew.Service.ProcessEvent(conn, event, scope)
		if err != nil {
			return fmt.Errorf("cannot process event %q: %w", event.Id, err)
		}

		processed = true
		return nil
	})
	if err != nil {
		return false, err
	}

	if jeCreated {
		if w := ew.Service.FindWorker("job-scheduler"); w != nil {
			w.WakeUp()
		}
	}

	return processed, nil
}
