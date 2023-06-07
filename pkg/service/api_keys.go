package service

import (
	"fmt"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/galdor/go-service/pkg/pg"
	"github.com/galdor/go-uuid"
)

type DuplicateAPIKeyNameError struct {
	Name string
}

func (err DuplicateAPIKeyNameError) Error() string {
	return fmt.Sprintf("duplicate api key name %q", err.Name)
}

func (s *Service) CreateAPIKey(newAPIKey *eventline.NewAPIKey, scope eventline.Scope) (*eventline.APIKey, string, error) {
	var apiKey *eventline.APIKey
	var key uuid.UUID
	var keyString string

	accountScope := scope.(*eventline.AccountScope)

	err := s.Pg.WithTx(func(conn pg.Conn) error {
		now := time.Now().UTC()

		exists, err := eventline.APIKeyNameExists(conn, newAPIKey.Name, scope)
		if err != nil {
			return fmt.Errorf("cannot check api key name existence: %w", err)
		} else if exists {
			return &DuplicateAPIKeyNameError{Name: newAPIKey.Name}
		}

		if err := key.Generate(uuid.V4); err != nil {
			return fmt.Errorf("cannot generate uuid: %w", err)
		}

		keyString = key.String()

		apiKey = &eventline.APIKey{
			Id:           eventline.GenerateId(),
			AccountId:    accountScope.AccountId,
			Name:         newAPIKey.Name,
			CreationTime: now,
			KeyHash:      eventline.HashAPIKey(keyString),
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

	return apiKey, keyString, nil
}

func (s *Service) DeleteAPIKey(keyId eventline.Id, scope eventline.Scope) error {
	return s.Pg.WithTx(func(conn pg.Conn) error {
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
