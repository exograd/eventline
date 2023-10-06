package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/exograd/eventline/pkg/eventline"
)

type Client struct {
	APIKey    string
	ProjectId *eventline.Id

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

func (c *Client) SetEndpoint(endpoint string) error {
	baseURI, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	c.baseURI = baseURI

	return nil
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

	if c.ProjectId != nil {
		req.Header.Set("X-Eventline-Project-Id", c.ProjectId.String())
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

func (c *Client) LogIn(username, password string) (*LoginResponse, error) {
	loginData := LoginData{
		Username: username,
		Password: password,
	}

	var res LoginResponse

	err := c.SendRequest("POST", NewURL("login"), &loginData, &res)
	if err != nil {
		return nil, err
	}

	if res.APIKey == nil || res.Key == "" {
		return nil, fmt.Errorf("missing or empty api key")
	}

	return &res, nil
}

func (c *Client) FetchProjects() (eventline.Projects, error) {
	var projects eventline.Projects

	cursor := eventline.Cursor{Size: 1}

	for {
		var page ProjectPage

		uri := NewURL("projects")
		uri.RawQuery = cursor.Query().Encode()

		err := c.SendRequest("GET", uri, nil, &page)
		if err != nil {
			return nil, err
		}

		projects = append(projects, page.Elements...)

		if page.Next == nil {
			break
		}

		cursor = *page.Next
	}

	return projects, nil
}

func (c *Client) FetchProjectByName(name string) (*eventline.Project, error) {
	uri := NewURL("projects", "name", name)

	var project eventline.Project

	err := c.SendRequest("GET", uri, nil, &project)
	if err != nil {
		return nil, err
	}

	return &project, nil
}

func (c *Client) CreateProject(project *eventline.Project) error {
	uri := NewURL("projects")

	return c.SendRequest("POST", uri, project, project)
}

func (c *Client) DeleteProject(id eventline.Id) error {
	uri := NewURL("projects", "id", id.String())

	return c.SendRequest("DELETE", uri, nil, nil)
}

func (c *Client) ReplayEvent(id string) (*eventline.Event, error) {
	var event eventline.Event

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

	cursor := eventline.Cursor{Size: 20}

	for {
		var page JobPage

		uri := NewURL("jobs")
		uri.RawQuery = cursor.Query().Encode()

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

	if dryRun {
		if err := c.SendRequest("PUT", uri, specs, nil); err != nil {
			return nil, err
		}

		return nil, nil
	} else {
		var jobs []*eventline.Job

		if err := c.SendRequest("PUT", uri, specs, &jobs); err != nil {
			return nil, err
		}

		return jobs, nil
	}
}

func (c *Client) RenameJob(id string, data *eventline.JobRenamingData) error {
	uri := NewURL("jobs", "id", id, "rename")

	return c.SendRequest("POST", uri, data, nil)
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

func (c *Client) FetchJobExecution(id eventline.Id) (*eventline.JobExecution, error) {
	uri := NewURL("job_executions", "id", id.String())

	var je eventline.JobExecution

	err := c.SendRequest("GET", uri, nil, &je)
	if err != nil {
		return nil, err
	}

	return &je, nil
}

func (c *Client) AbortJobExecution(id eventline.Id) error {
	uri := NewURL("job_executions", "id", id.String(), "abort")

	return c.SendRequest("POST", uri, nil, nil)
}

func (c *Client) RestartJobExecution(id eventline.Id) error {
	uri := NewURL("job_executions", "id", id.String(), "restart")

	return c.SendRequest("POST", uri, nil, nil)
}

func (c *Client) FetchIdentityByName(name string) (*eventline.RawIdentity, error) {
	uri := NewURL("identities", "name", name)

	var identity eventline.RawIdentity

	err := c.SendRequest("GET", uri, nil, &identity)
	if err != nil {
		return nil, err
	}

	return &identity, nil
}

func (c *Client) FetchIdentities() (eventline.RawIdentities, error) {
	var identities eventline.RawIdentities

	cursor := eventline.Cursor{Size: 20}

	for {
		var page RawIdentityPage

		uri := NewURL("identities")
		uri.RawQuery = cursor.Query().Encode()

		err := c.SendRequest("GET", uri, nil, &page)
		if err != nil {
			return nil, err
		}

		identities = append(identities, page.Elements...)

		if page.Next == nil {
			break
		}

		cursor = *page.Next
	}

	return identities, nil
}

func (c *Client) DeleteIdentity(id string) error {
	uri := NewURL("identities", "id", id)

	return c.SendRequest("DELETE", uri, nil, nil)
}
