package eventline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"go.n16f.net/service/pkg/pg"
	"go.n16f.net/uuid"
)

type UnknownSessionError struct {
	Id uuid.UUID
}

func (err UnknownSessionError) Error() string {
	return fmt.Sprintf("unknown session %q", err.Id)
}

type NewSession struct {
	Data            *SessionData     `json:"data"`
	AccountRole     AccountRole      `json:"account_role"`
	AccountSettings *AccountSettings `json:"account_settings"`
}

type Session struct {
	Id              uuid.UUID        `json:"id"`
	AccountId       uuid.UUID        `json:"account_id"`
	CreationTime    time.Time        `json:"creation_time"`
	UpdateTime      time.Time        `json:"update_time"`
	Data            *SessionData     `json:"data"`
	AccountRole     AccountRole      `json:"account_role"`
	AccountSettings *AccountSettings `json:"account_settings"`
}

type SessionData struct {
	ProjectId *uuid.UUID `json:"project_id,omitempty"`
}

func (s *Session) LoadUpdate(conn pg.Conn, id uuid.UUID) error {
	now := time.Now().UTC()

	query := `
UPDATE sessions SET
    update_time = $2
  WHERE id = $1
  RETURNING
    id, account_id, creation_time, update_time, data,
    account_role, account_settings;
`
	err := pg.QueryObject(conn, s, query, id, now)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownSessionError{Id: id}
	}

	return err
}

func (s *Session) Insert(conn pg.Conn) error {
	query := `
INSERT INTO sessions
    (id, account_id, creation_time, update_time, data, account_role,
     account_settings)
  VALUES
    ($1, $2, $3, $4, $5, $6,
     $7);
`
	return pg.Exec(conn, query,
		s.Id, s.AccountId, s.CreationTime, s.UpdateTime, s.Data, s.AccountRole,
		s.AccountSettings)
}

func (s *Session) UpdateData(conn pg.Conn) error {
	query := `
UPDATE sessions SET
    data = $2
  WHERE id = $1
`
	return pg.Exec(conn, query, s.Id, s.Data)
}

func (s *Session) UpdateAccountSettings(conn pg.Conn) error {
	query := `
UPDATE sessions SET
    account_settings = $2
  WHERE id = $1
`
	return pg.Exec(conn, query, s.Id, s.AccountSettings)
}

func UpdateSessionsForProjectDeletion(conn pg.Conn, projectId uuid.UUID) error {
	query := `
UPDATE sessions SET
    data = data - 'project_id'
  WHERE data->>'project_id' = $1
`
	return pg.Exec(conn, query, projectId)
}

func (s *Session) Delete(conn pg.Conn) error {
	query := `
DELETE FROM sessions
  WHERE id = $1;
`
	return pg.Exec(conn, query, s.Id)
}

func DeleteOldSessions(conn pg.Conn, retention int) (int64, error) {
	ctx := context.Background()

	now := time.Now().UTC()
	minDate := now.AddDate(0, 0, -retention)

	query := `
DELETE FROM sessions
  WHERE creation_time < $1
`
	res, err := conn.Exec(ctx, query, minDate)
	if err != nil {
		return -1, err
	}

	return res.RowsAffected(), nil
}

func (s *Session) FromRow(row pgx.Row) error {
	var data SessionData
	var accountSettings AccountSettings

	err := row.Scan(&s.Id, &s.AccountId, &s.CreationTime, &s.UpdateTime,
		&data, &s.AccountRole, &accountSettings)
	if err != nil {
		return err
	}

	s.Data = &data
	s.AccountSettings = &accountSettings

	return nil
}
