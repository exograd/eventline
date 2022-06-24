package eventline

import (
	"errors"
	"fmt"
	"time"

	"github.com/exograd/eventline/pkg/utils"
	"github.com/exograd/go-daemon/pg"
	"github.com/jackc/pgx/v4"
)

var JobExecutionSorts Sorts = Sorts{
	Sorts: map[string]string{
		"id":             "id",
		"scheduled_time": "scheduled_time",
	},

	Default: "scheduled_time",
}

type JobExecutionPageOptions struct {
	JobId *Id
}

type UnknownJobExecutionError struct {
	Id Id
}

func (err UnknownJobExecutionError) Error() string {
	return fmt.Sprintf("unknown job execution %q", err.Id)
}

type JobExecutionStatus string

const (
	JobExecutionStatusCreated    JobExecutionStatus = "created"
	JobExecutionStatusStarted    JobExecutionStatus = "started"
	JobExecutionStatusAborted    JobExecutionStatus = "aborted"
	JobExecutionStatusSuccessful JobExecutionStatus = "successful"
	JobExecutionStatusFailed     JobExecutionStatus = "failed"
)

var JobExecutionStatusValues = []JobExecutionStatus{
	JobExecutionStatusCreated,
	JobExecutionStatusStarted,
	JobExecutionStatusAborted,
	JobExecutionStatusSuccessful,
	JobExecutionStatusFailed,
}

type JobExecution struct {
	Id             Id                     `json:"id"`
	ProjectId      Id                     `json:"project_id"`
	JobId          Id                     `json:"job_id"`
	JobSpec        *JobSpec               `json:"job_spec"`
	EventId        *Id                    `json:"event_id,omitempty"`
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
	CreationTime   time.Time              `json:"creation_time"`
	UpdateTime     time.Time              `json:"update_time"`
	ScheduledTime  time.Time              `json:"scheduled_time,omitempty"`
	Status         JobExecutionStatus     `json:"status"`
	StartTime      *time.Time             `json:"start_time,omitempty"`
	EndTime        *time.Time             `json:"end_time,omitempty"`
	RefreshTime    *time.Time             `json:"refresh_time,omitempty"`
	ExpirationTime *time.Time             `json:"expiration_time,omitempty"`
	FailureMessage string                 `json:"failure_message,omitempty"`
}

type JobExecutions []*JobExecution

func (je *JobExecution) SortKey(sort string) (key string) {
	switch sort {
	case "id":
		key = je.Id.String()
	case "scheduled_time":
		key = je.ScheduledTime.Format(time.RFC3339)
	default:
		utils.Panicf("unknown job execution sort %q", sort)
	}

	return
}

func (je *JobExecution) Duration() *time.Duration {
	if je.StartTime == nil || je.EndTime == nil {
		return nil
	}

	d := je.EndTime.Sub(*je.StartTime)
	return &d
}

func (je *JobExecution) Finished() bool {
	return je.Status != JobExecutionStatusCreated &&
		je.Status != JobExecutionStatusStarted
}

func (je *JobExecution) Load(conn pg.Conn, id Id, scope Scope) error {
	query := fmt.Sprintf(`
SELECT id, project_id, job_id, job_spec, event_id, parameters,
       creation_time, update_time, scheduled_time, status, start_time,
       end_time, refresh_time, expiration_time, failure_message
  FROM job_executions
  WHERE %s AND id = $1;
`, scope.SQLCondition())

	err := pg.QueryObject(conn, je, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownJobExecutionError{Id: id}
	}

	return err
}

func (je *JobExecution) LoadForUpdate(conn pg.Conn, id Id, scope Scope) error {
	query := fmt.Sprintf(`
SELECT id, project_id, job_id, job_spec, event_id, parameters,
       creation_time, update_time, scheduled_time, status, start_time,
       end_time, refresh_time, expiration_time, failure_message
  FROM job_executions
  WHERE %s AND id = $1
  FOR UPDATE;
`, scope.SQLCondition())

	err := pg.QueryObject(conn, je, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownJobExecutionError{Id: id}
	}

	return err
}

func LoadJobExecutionForScheduling(conn pg.Conn) (*JobExecution, error) {
	query := `
SELECT je1.id, je1.project_id, je1.job_id, je1.job_spec, je1.event_id,
       je1.parameters, je1.creation_time, je1.update_time, je1.scheduled_time,
       je1.status, je1.start_time, je1.end_time, je1.refresh_time,
       je1.expiration_time, je1.failure_message
  FROM job_executions AS je1
  WHERE je1.status = 'created'
    AND (((je1.job_spec->'concurrent')::BOOLEAN IS TRUE)
         OR
         (NOT EXISTS
           (SELECT 1
              FROM job_executions AS je2
              WHERE je2.job_id = je1.job_id
                AND je2.id <> je1.id
                AND je2.status = 'started')))
  ORDER BY scheduled_time
  LIMIT 1
  FOR UPDATE SKIP LOCKED;
`
	var je JobExecution
	err := pg.QueryObject(conn, &je, query)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &je, nil
}

