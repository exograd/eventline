package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/pg"
	"github.com/exograd/go-log"
)

type IdentityRefresher struct {
	Log     *log.Logger
	Service *Service

	w *eventline.Worker
}

func NewIdentityRefresher(s *Service) *IdentityRefresher {
	return &IdentityRefresher{
		Service: s,
	}
}

func (ir *IdentityRefresher) Init(w *eventline.Worker) {
	ir.Log = w.Log
}

func (ir *IdentityRefresher) Start() error {
	return nil
}

func (ir *IdentityRefresher) Stop() {
}

func (ir *IdentityRefresher) ProcessJob() (bool, error) {
	var refreshErr error
	var processed bool

	err := ir.Service.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		identity, err := eventline.LoadIdentityForRefresh(conn)
		if err != nil {
			return fmt.Errorf("cannot load identity: %w", err)
		} else if identity == nil {
			return nil
		}

		ir.Log.Info("refreshing identity %q", identity.Id)

		// Refreshable identities always have a project id
		scope := eventline.NewProjectScope(*identity.ProjectId)

		err = ir.Service.refreshIdentity(conn, identity, scope)

		var externalErr *eventline.ExternalSubscriptionError
		isExternalErr := errors.As(err, &externalErr)

		if err != nil && !isExternalErr {
			return fmt.Errorf("cannot refresh identity: %w", err)
		}

		if err != nil && isExternalErr {
			refreshErr = err

			ir.Log.Error("cannot refresh identity: %v", err)

			delay := 600 * time.Second
			refreshTime := identity.RefreshTime.Add(delay)
			identity.RefreshTime = &refreshTime
		}

		if err := identity.Update(conn); err != nil {
			return fmt.Errorf("cannot update identity %q: %w",
				identity.Id, err)
		}

		processed = true
		return nil
	})
	if err != nil {
		return false, err
	}

	if refreshErr != nil {
		return false, refreshErr
	}

	return processed, nil
}
