package service

import (
	"net/http"
	"testing"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/test"
	"github.com/stretchr/testify/require"
	"go.n16f.net/uuid"
)

func TestWebProjects(t *testing.T) {
	require := require.New(t)

	var req *TestRequest
	var res *http.Response
	var err error

	client := NewTestWebClient(t)
	client.Authenticate()

	// Login with an unknown username
	req = client.NewRequest("POST", "/login")
	req.SetJSONBody(LoginData{Username: "unknown", Password: "wrong"})

	_, err = req.Send()
	assertRequestError(t, err, 403, "unknown_username")

	// Create a project with an invalid name
	req = client.NewRequest("POST", "/projects/create")
	req.SetJSONBody(eventline.NewProject{Name: ""})

	_, err = req.Send()
	assertRequestError(t, err, 400, "invalid_request_body")

	// Create a project
	projectName := test.RandomName("project", "1")

	req = client.NewRequest("POST", "/projects/create")
	req.SetJSONBody(eventline.NewProject{Name: projectName})

	res, err = req.Send()
	require.NoError(err)
	require.Equal(201, res.StatusCode)

	var responseData struct {
		ProjectId uuid.UUID `json:"project_id"`
	}

	assertResponseJSONBody(t, res, &responseData)

	projectId := responseData.ProjectId
	require.False(projectId.IsNil())

	// Create a project with a duplicate name
	req = client.NewRequest("POST", "/projects/create")
	req.SetJSONBody(eventline.NewProject{Name: projectName})

	_, err = req.Send()
	assertRequestError(t, err, 400, "duplicate_project_name")

	// Delete the project
	req = client.NewRequest("POST",
		"/projects/id/"+projectId.String()+"/delete")

	res, err = req.Send()
	require.NoError(err)
	require.Equal(204, res.StatusCode)

	// Delete the project again
	req = client.NewRequest("POST",
		"/projects/id/"+projectId.String()+"/delete")

	_, err = req.Send()
	assertRequestError(t, err, 404, "unknown_project")
}
