package eventline

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"go.n16f.net/ejson"
	"go.n16f.net/program"
	"go.n16f.net/service/pkg/pg"
	"go.n16f.net/uuid"
)

var EventSorts Sorts = Sorts{
	Sorts: map[string]string{
		"id":         "id",
		"event_time": "event_time",
	},

	Default: "event_time",
}

type UnknownEventError struct {
	Id uuid.UUID
}

func (err UnknownEventError) Error() string {
	return fmt.Sprintf("unknown event %q", err.Id)
}

type NewEvent struct {
	EventTime time.Time       `json:"event_time"`
	Connector string          `json:"connector"`
	Name      string          `json:"name"`
	Data      EventData       `json:"-"`
	RawData   json.RawMessage `json:"data"`
}

type Event struct {
	Id              uuid.UUID   `json:"id"`
	ProjectId       uuid.UUID   `json:"project_id"`
	JobId           uuid.UUID   `json:"job_id"`
	CreationTime    time.Time   `json:"creation_time"`
	EventTime       time.Time   `json:"event_time"`
	Connector       string      `json:"connector"`
	Name            string      `json:"name"`
	Data            EventData   `json:"data"`
	DataValue       interface{} `json:"-"`
	Processed       bool        `json:"processed,omitempty"`
	OriginalEventId *uuid.UUID  `json:"original_event_id,omitempty"`
}

type Events []*Event

func (ne *NewEvent) ValidateJSON(v *ejson.Validator) {
	if CheckConnectorName(v, "connector", ne.Connector) {
		CheckEventName(v, "name", ne.Connector, ne.Name)
	}
}

func (e *Event) SortKey(sort string) (key string) {
	switch sort {
	case "id":
		key = e.Id.String()
	case "event_time":
		key = e.EventTime.Format(time.RFC3339)
	default:
		program.Panic("unknown event sort %q", sort)
	}

	return
}

func (pne *NewEvent) UnmarshalJSON(data []byte) error {
	type NewEvent2 NewEvent

	ne := NewEvent2(*pne)

	if err := json.Unmarshal(data, &ne); err != nil {
		return err
	}

	if ConnectorExists(ne.Connector) && EventExists(ne.Connector, ne.Name) {
		cdef := GetConnectorDef(ne.Connector)
		edef := cdef.Event(ne.Name)

		edata, err := edef.DecodeData(ne.RawData)
		if err != nil {
			return fmt.Errorf("cannot decode data: %w", err)
		}

		ne.Data = edata
	}

	*pne = NewEvent(ne)
	return nil
}

func (e *Event) Def() *EventDef {
	cdef := GetConnectorDef(e.Connector)
	return cdef.Event(e.Name)
}

func (e *Event) Load(conn pg.Conn, id uuid.UUID, scope Scope) error {
	query := fmt.Sprintf(`
SELECT id, project_id, job_id, creation_time, event_time,
       connector, name, data, processed, original_event_id
  FROM events
  WHERE %s AND id = $1
`, scope.SQLCondition())

	err := pg.QueryObject(conn, e, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownEventError{Id: id}
	}

	return err
}

func LoadEventForProcessing(conn pg.Conn) (*Event, error) {
	query := `
SELECT id, project_id, job_id, creation_time, event_time,
       connector, name, data, processed, original_event_id
  FROM events
  WHERE processed = FALSE and job_id IS NOT NULL
  LIMIT 1
  FOR UPDATE SKIP LOCKED;
`
	var e Event
	err := pg.QueryObject(conn, &e, query)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &e, nil
}

func LoadEventPage(conn pg.Conn, cursor *Cursor, scope Scope) (*Page, error) {
	query := fmt.Sprintf(`
SELECT id, project_id, job_id, creation_time, event_time,
       connector, name, data, processed, original_event_id
  FROM events
  WHERE %s AND %s
`, scope.SQLCondition(), cursor.SQLConditionOrderLimit(EventSorts))

	var events Events
	if err := pg.QueryObjects(conn, &events, query); err != nil {
		return nil, err
	}

	return events.Page(cursor), nil
}

func (e *Event) Insert(conn pg.Conn) error {
	query := `
INSERT INTO events
    (id, project_id, job_id, creation_time, event_time,
     connector, name, data, processed, original_event_id)
  VALUES
    ($1, $2, $3, $4, $5,
     $6, $7, $8, $9, $10);
`

	return pg.Exec(conn, query,
		e.Id, e.ProjectId, e.JobId, e.CreationTime, e.EventTime,
		e.Connector, e.Name, e.Data, e.Processed, e.OriginalEventId)
}

func (e *Event) Update(conn pg.Conn) error {
	query := `
UPDATE events SET
    processed = $2
  WHERE id = $1
`

	return pg.Exec(conn, query,
		e.Id, e.Processed)
}

func (es Events) Page(cursor *Cursor) *Page {
	elements := make([]PageElement, len(es))
	for i, e := range es {
		elements[i] = e
	}

	return NewPage(cursor, elements, EventSorts)
}

func (e *Event) FromRow(row pgx.Row) error {
	var rawData []byte

	err := row.Scan(&e.Id, &e.ProjectId, &e.JobId, &e.CreationTime, &e.EventTime,
		&e.Connector, &e.Name, &rawData, &e.Processed, &e.OriginalEventId)
	if err != nil {
		return err
	}

	edef := e.Def()
	edata, err := edef.DecodeData(rawData)
	if err != nil {
		return fmt.Errorf("cannot decode data: %w", err)
	}
	e.Data = edata

	// Used for filter evaluation
	if err := json.Unmarshal(rawData, &e.DataValue); err != nil {
		return fmt.Errorf("cannot decode data: %w", err)
	}

	return nil
}

func (is *Events) AddFromRow(row pgx.Row) error {
	var i Event
	if err := i.FromRow(row); err != nil {
		return err
	}

	*is = append(*is, &i)
	return nil
}
