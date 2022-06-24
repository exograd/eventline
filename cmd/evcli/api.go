package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/check"
)

type APIError struct {
	Message string          `json:"error"`
	Code    string          `json:"code,omitempty"`
	RawData json.RawMessage `json:"data,omitempty"`
	Data    interface{}     `json:"-"`
}

type InvalidRequestBodyError struct {
	ValidationErrors check.ValidationErrors `json:"validation_errors"`
}

func (err APIError) Error() string {
	return err.Message
}

func (err *APIError) UnmarshalJSON(data []byte) error {
	type APIError2 APIError

	err2 := APIError2(*err)
	if jsonErr := json.Unmarshal(data, &err2); jsonErr != nil {
		return jsonErr
	}

	switch err2.Code {
	case "invalid_request_body":
		var errData InvalidRequestBodyError

		if err := json.Unmarshal(err2.RawData, &errData); err != nil {
			return fmt.Errorf("invalid jsv errors: %w", err)
		}

		err2.Data = errData
	}

	*err = APIError(err2)
	return nil
}

type APIStatus struct {
}

type Order string

const (
	OrderAsc  Order = "asc"
	OrderDesc Order = "desc"
)

type Cursor struct {
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
	Size   uint   `json:"size,omitempty"`
	Sort   string `json:"sort,omitempty"`
	Order  Order  `json:"order,omitempty"`
}

type ProjectPage struct {
	Elements []*Project `json:"elements"`
	Previous *Cursor    `json:"previous,omitempty"`
	Next     *Cursor    `json:"next,omitempty"`
}

type Project struct {
	Id   string `json:"id,omitempty"`
	Name string `json:"name"`
}

type Projects []*Project

func (ps Projects) GroupById() map[string]*Project {
	table := make(map[string]*Project)
	for _, p := range ps {
		table[p.Id] = p
	}

	return table
}

type Resource struct {
	Id           string       `json:"id"`
	ProjectId    string       `json:"project_id"`
	CreationTime time.Time    `json:"creation_time"`
	UpdateTime   time.Time    `json:"update_time"`
	Disabled     bool         `json:"disabled,omitempty"`
	Spec         ResourceSpec `json:"spec"`
}

