package service

import (
	"net/http"
	"net/url"
	"testing"

	ctime "github.com/exograd/eventline/pkg/connectors/time"
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/test"
	"github.com/stretchr/testify/require"
)

func TestAPIJobs(t *testing.T) {
	require := require.New(t)

	var req *TestRequest
	var res *http.Response
	var err error

	client := NewTestAPIClient(t)
	client.SetCurrentProject("main")

	// Deploy a job
	jobName1 := test.RandomName("job", "")
	period := 3600

	jobSpec1 := eventline.JobSpec{
		Name: jobName1,
		Trigger: &eventline.Trigger{
			Event: eventline.EventRef{
				Connector: "time",
				Event:     "tick",
			},
			Parameters: &ctime.Parameters{
				Periodic: &period,
			},
		},
		Steps: eventline.Steps{
			&eventline.Step{
				Label: "do something",
				Code:  "echo 'hello world'",
			},
		},
	}

	req = client.NewRequest("PUT", "/jobs/name/"+url.PathEscape(jobName1))
	req.SetJSONBody(&jobSpec1)

	res, err = req.Send()
	require.NoError(err)
	require.Equal(200, res.StatusCode)

	var createdJob eventline.Job
	assertResponseJSONBody(t, res, &createdJob)

	jobId1 := createdJob.Id

	// Update it
	jobSpec1.Steps = append(jobSpec1.Steps, &eventline.Step{
		Label: "do something else",
		Command: &eventline.StepCommand{
			Name:      "ls",
			Arguments: []string{"-l", "-a"},
		},
	})

	req = client.NewRequest("PUT", "/jobs/name/"+url.PathEscape(jobName1))
	req.SetJSONBody(&jobSpec1)

	res, err = req.Send()
	require.NoError(err)
	require.Equal(200, res.StatusCode)

	var updatedJob eventline.Job
	assertResponseJSONBody(t, res, &updatedJob)

	// Fetch it by name
	req = client.NewRequest("GET", "/jobs/name/"+url.PathEscape(jobName1))

	res, err = req.Send()
	require.NoError(err)
	require.Equal(200, res.StatusCode)

	var fetchedJob eventline.Job
	if assertResponseJSONBody(t, res, &fetchedJob) {
		require.Equal(updatedJob.Spec, fetchedJob.Spec)
	}

	// Delete it
	req = client.NewRequest("DELETE", "/jobs/id/"+jobId1.String())

	res, err = req.Send()
	require.NoError(err)
	require.Equal(204, res.StatusCode)

	// Delete it again
	req = client.NewRequest("DELETE", "/jobs/id/"+jobId1.String())

	res, err = req.Send()
	require.Error(err)
	require.Equal(404, res.StatusCode)

	// Deploy two jobs
	jobName2 := test.RandomName("job", "")

	jobSpec2 := eventline.JobSpec{
		Name: jobName2,
		Trigger: &eventline.Trigger{
			Event: eventline.EventRef{
				Connector: "time",
				Event:     "tick",
			},
			Parameters: &ctime.Parameters{
				Periodic: &period,
			},
		},
		Steps: eventline.Steps{
			&eventline.Step{
				Label: "do something",
				Code:  "echo 'hello world'",
			},
		},
	}

	jobSpecs := eventline.JobSpecs{&jobSpec1, &jobSpec2}

	req = client.NewRequest("PUT", "/jobs")
	req.SetJSONBody(jobSpecs)

	res, err = req.Send()
	require.NoError(err)
	require.Equal(200, res.StatusCode)

	var createdJobs eventline.Jobs
	assertResponseJSONBody(t, res, &createdJobs)
}
