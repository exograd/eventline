package eventline

import (
	"errors"
	"fmt"
	"time"

	"github.com/exograd/go-daemon/pg"
	"github.com/jackc/pgx/v4"
)

type UnknownStepExecutionError struct {
	Id Id
}

func (err UnknownStepExecutionError) Error() string {
	return fmt.Sprintf("unknown step execution %q", err.Id)
}

type StepExecutionStatus string

const (
	StepExecutionStatusCreated    StepExecutionStatus = "created"
	StepExecutionStatusStarted    StepExecutionStatus = "started"
	StepExecutionStatusAborted    StepExecutionStatus = "aborted"
	StepExecutionStatusSuccessful StepExecutionStatus = "successful"
	StepExecutionStatusFailed     StepExecutionStatus = "failed"
)

var StepExecutionStatusValues = []StepExecutionStatus{
	StepExecutionStatusCreated,
	StepExecutionStatusStarted,
	StepExecutionStatusAborted,
	StepExecutionStatusSuccessful,
	StepExecutionStatusFailed,
}

type StepExecution struct {
	Id             Id                  `json:"id"`
	ProjectId      Id                  `json:"project_id"`
	JobExecutionId Id                  `json:"job_execution_id"`
	Position       int                 `json:"position"`
	Status         StepExecutionStatus `json:"status"`
	StartTime      *time.Time          `json:"start_time,omitempty"`
	EndTime        *time.Time          `json:"end_time,omitempty"`
	FailureMessage string              `json:"failure_message,omitempty"`
	Output         string              `json:"output,omitempty"`
}

type StepExecutions []*StepExecution

func (se *StepExecution) Finished() bool {
	return se.Status != StepExecutionStatusCreated &&
		se.Status != StepExecutionStatusStarted
}

func (se *StepExecution) Duration() *time.Duration {
	if se.StartTime == nil || se.EndTime == nil {
		return nil
	}

	d := se.EndTime.Sub(*se.StartTime)
	return &d
}

func (se *StepExecution) Load(conn pg.Conn, id Id, scope Scope) error {
	query := fmt.Sprintf(`
SELECT id, project_id, job_execution_id, position, status,
       start_time, end_time, failure_message, output
  FROM step_executions
  WHERE %s AND id = $1;
`, scope.SQLCondition())

	err := pg.QueryObject(conn, se, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownStepExecutionError{Id: id}
	}

	return err
}

func (ses *StepExecutions) LoadByJobExecutionId(conn pg.Conn, jeId Id) error {
	query := `
SELECT id, project_id, job_execution_id, position, status,
       start_time, end_time, failure_message, output
  FROM step_executions
  WHERE job_execution_id = $1
  ORDER BY position;
`
	return pg.QueryObjects(conn, ses, query, jeId)
}

func (ses *StepExecutions) LoadByJobExecutionIdForUpdate(conn pg.Conn, jeId Id) error {
	query := `
SELECT id, project_id, job_execution_id, position, status,
       start_time, end_time, failure_message, output
  FROM step_executions
  WHERE job_execution_id = $1
  ORDER BY position
  FOR UPDATE;
`
	return pg.QueryObjects(conn, ses, query, jeId)
}

func (se *StepExecution) Insert(conn pg.Conn) error {
	query := `
INSERT INTO step_executions
    (id, project_id, job_execution_id, position, status, start_time,
     end_time, failure_message, output)
  VALUES
    ($1, $2, $3, $4, $5,
     $6, $7, $8, $9);
`
	return pg.Exec(conn, query,
		se.Id, se.ProjectId, se.JobExecutionId, se.Position, se.Status,
		se.StartTime, se.EndTime, se.FailureMessage, se.Output)
}

func (se *StepExecution) Update(conn pg.Conn) error {
	query := `
UPDATE step_executions SET
    status = $2,
    start_time = $3,
    end_time = $4,
    failure_message = $5
  WHERE id = $1;
`
	return pg.Exec(conn, query,
		se.Id, se.Status, se.StartTime, se.EndTime, se.FailureMessage)
}

func (se *StepExecution) UpdateOutput(conn pg.Conn, data []byte) error {
	query := `
UPDATE step_executions SET
    output = output || $2
  WHERE id = $1;
`
	return pg.Exec(conn, query,
		se.Id, data)
}

func (se *StepExecution) ClearOutput(conn pg.Conn) error {
	query := `
UPDATE step_executions SET
    output = ''
  WHERE id = $1;
`
	return pg.Exec(conn, query, se.Id)
}

func (se *StepExecution) FromRow(row pgx.Row) error {
	return row.Scan(&se.Id, &se.ProjectId, &se.JobExecutionId, &se.Position,
		&se.Status, &se.StartTime, &se.EndTime, &se.FailureMessage, &se.Output)
}

func (ses *StepExecutions) AddFromRow(row pgx.Row) error {
	var se StepExecution
	if err := se.FromRow(row); err != nil {
		return err
	}

	*ses = append(*ses, &se)
	return nil
}
