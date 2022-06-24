package github

import (
	"context"
	"errors"
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/pg"
	"github.com/jackc/pgx/v4"
)

type UnknownSubscriptionError struct {
	Id eventline.Id
}

func (err UnknownSubscriptionError) Error() string {
	return fmt.Sprintf("unknown subscription %q", err.Id)
}

type Subscription struct {
	Id           eventline.Id
	Organization string
	Repository   string // optional
	HookId       HookId // either an org hook or a repo hook
}

func LoadHookIdByParameters(conn pg.Conn, params *Parameters) (*HookId, error) {
	ctx := context.Background()

	repoCond := "TRUE"
	if params.Repository != "" {
		repoCond = "repository = $2"
	}

	query := fmt.Sprintf(`
SELECT hook_id
  FROM c_github_subscriptions
  WHERE organization = $1
    AND %s
  LIMIT 1
`, repoCond)

	args := []interface{}{params.Organization}
	if params.Repository != "" {
		args = append(args, params.Repository)
	}

	var hookId HookId
	err := conn.QueryRow(ctx, query, args...).Scan(&hookId)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &hookId, nil
}

func (s *Subscription) LoadForUpdate(conn pg.Conn, id eventline.Id) error {
	query := `
SELECT id, organization, repository, hook_id
  FROM c_github_subscriptions
  WHERE id = $1;
`
	err := pg.QueryObject(conn, s, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownSubscriptionError{Id: id}
	}

	return err
}

func CountSubscriptionsByHookId(conn pg.Conn, hookId HookId) (int64, error) {
	ctx := context.Background()

	query := `
SELECT COUNT(*)
  FROM c_github_subscriptions
  WHERE hook_id = $1
`
	var count int64
	err := conn.QueryRow(ctx, query, hookId).Scan(&count)
	if err != nil {
		return -1, err
	}

	return count, nil
}

func (s *Subscription) Insert(conn pg.Conn) error {
	query := `
INSERT INTO c_github_subscriptions
    (id, organization, repository, hook_id)
  VALUES
    ($1, $2, $3, $4);
`
	return pg.Exec(conn, query,
		s.Id, s.Organization, s.Repository, s.HookId)
}

func (s *Subscription) Delete(conn pg.Conn) error {
	query := `
DELETE FROM c_github_subscriptions
  WHERE id = $1;
`
	return pg.Exec(conn, query, s.Id)
}

func (s *Subscription) FromRow(row pgx.Row) error {
	return row.Scan(&s.Id, &s.Organization, &s.Repository, &s.HookId)
}
