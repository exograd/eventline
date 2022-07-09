package eventline

import (
	"errors"
	"fmt"
	"time"

	"github.com/exograd/go-daemon/pg"
	"github.com/jackc/pgx/v4"
)

type UnknownSessionError struct {
	Id Id
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
	Id              Id               `json:"id"`
	AccountId       Id               `json:"account_id"`
	CreationTime    time.Time        `json:"creation_time"`
	UpdateTime      time.Time        `json:"update_time"`
	Data            *SessionData     `json:"data"`
	AccountRole     AccountRole      `json:"account_role"`
	AccountSettings *AccountSettings `json:"account_settings"`
}

type SessionData struct {
	ProjectId *Id `json:"project_id,omitempty"`
}

func (s *Session) LoadUpdate(conn pg.Conn, id Id) error {
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

func UpdateSessionsForProjectDeletion(conn pg.Conn, projectId Id) error {
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
