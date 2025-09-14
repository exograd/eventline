package service

import (
	"fmt"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/service/pkg/pg"
	"go.n16f.net/uuid"
)

func (s *Service) ReplayEvent(eventId uuid.UUID, scope eventline.Scope) (*eventline.Event, error) {
	var event eventline.Event

	now := time.Now().UTC()

	err := s.Pg.WithTx(func(conn pg.Conn) error {
		var originalEvent eventline.Event
		if err := originalEvent.Load(conn, eventId, scope); err != nil {
			return fmt.Errorf("cannot load event: %w", err)
		}

		event = originalEvent

		event.Id = uuid.MustGenerate(uuid.V7)
		event.CreationTime = now
		event.Processed = false
		event.OriginalEventId = &eventId

		if err := event.Insert(conn); err != nil {
			return fmt.Errorf("cannot insert event: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &event, nil
}

func (s *Service) ProcessEvent(conn pg.Conn, event *eventline.Event, scope eventline.Scope) (bool, error) {
	var jeCreated bool

	// Load the job
	var job eventline.Job
	if err := job.Load(conn, event.JobId, scope); err != nil {
		return false, fmt.Errorf("cannot load job %q: %w", event.JobId, err)
	}

	// Instantiate the job if the job is enabled and filters match
	//
	// Note that the job may not have a trigger. This is not intuitive: if
	// there is an event, there must be a subscription and therefore a
	// trigger. But a job can be updated while there are unprocessed events in
	// the database. If the update removes the trigger, we end up with an
	// event referencing a job which does not have a trigger anymore.
	if !job.Disabled {
		filtersMatch := true
		if trigger := job.Spec.Trigger; trigger != nil {
			filtersMatch = trigger.Filters.Match(event.DataValue)
		}

		if filtersMatch {
			_, err := s.InstantiateJob(conn, &job, event, nil, scope)
			if err != nil {
				return false, fmt.Errorf("cannot instantiate job %q: %w",
					event.JobId, err)
			}

			jeCreated = true
		}
	}

	// Mark the event as processed
	event.Processed = true

	if err := event.Update(conn); err != nil {
		return false, fmt.Errorf("cannot update event %q: %w", event.Id, err)
	}

	return jeCreated, nil
}
