package service

import (
	"fmt"
	"time"

	"github.com/exograd/evgo/pkg/eventline"
	"github.com/exograd/go-daemon/pg"
	"github.com/google/uuid"
)

type DuplicateAPIKeyNameError struct {
	Name string
}

func (err DuplicateAPIKeyNameError) Error() string {
	return fmt.Sprintf("duplicate api key name %q", err.Name)
}

func (s *Service) CreateAPIKey(newAPIKey *eventline.NewAPIKey, scope eventline.Scope) (*eventline.APIKey, string, error) {
	var apiKey *eventline.APIKey
	var key string

	accountScope := scope.(*eventline.AccountScope)

	err := s.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		now := time.Now().UTC()

		exists, err := eventline.APIKeyNameExists(conn, newAPIKey.Name, scope)
		if err != nil {
			return fmt.Errorf("cannot check api key name existence: %w", err)
		} else if exists {
			return &DuplicateAPIKeyNameError{Name: newAPIKey.Name}
		}

		key = uuid.NewString()

		apiKey = &eventline.APIKey{
			Id:           eventline.GenerateId(),
			AccountId:    accountScope.AccountId,
			Name:         newAPIKey.Name,
			CreationTime: now,
			KeyHash:      eventline.HashAPIKey(key),
		}
		if err := apiKey.Insert(conn); err != nil {
			return fmt.Errorf("cannot insert api key: %w", err)
		}

		// TODO send notification

		return nil
	})
	if err != nil {
		return nil, "", err
	}

	return apiKey, key, nil
}

func (s *Service) DeleteAPIKey(keyId eventline.Id, scope eventline.Scope) error {
	return s.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		var key eventline.APIKey

		if err := key.LoadForUpdate(conn, keyId, scope); err != nil {
			return fmt.Errorf("cannot load api key: %w", err)
		}

		if err := key.Delete(conn, scope); err != nil {
			return fmt.Errorf("cannot delete key: %w", err)
		}

		return nil
	})
}