func (jes *JobExecutions) LoadByEvent(conn pg.Conn, eventId Id) error {
	query := `
SELECT id, project_id, job_id, job_spec, event_id, parameters,
       creation_time, update_time, scheduled_time, status, start_time,
       end_time, refresh_time, expiration_time, failure_message
  FROM job_executions
  WHERE event_id = $1
  ORDER BY scheduled_time DESC;
`
	return pg.QueryObjects(conn, jes, query, eventId)
}

func LoadLastJobExecutions(conn pg.Conn, jobIds Ids, scope Scope) (map[Id]*JobExecution, error) {
	query := fmt.Sprintf(`
WITH ranked_jobs AS
       (SELECT id, project_id, job_id, job_spec, event_id, parameters,
               creation_time, update_time, scheduled_time, status, start_time,
               end_time, refresh_time, expiration_time, failure_message,
               row_number() OVER (PARTITION BY job_id ORDER BY id DESC) AS rank
          FROM job_executions
          WHERE %s AND job_id = ANY ($1))
  SELECT id, project_id, job_id, job_spec, event_id, parameters,
         creation_time, update_time, scheduled_time, status, start_time,
         end_time, refresh_time, expiration_time, failure_message
    FROM ranked_jobs
    WHERE rank = 1;
`, scope.SQLCondition())

	var jobExecutions JobExecutions
	err := pg.QueryObjects(conn, &jobExecutions, query, jobIds)
	if err != nil {
		return nil, err
	}

	table := make(map[Id]*JobExecution)
	for _, je := range jobExecutions {
		table[je.JobId] = je
	}

	return table, nil
}

func LoadJobExecutionPage(conn pg.Conn, options JobExecutionPageOptions, cursor *Cursor, scope Scope) (*Page, error) {
	jobCond := "TRUE"
	if options.JobId != nil {
		jobId := *options.JobId
		jobCond = fmt.Sprintf("job_id=" + pg.QuoteString(jobId.String()))
	}

	query := fmt.Sprintf(`
SELECT id, project_id, job_id, job_spec, event_id, parameters,
       creation_time, update_time, scheduled_time, status, start_time,
       end_time, refresh_time, expiration_time, failure_message
  FROM job_executions
  WHERE %s AND %s AND %s;
`, scope.SQLCondition(), jobCond,
		cursor.SQLConditionOrderLimit(JobExecutionSorts))

	var jes JobExecutions
	if err := pg.QueryObjects(conn, &jes, query); err != nil {
		return nil, err
	}

	return jes.Page(cursor), nil
}

func (je *JobExecution) Insert(conn pg.Conn) error {
	var parameters interface{}
	if je.Parameters != nil {
		parameters = je.Parameters
	}

	query := `
INSERT INTO job_executions
    (id, project_id, job_id, job_spec, event_id, parameters,
     creation_time, update_time, scheduled_time, status, start_time,
     end_time, refresh_time, expiration_time, failure_message)
  VALUES
    ($1, $2, $3, $4, $5, $6,
     $7, $8, $9, $10, $11,
     $12, $13, $14, $15);
`
	return pg.Exec(conn, query,
		je.Id, je.ProjectId, je.JobId, je.JobSpec, je.EventId, parameters,
		je.CreationTime, je.UpdateTime, je.ScheduledTime, je.Status,
		je.StartTime, je.EndTime, je.RefreshTime, je.ExpirationTime,
		je.FailureMessage)
}

func (je *JobExecution) Update(conn pg.Conn) error {
	query := `
UPDATE job_executions SET
    update_time = $2,
    status = $3,
    start_time = $4,
    end_time = $5,
    refresh_time = $6,
    expiration_time = $7,
    failure_message = $8
  WHERE id = $1;
`
	return pg.Exec(conn, query,
		je.Id, je.UpdateTime, je.Status, je.StartTime, je.EndTime,
		je.RefreshTime, je.ExpirationTime, je.FailureMessage)
}

func (jes JobExecutions) Page(cursor *Cursor) *Page {
	elements := make([]PageElement, len(jes))
	for i, je := range jes {
		elements[i] = je
	}

	return NewPage(cursor, elements, JobExecutionSorts)
}

func (je *JobExecution) FromRow(row pgx.Row) error {
	var eventId Id

	err := row.Scan(&je.Id, &je.ProjectId, &je.JobId, &je.JobSpec, &eventId,
		&je.Parameters, &je.CreationTime, &je.UpdateTime, &je.ScheduledTime,
		&je.Status, &je.StartTime, &je.EndTime, &je.RefreshTime,
		&je.ExpirationTime, &je.FailureMessage)
	if err != nil {
		return err
	}

	if !eventId.IsZero() {
		je.EventId = &eventId
	}

	return nil
}

func (jes *JobExecutions) AddFromRow(row pgx.Row) error {
	var je JobExecution
	if err := je.FromRow(row); err != nil {
		return err
	}

	*jes = append(*jes, &je)
	return nil
}
