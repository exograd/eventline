package service

import (
	"net/http"
	"net/url"
	"testing"

	cgeneric "github.com/exograd/eventline/pkg/connectors/generic"
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/test"
	"github.com/stretchr/testify/require"
)

func TestAPIIdentities(t *testing.T) {
	require := require.New(t)

	var req *TestRequest
	var res *http.Response
	var err error

	client := NewTestAPIClient(t)
	client.SetCurrentProject("main")

	// Create an identity
	newIdentity1 := eventline.NewIdentity{
		Name:      test.RandomName("identity", ""),
		Connector: "generic",
		Type:      "api_key",
		Data:      &cgeneric.APIKeyIdentity{Key: "foobar-1"},
	}

	req = client.NewRequest("POST", "/identities")
	req.SetJSONBody(&newIdentity1)

	res, err = req.Send()
	require.NoError(err)
	require.Equal(201, res.StatusCode)

	var createdIdentity eventline.Identity
	assertResponseJSONBody(t, res, &createdIdentity)

	identityId := createdIdentity.Id

	// Update it
	newIdentity2 := newIdentity1
	newIdentity2.Data = &cgeneric.APIKeyIdentity{Key: "foobar-2"}

	req = client.NewRequest("PUT",
		"/identities/id/"+url.PathEscape(identityId.String()))
	req.SetJSONBody(&newIdentity2)

	res, err = req.Send()
	require.NoError(err)
	require.Equal(200, res.StatusCode)

	var updatedIdentity eventline.Identity
	assertResponseJSONBody(t, res, &updatedIdentity)

	require.Equal("foobar-2",
		updatedIdentity.Data.(*cgeneric.APIKeyIdentity).Key)

	// Fetch it
	req = client.NewRequest("GET",
		"/identities/id/"+url.PathEscape(identityId.String()))

	res, err = req.Send()
	require.NoError(err)
	require.Equal(200, res.StatusCode)

	var fetchedIdentity eventline.Identity
	assertResponseJSONBody(t, res, &fetchedIdentity)

	require.Equal("foobar-2",
		updatedIdentity.Data.(*cgeneric.APIKeyIdentity).Key)

	// Delete it
	req = client.NewRequest("DELETE",
		"/identities/id/"+url.PathEscape(identityId.String()))

	res, err = req.Send()
	require.NoError(err)
	require.Equal(204, res.StatusCode)

	// Try to fetch it again
	req = client.NewRequest("GET",
		"/identities/id/"+url.PathEscape(identityId.String()))

	_, err = req.Send()
	assertRequestError(t, err, 404, "unknown_identity")
}
