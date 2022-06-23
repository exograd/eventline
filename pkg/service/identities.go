package service

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/exograd/evgo/pkg/eventline"
	"github.com/exograd/go-daemon/dhttp"
	"github.com/exograd/go-daemon/pg"
	"github.com/exograd/go-log"
)

var (
	ErrIdentityNotRefreshable = errors.New("identity is not refreshable")
)

type DuplicateIdentityNameError struct {
	Name string
}

func (err DuplicateIdentityNameError) Error() string {
	return fmt.Sprintf("duplicate identity name %q", err.Name)
}

type IdentityInUseError struct {
	Id eventline.Id
}

func (err IdentityInUseError) Error() string {
	return fmt.Sprintf("identity %q is currently being used", err.Id)
}

func (s *Service) CreateIdentity(newIdentity *eventline.NewIdentity, scope eventline.Scope) (*eventline.Identity, error) {
	var identity *eventline.Identity

	projectScope := scope.(*eventline.ProjectScope)

	err := s.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		now := time.Now().UTC()

		exists, err := eventline.IdentityNameExists(conn, newIdentity.Name, scope)
		if err != nil {
			return fmt.Errorf("cannot check identity name existence: %w", err)
		} else if exists {
			return &DuplicateIdentityNameError{Name: newIdentity.Name}
		}

		cdef := eventline.GetConnectorDef(newIdentity.Connector)
		idef := cdef.Identity(newIdentity.Type)

		status := eventline.IdentityStatusReady
		if idef.DeferredReadiness {
			status = eventline.IdentityStatusPending
		}

		identity = &eventline.Identity{
			Id:           eventline.GenerateId(),
			ProjectId:    &projectScope.ProjectId,
			Name:         newIdentity.Name,
			Status:       status,
			CreationTime: now,
			UpdateTime:   now,
			Connector:    newIdentity.Connector,
			Type:         newIdentity.Type,
			Data:         newIdentity.Data,
		}

		if err := identity.Insert(conn); err != nil {
			return fmt.Errorf("cannot insert identity: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return identity, nil
}

func (s *Service) UpdateIdentity(identityId eventline.Id, newIdentity *eventline.NewIdentity, scope eventline.Scope) (*eventline.Identity, error) {
	var identity eventline.Identity

	err := s.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		if err := identity.LoadForUpdate(conn, identityId, scope); err != nil {
			return fmt.Errorf("cannot load identity: %w", err)
		}

		if newIdentity.Name != identity.Name {
			exists, err := eventline.IdentityNameExists(conn, newIdentity.Name,
				scope)
			if err != nil {
				return fmt.Errorf("cannot check identity name existence: %w",
					err)
			} else if exists {
				return &DuplicateIdentityNameError{Name: newIdentity.Name}
			}

			used, err := identity.IsUsed(conn, scope)
			if err != nil {
				return fmt.Errorf("cannot check identity usage: %w", err)
			} else if used {
				return &IdentityInUseError{Id: identity.Id}
			}
		}

		now := time.Now().UTC()

		identity.Name = newIdentity.Name
		identity.UpdateTime = now
		identity.Connector = newIdentity.Connector
		identity.Type = newIdentity.Type
		identity.Data = newIdentity.Data

		if err := identity.Update(conn); err != nil {
			return fmt.Errorf("cannot update identity: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &identity, nil
}

func (s *Service) DeleteIdentity(identityId eventline.Id, scope eventline.Scope) error {
	return s.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		var identity eventline.Identity

		if err := identity.LoadForUpdate(conn, identityId, scope); err != nil {
			return fmt.Errorf("cannot load identity: %w", err)
		}

		used, err := identity.IsUsed(conn, scope)
		if err != nil {
			return fmt.Errorf("cannot check identity usage: %w", err)
		} else if used {
			return &IdentityInUseError{Id: identity.Id}
		}

		if err := identity.Delete(conn); err != nil {
			return err
		}

		return nil
	})
}

func (s *Service) IdentityRedirectionURI(identity *eventline.Identity, sessionId eventline.Id, defaultURI string) (string, error) {
	// For the time being, OAuth2 identities are the only ones using a
	// redirection mechanism.

	state, err := EncodeOAuth2State(identity.Id, sessionId)
	if err != nil {
		return "", fmt.Errorf("cannot encode oauth2 state: %w", err)
	}

	httpClient, err := s.oauth2HTTPClient(identity.Id, &sessionId)
	if err != nil {
		return "", fmt.Errorf("cannot create http client: %w", err)
	}

	path := path.Join("ext", "connectors", identity.Connector, identity.Type)
	redirectionURI := s.WebHTTPServerURI.ResolveReference(&url.URL{Path: path})

	identityData, ok := identity.Data.(eventline.OAuth2IdentityData)
	if ok {
		return identityData.RedirectionURI(httpClient, state,
			redirectionURI.String())
	}

	return defaultURI, nil
}

func (s *Service) RefreshIdentity(identityId eventline.Id, scope eventline.Scope) error {
	var refreshErr error

	err := s.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		var identity eventline.Identity
		if err := identity.LoadForUpdate(conn, identityId, scope); err != nil {
			return fmt.Errorf("cannot load identity: %w", err)
		}

		if !identity.Refreshable() {
			return ErrIdentityNotRefreshable
		}

		// We want the transaction to be commited if the refresh procedure
		// fails in order to make sure the identity refresh time is updated.
		if err := s.refreshIdentity(conn, &identity, scope); err != nil {
			refreshErr = err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return refreshErr
}

func (s *Service) refreshIdentity(conn pg.Conn, identity *eventline.Identity, scope eventline.Scope) error {
	identityData := identity.Data.(eventline.RefreshableOAuth2IdentityData)

	httpClient, err := s.oauth2HTTPClient(identity.Id, nil)
	if err != nil {
		return fmt.Errorf("cannot create http client: %w", err)
	}

	if err := identityData.Refresh(httpClient); err != nil {
		return err
	}

	refreshTime := identityData.RefreshTime()
	identity.RefreshTime = &refreshTime

	if err := identity.Update(conn); err != nil {
		return fmt.Errorf("cannot update identity: %w", err)
	}

	return nil
}

func (s *Service) oauth2HTTPClient(identityId eventline.Id, sessionId *eventline.Id) (*http.Client, error) {

	logger := s.Log.Child("oauth2", log.Data{
		"identity": identityId.String(),
	})

	if sessionId != nil {
		logger.Data["session"] = sessionId.String()
	}

	httpClientCfg := dhttp.ClientCfg{
		Log:         logger,
		LogRequests: true,
	}

	httpClient, err := dhttp.NewClient(httpClientCfg)
	if err != nil {
		return nil, fmt.Errorf("cannot create http client: %w", err)
	}

	return httpClient.Client, nil
}
