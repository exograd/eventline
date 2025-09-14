package eventline

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"go.n16f.net/service/pkg/pg"
	"go.n16f.net/uuid"
)

type Notification struct {
	Id               uuid.UUID
	ProjectId        uuid.UUID
	Recipients       []string
	Message          []byte
	NextDeliveryTime time.Time
	DeliveryDelay    int // seconds
}

func LoadNotificationForDelivery(conn pg.Conn) (*Notification, error) {
	now := time.Now().UTC()

	query := `
SELECT id, project_id, recipients, message, next_delivery_time,
       delivery_delay
  FROM notifications
  WHERE next_delivery_time < $1
  ORDER BY next_delivery_time
  LIMIT 1
  FOR UPDATE SKIP LOCKED;
`
	var n Notification
	err := pg.QueryObject(conn, &n, query, now)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &n, nil
}

func (n *Notification) Insert(conn pg.Conn) error {
	query := `
INSERT INTO notifications
    (id, project_id, recipients, message, next_delivery_time,
     delivery_delay)
  VALUES
    ($1, $2, $3, $4, $5,
     $6);
`
	return pg.Exec(conn, query,
		n.Id, n.ProjectId, n.Recipients, n.Message, n.NextDeliveryTime,
		n.DeliveryDelay)
}

func (n *Notification) Update(conn pg.Conn) error {
	query := `
UPDATE notifications SET
    next_delivery_time = $2,
    delivery_delay = $3
  WHERE id = $1
`
	return pg.Exec(conn, query,
		n.Id, n.NextDeliveryTime, n.DeliveryDelay)
}

func (n *Notification) Delete(conn pg.Conn) error {
	query := `
DELETE FROM notifications
  WHERE id = $1;
`
	return pg.Exec(conn, query, n.Id)
}

func (n *Notification) FromRow(row pgx.Row) error {
	return row.Scan(&n.Id, &n.ProjectId, &n.Recipients, &n.Message,
		&n.NextDeliveryTime, &n.DeliveryDelay)
}
