package time

import (
	"errors"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/jackc/pgx/v5"
	"go.n16f.net/service/pkg/pg"
	"go.n16f.net/uuid"
)

type Subscription struct {
	Id       uuid.UUID
	LastTick *time.Time
	NextTick time.Time
}

func LoadSubscriptionForProcessing(conn pg.Conn) (*Subscription, *eventline.Subscription, error) {
	now := time.Now().UTC()

	query := `
SELECT es.id, s.last_tick, s.next_tick
  FROM subscriptions AS es
  JOIN c_time_subscriptions AS s ON s.id = es.id
  WHERE es.status = 'active'
    AND s.next_tick <= $1
  LIMIT 1
  FOR UPDATE SKIP LOCKED
`
	var s Subscription
	var es eventline.Subscription

	err := pg.QueryObject(conn, &s, query, now)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	} else if err != nil {
		return nil, nil, err
	}

	if err := es.Load(conn, s.Id); err != nil {
		return nil, nil, err
	}

	return &s, &es, nil
}

func (s *Subscription) Insert(conn pg.Conn) error {
	query := `
INSERT INTO c_time_subscriptions
    (id, last_tick, next_tick)
  VALUES
    ($1, $2, $3);
`
	return pg.Exec(conn, query,
		s.Id, s.LastTick, s.NextTick)
}

func (s *Subscription) Update(conn pg.Conn) error {
	query := `
UPDATE c_time_subscriptions SET
    last_tick = $2,
    next_tick = $3
  WHERE id = $1
`
	return pg.Exec(conn, query,
		s.Id, s.LastTick, s.NextTick)
}

func DeleteSubscription(conn pg.Conn, id uuid.UUID) error {
	query := `
DELETE FROM c_time_subscriptions
  WHERE id = $1;
`
	return pg.Exec(conn, query, id)
}

func (s *Subscription) FromRow(row pgx.Row) error {
	return row.Scan(&s.Id, &s.LastTick, &s.NextTick)
}
