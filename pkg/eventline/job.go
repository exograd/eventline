package eventline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/exograd/evgo/pkg/utils"
	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/djson"
	"github.com/exograd/go-daemon/pg"
	"github.com/jackc/pgx/v4"
	"gopkg.in/yaml.v3"
)

var JobSorts Sorts = Sorts{
	Sorts: map[string]string{
		"id":   "id",
		"name": "spec->>'name'",
	},

	Default: "name",
}

type JobPageOptions struct {
	ExcludeFavouriteJobAccountId *Id
}

type UnknownJobError struct {
	Id Id
}

func (err UnknownJobError) Error() string {
	return fmt.Sprintf("unknown job %q", err.Id)
}

type UnknownJobNameError struct {
	Name string
}

func (err UnknownJobNameError) Error() string {
	return fmt.Sprintf("unknown job %q", err.Name)
}

type StepFailureAction string

const (
	StepFailureActionAbort    StepFailureAction = "abort"
	StepFailureActionContinue StepFailureAction = "continue"
)

var StepFailureActionValues = []StepFailureAction{
	StepFailureActionAbort,
	StepFailureActionContinue,
}

type Job struct {
	Id           Id        `json:"id"`
	ProjectId    Id        `json:"project_id"`
	CreationTime time.Time `json:"creation_time"`
	UpdateTime   time.Time `json:"update_time"`
	Disabled     bool      `json:"disabled,omitempty"`
	Spec         *JobSpec  `json:"spec"`
}

type Jobs []*Job

type JobSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	Trigger    *Trigger   `json:"trigger,omitempty"`
	Parameters Parameters `json:"parameters,omitempty"`

	Runtime    *Runtime `json:"runtime"`
	Concurrent bool     `json:"concurrent,omitempty"`

	Retention int `json:"retention,omitempty"` // days

	Identities  []string          `json:"identities,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Steps       Steps             `json:"steps"`
}

type Trigger struct {
	Connector     string                 `json:"connector"`
	Event         string                 `json:"event"`
	Parameters    SubscriptionParameters `json:"-"`
	RawParameters json.RawMessage        `json:"parameters,omitempty"`
	Identity      string                 `json:"identity,omitempty"`
	Filters       Filters                `json:"filters,omitempty"`
}

type Step struct {
	Label string `json:"label,omitempty"`

	Code    string       `json:"code,omitempty"`
	Command *StepCommand `json:"command,omitempty"`
	Script  *StepScript  `json:"script,omitempty"`
	Bundle  *StepBundle  `json:"bundle,omitempty"`

	OnFailure StepFailureAction `json:"on_failure,omitempty"`
}

type Steps []*Step

type StepCommand struct {
	Name      string   `json:"name"`
	Arguments []string `json:"arguments,omitempty"`
}

type StepScript struct {
	Path      string   `json:"path"`
	Arguments []string `json:"arguments,omitempty"`

	Content string `json:"content"` // content of the script file
}

type StepBundle struct {
	Path      string   `json:"path"`
	Exclude   []string `json:"exclude,omitempty"`
	Command   string   `json:"command"`
	Arguments []string `json:"arguments,omitempty"`

	Files []StepBundleFile `json:"files,omitempty"`
}

func (sb *StepBundle) PathAndCommand() string {
	// Used in templates
	return path.Join(sb.Path, sb.Command)
}

type StepBundleFile struct {
	Name    string      `json:"name"`
	Mode    os.FileMode `json:"mode"`
	Content string      `json:"content"`
}

func (j *Job) SortKey(sort string) (key string) {
	switch sort {
	case "id":
		key = j.Id.String()
	case "name":
		key = j.Spec.Name
	default:
		utils.Panicf("unknown job sort %q", sort)
	}

	return
}

func (spec JobSpec) Check(c *check.Checker) {
	CheckName(c, "name", spec.Name)
	if spec.Description != "" {
		CheckDescription(c, "description", spec.Description)
	}

	c.CheckOptionalObject("trigger", spec.Trigger)
	c.CheckObjectArray("parameters", spec.Parameters)

	c.CheckOptionalObject("runtime", spec.Runtime)

	if spec.Retention != 0 {
		c.CheckIntMin("retention", spec.Retention, 1)
	}

	c.WithChild("identities", func() {
		for i, iname := range spec.Identities {
			CheckName(c, i, iname)
		}
	})

	c.CheckObjectArray("steps", spec.Steps)
	for i, s := range spec.Steps {
		if s.Label == "" {
			s.Label = "Step " + strconv.Itoa(i+1)
		}
	}
}

func (t *Trigger) Check(c *check.Checker) {
	if CheckConnectorName(c, "connector", t.Connector) {
		CheckEventName(c, "event", t.Connector, t.Event)
	}

	c.CheckOptionalObject("parameters", t.Parameters)

	c.CheckObjectArray("filters", t.Filters)

}

func (pt *Trigger) MarshalJSON() ([]byte, error) {
	type Trigger2 Trigger

	t := Trigger2(*pt)

	if t.Parameters != nil {
		parametersData, err := json.Marshal(t.Parameters)
		if err != nil {
			return nil, fmt.Errorf("cannot encode parameters: %w", err)
		}

		t.RawParameters = parametersData
	}

	return json.Marshal(t)
}

func (pt *Trigger) UnmarshalJSON(data []byte) error {
	type Trigger2 Trigger

	t := Trigger2(*pt)
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	// If the connector or event are invalid, let validation
	// (Trigger.Check) signal the error.
	if t.RawParameters != nil && EventExists(t.Connector, t.Event) {
		cdef := GetConnectorDef(t.Connector)
		edef := cdef.Event(t.Event)

		parameters, err := edef.DecodeSubscriptionParameters(t.RawParameters)
		if err != nil {
			return fmt.Errorf("cannot decode parameters: %w", err)
		}

		t.Parameters = parameters
	}

	*pt = Trigger(t)
	return nil
}

func (s *Step) Check(c *check.Checker) {
	if s.Label != "" {
		CheckLabel(c, "label", s.Label)
	}

	n := 0
	if s.Code != "" {
		n += 1
	}
	if s.Command != nil {
		n += 1
	}
	if s.Bundle != nil {
		n += 1
	}
	if s.Script != nil {
		n += 1
	}

	if n == 0 {
		c.AddError(djson.Pointer{}, "missing_step_content",
			"missing code, command, bundle or script member")
	} else if n > 1 {
		c.AddError(djson.Pointer{}, "multiple_step_contents",
			"multiple code, command, bundle or script members")
	}

	c.CheckOptionalObject("command", s.Command)
	c.CheckOptionalObject("script", s.Script)
	c.CheckOptionalObject("bundle", s.Bundle)
}

func (s *StepCommand) Check(c *check.Checker) {
	c.CheckStringNotEmpty("name", s.Name)
}

func (s *StepScript) Check(c *check.Checker) {
	c.CheckStringNotEmpty("path", s.Path)
}

func (s *StepBundle) Check(c *check.Checker) {
	c.CheckStringNotEmpty("path", s.Path)

	c.WithChild("exclude", func() {
		for i, pattern := range s.Exclude {
			c.CheckStringNotEmpty(i, pattern)
		}
	})

	c.CheckStringNotEmpty("command", s.Command)
}

func (spec *JobSpec) ParseYAML(data []byte) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))

	var yamlValue interface{}
	if err := decoder.Decode(&yamlValue); err != nil {
		return err
	}

	jsonValue, err := utils.YAMLValueToJSONValue(yamlValue)
	if err != nil {
		return fmt.Errorf("invalid yaml data: %w", err)
	}

	jsonData, err := json.Marshal(jsonValue)
	if err != nil {
		return fmt.Errorf("cannot encode json data: %w", err)
	}

	if err := json.Unmarshal(jsonData, spec); err != nil {
		return fmt.Errorf("cannot decode json data: %w", err)
	}

	return nil
}

func (spec *JobSpec) IdentityNames() []string {
	var names []string

	if spec.Trigger != nil && spec.Trigger.Identity != "" {
		names = append(names, spec.Trigger.Identity)
	}

	names = append(names, spec.Identities...)

	return names
}

func (j *Job) Load(conn pg.Conn, id Id, scope Scope) error {
	query := fmt.Sprintf(`
SELECT id, project_id, creation_time, update_time, disabled, spec
  FROM jobs
  WHERE %s AND id = $1;
`, scope.SQLCondition())

	err := pg.QueryObject(conn, j, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownJobError{Id: id}
	}

	return err
}

func (j *Job) LoadForUpdate(conn pg.Conn, id Id, scope Scope) error {
	query := fmt.Sprintf(`
SELECT id, project_id, creation_time, update_time, disabled, spec
  FROM jobs
  WHERE %s AND id = $1
  FOR UPDATE;
`, scope.SQLCondition())

	err := pg.QueryObject(conn, j, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownJobError{Id: id}
	}

	return err
}

func (j *Job) LoadByName(conn pg.Conn, name string, scope Scope) error {
	query := fmt.Sprintf(`
SELECT id, project_id, creation_time, update_time, disabled, spec
  FROM jobs
  WHERE %s AND spec->>'name' = $1;