type ResourceSpec struct {
	Type        string          `json:"type"`
	Version     int             `json:"version"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	RawData     json.RawMessage `json:"data"`
	Data        ResourceData    `json:"-"`
}

func (spec *ResourceSpec) UnmarshalJSON(data []byte) error {
	type ResourceSpec2 ResourceSpec

	spec2 := ResourceSpec2(*spec)
	if err := json.Unmarshal(data, &spec2); err != nil {
		return err
	}

	switch spec2.Type {
	case "trigger":
		// TODO
		return fmt.Errorf("unsupported resource type %q", spec2.Type)

	case "command":
		var command CommandData

		if err := json.Unmarshal(spec2.RawData, &command); err != nil {
			return fmt.Errorf("invalid command data: %w", err)
		}

		spec2.RawData = nil
		spec2.Data = &command

	case "task":
		// TODO
		return fmt.Errorf("unsupported resource type %q", spec2.Type)

	case "pipeline":
		// TODO
		return fmt.Errorf("unsupported resource type %q", spec2.Type)

	default:
		return fmt.Errorf("unknown resource type %q", spec2.Type)
	}

	*spec = ResourceSpec(spec2)
	return nil
}

type ResourceData interface{}

type Resources []*Resource

type ResourcePage struct {
	Elements []*Resource `json:"elements"`
	Previous *Cursor     `json:"previous,omitempty"`
	Next     *Cursor     `json:"next,omitempty"`
}

type CommandData struct {
	Parameters Parameters `json:"parameters"`
	Pipelines  []string   `json:"pipelines"`
}

type Parameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Default     interface{} `json:"default"`
	Description string      `json:"description"`
}

type Parameters []*Parameter

type PipelinePage struct {
	Elements []*Pipeline `json:"elements"`
	Previous *Cursor     `json:"previous,omitempty"`
	Next     *Cursor     `json:"next,omitempty"`
}

type Pipeline struct {
	Id           string     `json:"id,omitempty"`
	Name         string     `json:"name"`
	ProjectId    string     `json:"project_id,omitempty"`
	CreationTime time.Time  `json:"creation_time"`
	PipelineId   string     `json:"pipeline_id,omitempty"`
	TriggerId    string     `json:"trigger_id,omitempty"`
	EventId      string     `json:"event_id,omitempty"`
	EventTime    string     `json:"event_time"`
	Concurrent   bool       `json:"concurrent,omitempty"`
	Status       string     `json:"status"`
	StartTime    *time.Time `json:"start_time,omitempty"`
	EndTime      *time.Time `json:"end_time,omitempty"`
}

func (p *Pipeline) Duration() *time.Duration {
	if p.StartTime == nil || p.EndTime == nil {
		return nil
	}

	d := p.EndTime.Sub(*p.StartTime)
	return &d
}

type Pipelines []*Pipeline

func (ps Pipelines) ProjectIds() []string {
	idTable := make(map[string]struct{})
	for _, p := range ps {
		idTable[p.ProjectId] = struct{}{}
	}

	ids := make([]string, 0, len(idTable))
	for id := range idTable {
		ids = append(ids, id)
	}

	return ids
}

type TaskPage struct {
	Elements []*Task `json:"elements"`
	Previous *Cursor `json:"previous,omitempty"`
	Next     *Cursor `json:"next,omitempty"`
}

type Task struct {
	Id             string     `json:"id,omitempty"`
	ProjectId      string     `json:"project_id,omitempty"`
	PipelineId     string     `json:"pipeline_id,omitempty"`
	TaskId         string     `json:"task_id,omitempty"`
	InstanceId     int        `json:"instance_id,omitempty"`
	Status         string     `json:"status"`
	StartTime      *time.Time `json:"start_time,omitempty"`
	EndTime        *time.Time `json:"end_time,omitempty"`
	FailureMessage string     `json:"failure_message,omitempty"`
}

func (t *Task) Duration() *time.Duration {
	if t.StartTime == nil || t.EndTime == nil {
		return nil
	}

	d := t.EndTime.Sub(*t.StartTime)
	return &d
}

type Tasks []*Task

type NewEvent struct {
	EventTime *time.Time      `json:"event_time,omitempty"`
	Connector string          `json:"connector"`
	Name      string          `json:"name"`
	Data      json.RawMessage `json:"data"`
}

type Event struct {
	Id              string          `json:"id"`
	ProjectId       string          `json:"project_id"`
	TriggerId       string          `json:"trigger_id,omitempty"`
	CreationTime    time.Time       `json:"creation_time"`
	EventTime       time.Time       `json:"event_time"`
	Connector       string          `json:"connector"`
	Name            string          `json:"name"`
	Data            json.RawMessage `json:"data"`
	Processed       bool            `json:"processed,omitempty"`
	OriginalEventId string          `json:"original_event_id"`
}

type Events []*Event

type CommandExecutionInput struct {
	Parameters map[string]interface{} `json:"parameters"`
}

type CommandExecution struct {
	Id            string                 `json:"id"`
	ProjectId     string                 `json:"project_id"`
	ExecutorId    string                 `json:"executor_id,omitempty"`
	ExecutionTime time.Time              `json:"execution_time"`
	CommandId     string                 `json:"command_id"`
	Parameters    map[string]interface{} `json:"parameters"`
	EventId       string                 `json:"event_id"`
	PipelineIds   []string               `json:"pipeline_ids"`
}

type JobPage struct {
	Elements eventline.Jobs `json:"elements"`
	Previous *Cursor        `json:"previous,omitempty"`
	Next     *Cursor        `json:"next,omitempty"`
}
