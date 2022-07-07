package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/exograd/eventline/pkg/eventline"
)

type Client struct {
	APIKey    string
	ProjectId string

	httpClient *http.Client

	baseURI *url.URL
}

func NewClient(config *Config) (*Client, error) {
	baseURI, err := url.Parse(config.API.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid api endpoint: %w", err)
	}

	client := &Client{
		baseURI: baseURI,
	}

	return client, nil
}

func (c *Client) SendRequest(method string, relURI *url.URL, body, dest interface{}) error {
	uri := c.baseURI.ResolveReference(relURI)

	var bodyReader io.Reader
	if body == nil {
		bodyReader = nil
	} else if br, ok := body.(io.Reader); ok {
		bodyReader = br
	} else {
		bodyData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("cannot encode body: %w", err)
		}

		bodyReader = bytes.NewReader(bodyData)
	}

	req, err := http.NewRequest(method, uri.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("cannot create request: %w", err)
	}

	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	if c.ProjectId != "" {
		req.Header.Set("X-Eventline-Project-Id", c.ProjectId)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot send request: %w", err)
	}
	defer res.Body.Close()

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("cannot read response body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var apiErr APIError

		err := json.Unmarshal(resBody, &apiErr)
		if err == nil {
			return &apiErr
		}

		p.Debug(1, "cannot decode response body: %v", err)

		return fmt.Errorf("request failed with status %d: %s",
			res.StatusCode, string(resBody))
	}

	if dest != nil {
		if dataPtr, ok := dest.(*[]byte); ok {
			*dataPtr = resBody
		} else {
			if len(resBody) == 0 {
				return fmt.Errorf("empty response body")
			}

			if err := json.Unmarshal(resBody, dest); err != nil {
				return fmt.Errorf("cannot decode response body: %w", err)
			}
		}
	}

	return err
}

func (c *Client) FetchProjects() ([]*Project, error) {
	var page ProjectPage

	uri := NewURL("projects")

	query := url.Values{}
	query.Add("size", "20")
	uri.RawQuery = query.Encode()

	err := c.SendRequest("GET", uri, nil, &page)
	if err != nil {
		return nil, err
	}

	return page.Elements, nil
}

func (c *Client) FetchProjectByName(name string) (*Project, error) {
	uri := NewURL("projects", "name", name)

	var project Project

	err := c.SendRequest("GET", uri, nil, &project)
	if err != nil {
		return nil, err
	}

	return &project, nil
}

func (c *Client) CreateProject(project *Project) error {
	uri := NewURL("projects")

	return c.SendRequest("POST", uri, project, project)
}

func (c *Client) DeleteProject(id string) error {
	uri := NewURL("projects", "id", id)

	return c.SendRequest("DELETE", uri, nil, nil)
}

func (c *Client) FetchCommands() ([]*Resource, error) {
	var commands Resources

	cursor := Cursor{Size: 20}

	for {
		var page ResourcePage

		uri := NewURL("resources")

		query := url.Values{}
		query.Add("type", "command")
		query.Add("size", strconv.FormatUint(uint64(cursor.Size), 10))
		if cursor.After != "" {
			query.Add("after", cursor.After)
		}
		uri.RawQuery = query.Encode()

		err := c.SendRequest("GET", uri, nil, &page)
		if err != nil {
			return nil, err
		}

		commands = append(commands, page.Elements...)

		if page.Next == nil {
			break
		}

		cursor = *page.Next
	}

	return commands, nil
}

func (c *Client) FetchCommandByName(name string) (*Resource, error) {
	uri := NewURL("resources", "type", "command", "name", name)

	var command Resource

	err := c.SendRequest("GET", uri, nil, &command)
	if err != nil {
		return nil, err
	}

	return &command, nil
}

func (c *Client) ExecuteCommand(id string, input *CommandExecutionInput) (*CommandExecution, error) {
	uri := NewURL("commands", "id", id, "execute")

	var execution CommandExecution

	err := c.SendRequest("POST", uri, input, &execution)
	if err != nil {
		return nil, err
	}

	return &execution, nil
}

func (c *Client) FetchPipelines() (Pipelines, error) {
	var page PipelinePage

	uri := NewURL("pipelines")

	query := url.Values{}
	query.Add("size", "20")
	query.Add("sort", "event_time")
	query.Add("order", "desc")
	uri.RawQuery = query.Encode()

	err := c.SendRequest("GET", uri, nil, &page)
	if err != nil {
		return nil, err
	}

	return page.Elements, nil
}

func (c *Client) AbortPipeline(id string) error {
	uri := NewURL("pipelines", "id", id, "abort")

	return c.SendRequest("POST", uri, nil, nil)
}

