package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestRequest struct {
	Method  string
	Address string
	Path    string
	Query   url.Values
	Header  map[string]string
	Body    io.Reader

	t *testing.T
}

func (req *TestRequest) SetJSONBody(value interface{}) {
	req.Header["Content-Type"] = "application/json"

	if value != nil {
		var buf bytes.Buffer

		encoder := json.NewEncoder(&buf)
		if err := encoder.Encode(value); err != nil {
			req.t.Fatalf("cannot encode request body: %v", err)
		}

		req.Body = &buf
	}
}

func (req TestRequest) Send() (*http.Response, error) {
	uri := url.URL{
		Scheme: "http",
		Host:   req.Address,
		Path:   req.Path,
	}

	return testAPIClient.SendRequest(req.Method, &uri, req.Header, req.Body)
}

func assertRequestError(t *testing.T, err error, status int, code string) bool {
	t.Helper()

	if err == nil {
		assert.Fail(t, "request succeeded")
		return false
	}

	var reqErr *APIRequestError
	if !errors.As(err, &reqErr) {
		assert.Fail(t, fmt.Sprintf("%#v is not an api request error", err))
		return false
	}

	if reqErr.Status != status {
		assert.Fail(t, fmt.Sprintf("response status is %d but should be %d",
			reqErr.Status, status))
		return false
	}

	if reqErr.APIError == nil && code != "" {
		assert.Fail(t, "response is missing an error code")
		return false
	}

	if reqErr.APIError != nil && code == "" {
		assert.Fail(t,
			fmt.Sprintf("response has error code %q but there should not be one",
				reqErr.APIError.Code))
		return false
	}

	if reqErr.APIError != nil && reqErr.APIError.Code != code {
		assert.Fail(t,
			fmt.Sprintf("response has error code %q but it should be %q",
				reqErr.APIError.Code, code))
		return false
	}

	return true
}

func assertResponseJSONBody(t *testing.T, res *http.Response, dest interface{}) bool {
	t.Helper()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		assert.Fail(t, fmt.Sprintf("cannot read response body: %v", err))
		return false
	}

	if err := json.Unmarshal(data, dest); err != nil {
		assert.Fail(t, fmt.Sprintf("cannot decode response body: %v", err))
		return false
	}

	return true
}
