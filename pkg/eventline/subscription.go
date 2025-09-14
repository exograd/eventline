package eventline

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"go.n16f.net/service/pkg/pg"
	"go.n16f.net/uuid"
)

type ExternalSubscriptionError struct {
	Err error
}

func NewExternalSubscriptionError(err error) *ExternalSubscriptionError {
	return &ExternalSubscriptionError{Err: err}
}

func (err ExternalSubscriptionError) Error() string {
	return err.Err.Error()
}

type UnknownSubscriptionError struct {
	Id uuid.UUID
}

func (err UnknownSubscriptionError) Error() string {
	return fmt.Sprintf("unknown subscription %q", err.Id)
}

type UnknownJobSubscriptionError struct {
	JobId uuid.UUID
}

func (err UnknownJobSubscriptionError) Error() string {
	return fmt.Sprintf("unknown subscription for job %q", err.JobId)
}

type SubscriptionStatus string

const (
	SubscriptionStatusInactive    SubscriptionStatus = "inactive"
	SubscriptionStatusActive      SubscriptionStatus = "active"
	SubscriptionStatusTerminating SubscriptionStatus = "terminating"
)

type Subscription struct {
	Id             uuid.UUID
	ProjectId      *uuid.UUID
	JobId          *uuid.UUID
	IdentityId     *uuid.UUID
	Connector      string
	Event          string
	Parameters     SubscriptionParameters
	CreationTime   time.Time
	Status         SubscriptionStatus
	UpdateDelay    int // seconds
	LastUpdateTime *time.Time
	NextUpdateTime *time.Time
}

type Subscriptions []*Subscription

func (s *Subscription) EventDef() *EventDef {
	cdef := GetConnectorDef(s.Connector)
	return cdef.Event(s.Event)
}

func (s *Subscription) NewEvent(cname, ename string, etime *time.Time, data EventData) *Event {
	now := time.Now().UTC()
	if etime == nil {
		etime = &now
	}

	return &Event{
		Id:           uuid.MustGenerate(uuid.V7),
		ProjectId:    *s.ProjectId,
		JobId:        *s.JobId,
		CreationTime: now,
		EventTime:    *etime,
		Connector:    cname,
		Name:         ename,
		Data:         data,
	}
}

func (s *Subscription) Load(conn pg.Conn, id uuid.UUID) error {
	query := `
SELECT id, project_id, job_id, identity_id, connector, event, parameters,
       creation_time, status, update_delay, last_update_time, next_update_time
  FROM subscriptions
  WHERE id = $1
`
	err := pg.QueryObject(conn, s, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownSubscriptionError{Id: id}
	}

	return err
}

func (ss *Subscriptions) LoadAllForUpdate(conn pg.Conn, scope Scope) error {
	query := fmt.Sprintf(`
SELECT id, project_id, job_id, identity_id, connector, event, parameters,
       creation_time, status, update_delay, last_update_time, next_update_time
  FROM subscriptions
  WHERE %s
  FOR UPDATE;
`, scope.SQLCondition())

	return pg.QueryObjects(conn, ss, query)
}

func (s *Subscription) LoadByJobForUpdate(conn pg.Conn, jobId uuid.UUID, scope Scope) error {
	query := fmt.Sprintf(`
SELECT id, project_id, job_id, identity_id, connector, event, parameters,
       creation_time, status, update_delay, last_update_time, next_update_time
  FROM subscriptions
  WHERE %s AND job_id = $1
  FOR UPDATE;
`, scope.SQLCondition())

	err := pg.QueryObject(conn, s, query, jobId)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownJobSubscriptionError{JobId: jobId}
	}

	return err
}

func LoadSubscriptionForProcessing(conn pg.Conn) (*Subscription, error) {
	now := time.Now().UTC()

	query := `
SELECT id, project_id, job_id, identity_id, connector, event, parameters,
       creation_time, status, update_delay, last_update_time, next_update_time
  FROM subscriptions
  WHERE status = 'inactive' OR status = 'terminating'
    AND next_update_time < $1
  ORDER BY op
  LIMIT 1
  FOR UPDATE SKIP LOCKED;
`
	var s Subscription

	err := pg.QueryObject(conn, &s, query, now)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &s, nil
}

func (s *Subscription) Insert(conn pg.Conn) error {
	query := `
INSERT INTO subscriptions
    (id, project_id, job_id, identity_id, connector, event, parameters,
     creation_time, status, update_delay, last_update_time, next_update_time)
  VALUES
    ($1, $2, $3, $4, $5, $6, $7,
     $8, $9, $10, $11, $12);
`
	return pg.Exec(conn, query,
		s.Id, s.ProjectId, s.JobId, s.IdentityId,
		s.Connector, s.Event, s.Parameters, s.CreationTime, s.Status,
		s.UpdateDelay, s.LastUpdateTime, s.NextUpdateTime)
}

func (s *Subscription) Update(conn pg.Conn) error {
	query := `
UPDATE subscriptions SET
    project_id = $2,
    job_id = $3,
    identity_id = $4,
    status = $5,
    update_delay = $6,
    last_update_time = $7,
    next_update_time = $8
  WHERE id = $1
`
	return pg.Exec(conn, query,
		s.Id, s.ProjectId, s.JobId, s.IdentityId, s.Status, s.UpdateDelay,
		s.LastUpdateTime, s.NextUpdateTime)
}

func (s *Subscription) UpdateOp(conn pg.Conn) error {
	query := `
UPDATE subscriptions SET
    op = nextval('subscription_op')
  WHERE id = $1
`

	return pg.Exec(conn, query, s.Id)
}

func (s *Subscription) Delete(conn pg.Conn) error {
	query := `
DELETE FROM subscriptions
  WHERE id = $1;
`
	return pg.Exec(conn, query, s.Id)
}

func (s *Subscription) FromRow(row pgx.Row) error {
	var rawParameters json.RawMessage

	err := row.Scan(&s.Id, &s.ProjectId, &s.JobId, &s.IdentityId,
		&s.Connector, &s.Event, &rawParameters, &s.CreationTime, &s.Status,
		&s.UpdateDelay, &s.LastUpdateTime, &s.NextUpdateTime)
	if err != nil {
		return err
	}

	edef := s.EventDef()
	parameters, err := edef.DecodeSubscriptionParameters(rawParameters)
	if err != nil {
		return fmt.Errorf("cannot decode parameters: %w", err)
	}
	s.Parameters = parameters

	return nil
}

func (ss *Subscriptions) AddFromRow(row pgx.Row) error {
	var s Subscription
	if err := s.FromRow(row); err != nil {
		return err
	}

	*ss = append(*ss, &s)
	return nil
}
