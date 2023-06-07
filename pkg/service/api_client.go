package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/galdor/go-service/pkg/shttp"
)

type APIRequestError struct {
	Status   int
	APIError *shttp.JSONError
	Message  string
}

func (err APIRequestError) Error() string {
	return err.Message
}

type APIClient struct {
	*shttp.Client
}

func NewAPIClient(c *shttp.Client) *APIClient {
	return &APIClient{
		Client: c,
	}
}

func (c *APIClient) SendRequest(method string, uri *url.URL, header map[string]string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, uri.String(), body)

	for name, value := range header {
		req.Header.Add(name, value)
	}

	res, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if !(res.StatusCode >= 200 && res.StatusCode < 300) {
		reqErr := &APIRequestError{
			Status:   res.StatusCode,
			APIError: nil,
			Message: fmt.Sprintf("request failed with status %d",
				res.StatusCode),
		}

		resBody, err := ioutil.ReadAll(res.Body)
		if err == nil {
			if res.Header.Get("Content-Type") == "application/json" {
				var apiErr shttp.JSONError
				if err := json.Unmarshal(resBody, &apiErr); err == nil {
					reqErr.APIError = &apiErr
					reqErr.Message += ": " + apiErr.Message
				} else {
					c.Log.Error("cannot decode api error response: %v", err)
				}
			}

			if reqErr.APIError == nil && len(resBody) > 0 {
				reqErr.Message += ": " + string(resBody)
			}
		} else {
			c.Log.Error("cannot read response body: %v", err)
		}

		return res, reqErr
	}

	return res, nil
}

func (c *APIClient) SendJSONRequest(method string, uri *url.URL, header map[string]string, value interface{}) (*http.Response, error) {
	var body io.Reader

	if value != nil {
		var buf bytes.Buffer

		encoder := json.NewEncoder(&buf)
		if err := encoder.Encode(value); err != nil {
			return nil, fmt.Errorf("cannot encode request body: %w", err)
		}

		body = &buf
	}

	if header == nil {
		header = make(map[string]string)
	}

	if _, found := header["Content-Type"]; !found {
		header["Content-Type"] = "application/json"
	}

	return c.SendRequest(method, uri, header, body)
}