`, scope.SQLCondition())

	err := pg.QueryObject(conn, j, query, name)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownJobNameError{Name: name}
	}

	return err
}

func (js *Jobs) LoadByIdentityName(conn pg.Conn, name string, scope Scope) error {
	query := fmt.Sprintf(`
SELECT id, project_id, creation_time, update_time, disabled, spec
  FROM jobs
  WHERE %s
    AND (spec->'trigger'->>'identity' = $1 OR spec->'identities' ? $1)
  ORDER BY id
`, scope.SQLCondition())

	return pg.QueryObjects(conn, js, query, name)
}

func LoadJobNamesById(conn pg.Conn, ids Ids) (map[Id]string, error) {
	ctx := context.Background()

	query := `
SELECT id, spec->>'name'
  FROM jobs
  WHERE id = ANY ($1)
`
	rows, err := conn.Query(ctx, query, ids)
	if err != nil {
		return nil, err
	}

	names := make(map[Id]string)

	for rows.Next() {
		var id Id
		var name string

		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}

		names[id] = name
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return names, nil
}

func LoadJobPage(conn pg.Conn, options JobPageOptions, cursor *Cursor, scope Scope) (*Page, error) {
	favouriteJobsJoin := ``
	favouriteJobsCond := `TRUE`

	if options.ExcludeFavouriteJobAccountId != nil {
		favouriteJobsJoin = fmt.Sprintf(`LEFT JOIN favourite_jobs AS fj
    ON fj.job_id = j.id AND fj.account_id = %s
`, pg.QuoteString(options.ExcludeFavouriteJobAccountId.String()))

		favouriteJobsCond = `fj.job_id IS NULL`
	}

	query := fmt.Sprintf(`
SELECT j.id, j.project_id, j.creation_time, j.update_time, j.disabled, j.spec
  FROM jobs AS j
  %s
  WHERE %s AND %s AND %s;
`,
		favouriteJobsJoin,
		favouriteJobsCond,
		scope.SQLCondition2("j"),
		cursor.SQLConditionOrderLimit2(JobSorts, "j"))

	var jobs Jobs
	if err := pg.QueryObjects(conn, &jobs, query); err != nil {
		return nil, err
	}

	return jobs.Page(cursor), nil
}

func (j *Job) Upsert(conn pg.Conn) (Id, error) {
	ctx := context.Background()

	query := `
INSERT INTO jobs
    (id, project_id, creation_time, update_time, disabled, spec)
  VALUES
    ($1, $2, $3, $4, $5, $6)
  ON CONFLICT (project_id, (spec->>'name')) DO UPDATE SET
    update_time = EXCLUDED.update_time,
    disabled = EXCLUDED.disabled,
    spec = EXCLUDED.spec
  RETURNING id
`
	var id Id
	row := conn.QueryRow(ctx, query,
		j.Id, j.ProjectId, j.CreationTime, j.UpdateTime, j.Disabled, j.Spec)
	if err := row.Scan(&id); err != nil {
		return ZeroId, err
	}

	return id, nil
}

func (j *Job) Update(conn pg.Conn, scope Scope) error {
	query := fmt.Sprintf(`
UPDATE jobs SET
    update_time = $2,
    disabled = $3,
    spec = $4
  WHERE %s AND id = $1;
`, scope.SQLCondition())

	return pg.Exec(conn, query,
		j.Id, j.UpdateTime, j.Disabled, j.Spec)
}

func (j *Job) Delete(conn pg.Conn, scope Scope) error {
	query := fmt.Sprintf(`
DELETE FROM jobs
  WHERE %s AND id = $1;
`, scope.SQLCondition())

	return pg.Exec(conn, query, j.Id)
}

func DeleteJobs(conn pg.Conn, scope Scope) error {
	query := fmt.Sprintf(`
DELETE FROM jobs
  WHERE %s
`, scope.SQLCondition())

	return pg.Exec(conn, query)
}

func (js Jobs) Page(cursor *Cursor) *Page {
	elements := make([]PageElement, len(js))
	for i, j := range js {
		elements[i] = j
	}

	return NewPage(cursor, elements, JobSorts)
}

func (j *Job) FromRow(row pgx.Row) error {
	return row.Scan(&j.Id, &j.ProjectId, &j.CreationTime, &j.UpdateTime,
		&j.Disabled, &j.Spec)
}

func (js *Jobs) AddFromRow(row pgx.Row) error {
	var j Job
	if err := j.FromRow(row); err != nil {
		return err
	}

	*js = append(*js, &j)
	return nil
}

func (s *Step) AbortOnFailure() bool {
	switch s.OnFailure {
	case StepFailureActionAbort:
		return true

	case StepFailureActionContinue:
		return false

	default:
		return true
	}
}