func (c *Client) RestartPipeline(id string) error {
	uri := NewURL("pipelines", "id", id, "restart")

	return c.SendRequest("POST", uri, nil, nil)
}

func (c *Client) RestartPipelineFromFailure(id string) error {
	uri := NewURL("pipelines", "id", id, "restart_from_failure")

	return c.SendRequest("POST", uri, nil, nil)
}

func (c *Client) GetScratchpad(id string) (map[string]string, error) {
	uri := NewURL("pipelines", "id", id, "scratchpad")

	var entries map[string]string

	if err := c.SendRequest("GET", uri, nil, &entries); err != nil {
		return nil, err
	}

	return entries, nil
}

func (c *Client) ClearScratchpad(id string) error {
	uri := NewURL("pipelines", "id", id, "scratchpad")

	return c.SendRequest("DELETE", uri, nil, nil)
}

func (c *Client) GetScratchpadEntry(id, key string) (string, error) {
	uri := NewURL("pipelines", "id", id, "scratchpad", "key", key)

	var value []byte

	if err := c.SendRequest("GET", uri, nil, &value); err != nil {
		return "", err
	}

	return string(value), nil
}

func (c *Client) SetScratchpadEntry(id, key, value string) error {
	uri := NewURL("pipelines", "id", id, "scratchpad", "key", key)

	return c.SendRequest("PUT", uri, strings.NewReader(value), nil)
}

func (c *Client) DeleteScratchpadEntry(id, key string) error {
	uri := NewURL("pipelines", "id", id, "scratchpad", "key", key)

	return c.SendRequest("DELETE", uri, nil, nil)
}

func (c *Client) CreateEvent(newEvent *NewEvent) (Events, error) {
	var events Events

	uri := NewURL("events")

	err := c.SendRequest("POST", uri, newEvent, &events)
	if err != nil {
		return nil, err
	}

	return events, nil
}

func (c *Client) ReplayEvent(id string) (*Event, error) {
	var event Event

	uri := NewURL("events", "id", id, "replay")

	err := c.SendRequest("POST", uri, nil, &event)
	if err != nil {
		return nil, err
	}

	return &event, nil
}

func (c *Client) FetchJobByName(name string) (*eventline.Job, error) {
	uri := NewURL("jobs", "name", name)

	var job eventline.Job

	err := c.SendRequest("GET", uri, nil, &job)
	if err != nil {
		return nil, err
	}

	return &job, nil
}

func (c *Client) FetchJobs() (eventline.Jobs, error) {
	var jobs eventline.Jobs

	cursor := Cursor{Size: 20}

	for {
		var page JobPage

		uri := NewURL("jobs")

		query := url.Values{}
		query.Add("size", strconv.FormatUint(uint64(cursor.Size), 10))
		if cursor.After != "" {
			query.Add("after", cursor.After)
		}
		uri.RawQuery = query.Encode()

		err := c.SendRequest("GET", uri, nil, &page)
		if err != nil {
			return nil, err
		}

		jobs = append(jobs, page.Elements...)

		if page.Next == nil {
			break
		}

		cursor = *page.Next
	}

	return jobs, nil
}

func (c *Client) DeployJob(spec *eventline.JobSpec, dryRun bool) (*eventline.Job, error) {
	uri := NewURL("jobs", "name", spec.Name)

	query := url.Values{}
	if dryRun {
		query.Add("dry-run", "")
	}
	uri.RawQuery = query.Encode()

	if dryRun {
		if err := c.SendRequest("PUT", uri, spec, nil); err != nil {
			return nil, err
		}

		return nil, nil
	} else {
		var job eventline.Job
		if err := c.SendRequest("PUT", uri, spec, &job); err != nil {
			return nil, err
		}

		return &job, nil
	}

}

func (c *Client) DeployJobs(specs []*eventline.JobSpec, dryRun bool) ([]*eventline.Job, error) {
	uri := NewURL("jobs")

	query := url.Values{}
	if dryRun {
		query.Add("dry-run", "")
	}
	uri.RawQuery = query.Encode()

	var jobs []*eventline.Job
	if err := c.SendRequest("POST", uri, specs, &jobs); err != nil {
		return nil, err
	}

	return jobs, nil
}

func (c *Client) DeleteJob(id string) error {
	uri := NewURL("jobs", "id", id)

	return c.SendRequest("DELETE", uri, nil, nil)
}

func (c *Client) ExecuteJob(id string, input *eventline.JobExecutionInput) (*eventline.JobExecution, error) {
	uri := NewURL("jobs", "id", id, "execute")

	var jobExecution eventline.JobExecution

	if err := c.SendRequest("POST", uri, input, &jobExecution); err != nil {
		return nil, err
	}

	return &jobExecution, nil
}
