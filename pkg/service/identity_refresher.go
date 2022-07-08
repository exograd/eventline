package service

import (
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

		// If the refresh fails, we update the refresh time to try again
		// later.
		refreshErr = ir.Service.refreshIdentity(conn, identity, scope)
		if refreshErr != nil {
			ir.Log.Error("cannot refresh identity: %v", err)

			err := ir.sendErrorNotification(conn, identity, err, scope)
			if err != nil {
				ir.Log.Error("cannot send notification: %w", err)
			}

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

func (ir *IdentityRefresher) sendErrorNotification(conn pg.Conn, identity *eventline.Identity, refreshErr error, scope eventline.Scope) error {
	projectId := scope.(*eventline.ProjectScope).ProjectId

	var settings eventline.ProjectNotificationSettings
	if err := settings.Load(conn, projectId); err != nil {
		return fmt.Errorf("cannot load notification settings: %w", err)
	}

	if !settings.OnIdentityRefreshError {
		return nil
	}

	subject := "identity refresh error"
	templateName := "identity_refresh_error.txt"
	templateData := struct{}{}

	return ir.Service.CreateNotification(conn, settings.EmailAddresses,
		subject, templateName, templateData, scope)
}
