package service

import (
	"net/http"
	"testing"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/test"
	"github.com/stretchr/testify/require"
)

func TestWebLogin(t *testing.T) {
	require := require.New(t)

	var req *TestRequest
	var res *http.Response
	var cookies []*http.Cookie
	var cookie *http.Cookie
	var err error

	// Create a test account
	username := test.RandomName("account", "")

	newAccount := eventline.NewAccount{
		Username:             username,
		Password:             "right",
		PasswordConfirmation: "right",
		Role:                 eventline.AccountRoleUser,
	}

	_, err = testService.CreateAccount(&newAccount)
	require.NoError(err)

	// Login with an unknown username
	req = NewTestWebRequest(t, "POST", "/login")
	req.SetJSONBody(LoginData{Username: "unknown", Password: "wrong"})

	_, err = req.Send()
	assertRequestError(t, err, 403, "unknown_username")

	// Login with the wrong password
	req = NewTestWebRequest(t, "POST", "/login")
	req.SetJSONBody(LoginData{Username: username, Password: "wrong"})

	_, err = req.Send()
	assertRequestError(t, err, 403, "wrong_password")

	// Login with the right information
	req = NewTestWebRequest(t, "POST", "/login")
	req.SetJSONBody(LoginData{Username: username, Password: "right"})

	res, err = req.Send()
	require.NoError(err)
	require.Equal(200, res.StatusCode)

	cookies = res.Cookies()
	require.Equal(1, len(cookies))
	cookie = cookies[0]
	require.Equal(SessionCookieName, cookie.Name)
	require.True(cookie.Secure)
	require.True(cookie.HttpOnly)

	sessionId := cookie.Value

	// Login with the wrong password while being already logged in
	req = NewTestWebRequest(t, "POST", "/login")
	req.Header["Cookie"] = cookie.String()
	req.SetJSONBody(LoginData{Username: username, Password: "wrong"})

	_, err = req.Send()
	assertRequestError(t, err, 403, "wrong_password")

	cookies = res.Cookies()
	require.Equal(1, len(cookies))
	cookie = cookies[0]
	require.Equal(sessionId, cookie.Value) // we are not being de-logged

	// Login with the right information while being already logged in
	req = NewTestWebRequest(t, "POST", "/login")
	req.Header["Cookie"] = cookie.String()
	req.SetJSONBody(LoginData{Username: username, Password: "right"})

	res, err = req.Send()
	require.NoError(err)
	require.Equal(200, res.StatusCode)

	cookies = res.Cookies()
	require.Equal(1, len(cookies))
	cookie = cookies[0]
	require.NotEqual(sessionId, cookie.Value) // new session

	// Logout
	req = NewTestWebRequest(t, "POST", "/logout")
	req.Header["Cookie"] = cookie.String()

	res, err = req.Send()
	require.NoError(err)
	require.Equal(201, res.StatusCode)
}
