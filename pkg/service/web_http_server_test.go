package service

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/test"
	"github.com/stretchr/testify/require"
)

type TestWebClient struct {
	Cookie    *http.Cookie
	SessionId string

	t *testing.T
}

func NewTestWebClient(t *testing.T) *TestWebClient {
	return &TestWebClient{
		t: t,
	}
}

func (c *TestWebClient) Authenticate() {
	username := test.RandomName("account", "")

	newAccount := eventline.NewAccount{
		Username:             username,
		Password:             "password",
		PasswordConfirmation: "password",
		Role:                 eventline.AccountRoleAdmin,
	}

	_, err := testService.CreateAccount(&newAccount)
	require.NoError(c.t, err)

	req := NewTestWebRequest(c.t, "POST", "/login")
	req.SetJSONBody(LoginData{Username: username, Password: "password"})

	res, err := req.Send()
	require.NoError(c.t, err)
	require.Equal(c.t, 200, res.StatusCode)

	cookies := res.Cookies()
	require.Equal(c.t, 1, len(cookies))

	c.Cookie = cookies[0]
	c.SessionId = c.Cookie.Value
}

func (c *TestWebClient) NewRequest(method, path string) *TestRequest {
	req := NewTestWebRequest(c.t, method, path)
	req.Header["Cookie"] = c.Cookie.String()
	return req
}

func NewTestWebRequest(t *testing.T, method, path string) *TestRequest {
	return &TestRequest{
		Method:  method,
		Address: testService.Cfg.WebHTTPServer.Address,
		Path:    path,
		Query:   make(url.Values),
		Header:  make(map[string]string),

		t: t,
	}
}

func TestWebStatus(t *testing.T) {
	require := require.New(t)

	var req *TestRequest
	var err error

	req = NewTestWebRequest(t, "HEAD", "/status")
	_, err = req.Send()
	require.NoError(err)
}
