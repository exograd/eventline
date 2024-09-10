package eventline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/exograd/eventline/pkg/utils"
	"github.com/jackc/pgx/v5"
	"go.n16f.net/ejson"
	"go.n16f.net/program"
	"go.n16f.net/service/pkg/pg"
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

type JobRenamingData struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
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

	Runner     *JobRunner `json:"runner"`
	Concurrent bool       `json:"concurrent,omitempty"`

	Retention int `json:"retention,omitempty"` // days

	Identities  []string          `json:"identities,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Steps       Steps             `json:"steps"`
}

type JobSpecs []*JobSpec

type JobRunner struct {
	Name          string           `json:"name"`
	Parameters    RunnerParameters `json:"-"`
	RawParameters json.RawMessage  `json:"parameters"`
	Identity      string           `json:"identity,omitempty"`
}

type RunnerParameters interface {
	ejson.Validatable
}

type Trigger struct {
	Event         EventRef               `json:"event"`
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

	Content string `json:"content,omitempty"` // content of the script file
}

func (j *Job) SortKey(sort string) (key string) {
	switch sort {
	case "id":
		key = j.Id.String()
	case "name":
		key = j.Spec.Name
	default:
		program.Panic("unknown job sort %q", sort)
	}

	return
}

func (data *JobRenamingData) ValidateJSON(v *ejson.Validator) {
	CheckName(v, "name", data.Name)
	if data.Description != "" {
		CheckDescription(v, "description", data.Description)
	}
}

func (spec JobSpec) ValidateJSON(v *ejson.Validator) {
	CheckName(v, "name", spec.Name)
	if spec.Description != "" {
		CheckDescription(v, "description", spec.Description)
	}

	v.CheckOptionalObject("trigger", spec.Trigger)
	v.CheckObjectArray("parameters", spec.Parameters)

	v.CheckOptionalObject("runner", spec.Runner)

	if spec.Retention != 0 {
		v.CheckIntMin("retention", spec.Retention, 1)
	}

	v.WithChild("identities", func() {
		for i, iname := range spec.Identities {
			CheckName(v, i, iname)
		}
	})

	v.CheckObjectArray("steps", spec.Steps)
	for i, s := range spec.Steps {
		if s.Label == "" {
			s.Label = "Step " + strconv.Itoa(i+1)
		}
	}
}

func (r *JobRunner) ValidateJSON(v *ejson.Validator) {
	runnerNames := make([]string, 0, len(RunnerDefs))
	for name := range RunnerDefs {
		runnerNames = append(runnerNames, name)
	}

	if v.CheckStringValue("name", r.Name, runnerNames) {
		v.CheckObject("parameters", r.Parameters)
	}
}

func (pr *JobRunner) MarshalJSON() ([]byte, error) {
	type JobRunner2 JobRunner

	r := JobRunner2(*pr)

	// Careful here: in evcli, we do not have access to runner definitions, so
	// we cannot decode runner parameters. We want to be able to read them
	// into RawParameters and send them as-is to the server.

	if r.RawParameters == nil {
		params, err := json.Marshal(r.Parameters)
		if err != nil {
			return nil, fmt.Errorf("cannot encode parameters: %w", err)
		}

		r.RawParameters = params
	}

	return json.Marshal(r)
}

func (pr *JobRunner) UnmarshalJSON(data []byte) error {
	type JobRunner2 JobRunner

	r := JobRunner2(*pr)
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}

	if def, found := RunnerDefs[r.Name]; found {
		params := def.InstantiateParameters()

		// Note that at this moment, Check has not been called yet, so the
		// runner name may be invalid. It is better to let Check validate it
		// so that users get full validation errors.

		if r.RawParameters != nil {
			d := json.NewDecoder(bytes.NewReader(r.RawParameters))
			d.DisallowUnknownFields()

			if err := d.Decode(params); err != nil {
				return fmt.Errorf("invalid runner parameters: %w", err)
			}
		}

		r.Parameters = params
	}

	*pr = JobRunner(r)
	return nil
}

func (t *Trigger) ValidateJSON(v *ejson.Validator) {
	CheckEventRef(v, "event", t.Event)

	v.CheckOptionalObject("parameters", t.Parameters)

	v.CheckObjectArray("filters", t.Filters)
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
	if t.RawParameters != nil && EventDefExists(t.Event) {
		edef := GetEventDef(t.Event)

		parameters, err := edef.DecodeSubscriptionParameters(t.RawParameters)
		if err != nil {
			return fmt.Errorf("cannot decode parameters: %w", err)
		}

		t.Parameters = parameters
	}

	*pt = Trigger(t)
	return nil
}

func (s *Step) ValidateJSON(v *ejson.Validator) {
	if s.Label != "" {
		CheckLabel(v, "label", s.Label)
	}

	n := 0
	if s.Code != "" {
		n += 1
	}
	if s.Command != nil {
		n += 1
	}
	if s.Script != nil {
		n += 1
	}

	if n == 0 {
		v.AddError(ejson.Pointer{}, "missing_step_content",
			"missing code, command or script member")
	} else if n > 1 {
		v.AddError(ejson.Pointer{}, "multiple_step_contents",
			"multiple code, command or script members")
	}

	v.CheckOptionalObject("command", s.Command)
	v.CheckOptionalObject("script", s.Script)
}

func (s *StepCommand) ValidateJSON(v *ejson.Validator) {
	v.CheckStringNotEmpty("name", s.Name)
}

func (s *StepScript) ValidateJSON(v *ejson.Validator) {
	v.CheckStringNotEmpty("path", s.Path)
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

	d := json.NewDecoder(bytes.NewReader(jsonData))
	d.DisallowUnknownFields()
	if err := d.Decode(spec); err != nil {
		return fmt.Errorf("cannot decode json data: %w", err)
	}

	return nil
}

func (spec *JobSpec) IdentityNames() []string {
	var names []string

	if spec.Trigger != nil && spec.Trigger.Identity != "" {
		names = append(names, spec.Trigger.Identity)
	}

	if spec.Runner != nil && spec.Runner.Identity != "" {
		names = append(names, spec.Runner.Identity)
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
    AND (spec->'trigger'->>'identity' = $1
         OR spec->'runner'->>'identity' = $1
         OR spec->'identities' ? $1)
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

func (j *Job) UpdateRename(conn pg.Conn, scope Scope) error {
	query := fmt.Sprintf(`
UPDATE jobs SET
    spec = spec ||
      jsonb_build_object('name', $2::text,
                         'description', $3::text)
  WHERE %s AND id = $1;
`, scope.SQLCondition())

	return pg.Exec(conn, query,
		j.Id, j.Spec.Name, j.Spec.Description)
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
