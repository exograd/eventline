package service

import (
	"net/url"
	"testing"

	"github.com/exograd/evgo/pkg/eventline"
	"github.com/exograd/evgo/pkg/test"
	"github.com/exograd/go-daemon/pg"
	"github.com/stretchr/testify/require"
)

type TestAPIClient struct {
	Account          *eventline.Account
	APIKey           *eventline.APIKey
	APIKeyValue      string
	CurrentProjectId *eventline.Id

	t *testing.T
}

func NewTestAPIClient(t *testing.T) *TestAPIClient {
	newAccount := eventline.NewAccount{
		Username:             test.RandomName("account", ""),
		Password:             "password",
		PasswordConfirmation: "password",
		Role:                 eventline.AccountRoleAdmin,
	}

	account, err := testService.CreateAccount(&newAccount)
	require.NoError(t, err)

	newAPIKey := eventline.NewAPIKey{
		Name: test.RandomName("api-key", ""),
	}

	scope := eventline.NewAccountScope(account.Id)

	apiKey, key, err := testService.CreateAPIKey(&newAPIKey, scope)
	require.NoError(t, err)

	return &TestAPIClient{
		Account:     account,
		APIKey:      apiKey,
		APIKeyValue: key,

		t: t,
	}
}

func (c *TestAPIClient) SetCurrentProject(name string) {
	var project eventline.Project

	err := testService.Daemon.Pg.WithConn(func(conn pg.Conn) (err error) {
		err = project.LoadByName(conn, name)
		return
	})
	require.Nil(c.t, err)

	c.CurrentProjectId = &project.Id
}

func (c *TestAPIClient) NewRequest(method, path string) *TestRequest {
	req := NewTestAPIRequest(c.t, method, path)

	req.Header["Authorization"] = "Bearer " + c.APIKeyValue

	if c.CurrentProjectId != nil {
		req.Header["X-Eventline-Project-Id"] = c.CurrentProjectId.String()
	}

	return req
}

func NewTestAPIRequest(t *testing.T, method, path string) *TestRequest {
	return &TestRequest{
		Method:  method,
		Address: testService.Cfg.APIHTTPServer.Address,
		Path:    path,
		Query:   make(url.Values),
		Header:  make(map[string]string),

		t: t,
	}
}

func TestAPIStatus(t *testing.T) {
	require := require.New(t)

	var req *TestRequest
	var err error

	req = NewTestAPIRequest(t, "HEAD", "/status")
	_, err = req.Send()
	require.NoError(err)
}

func TestAPIAuth(t *testing.T) {
	var req *TestRequest
	var err error

	// No API key
	req = NewTestAPIRequest(t, "GET", "/account")
	_, err = req.Send()
	assertRequestError(t, err, 401, "authentication_required")

	// Wrong authorization schema
	req = NewTestAPIRequest(t, "GET", "/account")
	req.Header["Authorization"] = "Basic Zm9vOmJhcg=="
	_, err = req.Send()
	assertRequestError(t, err, 401, "authentication_required")

	// Unknown API key
	req = NewTestAPIRequest(t, "GET", "/account")
	req.Header["Authorization"] = "Bearer f4c78dcb-7ab4-4cd5-ac23-4530f02f16f1"
	_, err = req.Send()
	assertRequestError(t, err, 403, "unknown_api_key")
}
