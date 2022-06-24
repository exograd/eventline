package service

import (
	"net/http"
	"testing"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/stretchr/testify/require"
)

func TestAPIAccount(t *testing.T) {
	require := require.New(t)

	var req *TestRequest
	var res *http.Response
	var err error

	client := NewTestAPIClient(t)

	// Fetch the current account
	req = client.NewRequest("GET", "/account")
	res, err = req.Send()
	require.NoError(err)

	var account eventline.Account
	assertResponseJSONBody(t, res, &account)

	require.Equal(client.Account.Id, account.Id)
}
