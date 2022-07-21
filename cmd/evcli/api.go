package main

import (
	"encoding/json"
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/check"
)

type APIError struct {
	Message string          `json:"error"`
	Code    string          `json:"code,omitempty"`
	RawData json.RawMessage `json:"data,omitempty"`
	Data    interface{}     `json:"-"`
}

type InvalidRequestBodyError struct {
	ValidationErrors check.ValidationErrors `json:"validation_errors"`
}

func (err APIError) Error() string {
	return err.Message
}

func (err *APIError) UnmarshalJSON(data []byte) error {
	type APIError2 APIError

	err2 := APIError2(*err)
	if jsonErr := json.Unmarshal(data, &err2); jsonErr != nil {
		return jsonErr
	}

	switch err2.Code {
	case "invalid_request_body":
		var errData InvalidRequestBodyError

		if err := json.Unmarshal(err2.RawData, &errData); err != nil {
			return fmt.Errorf("invalid jsv errors: %w", err)
		}

		err2.Data = errData
	}

	*err = APIError(err2)
	return nil
}

type LoginData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	APIKey *eventline.APIKey `json:"api_key"`
	Key    string            `json:"key"`
}

type ProjectPage struct {
	Elements eventline.Projects `json:"elements"`
	Previous *eventline.Cursor  `json:"previous,omitempty"`
	Next     *eventline.Cursor  `json:"next,omitempty"`
}

type Parameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Default     interface{} `json:"default"`
	Description string      `json:"description"`
}

type Parameters []*Parameter

type JobPage struct {
	Elements eventline.Jobs    `json:"elements"`
	Previous *eventline.Cursor `json:"previous,omitempty"`
	Next     *eventline.Cursor `json:"next,omitempty"`
}
