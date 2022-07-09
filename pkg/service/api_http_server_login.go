package service

import (
	"errors"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
)

func (s *APIHTTPServer) setupLoginRoute() {
	s.route("/login", "POST", s.hLoginPOST,
		HTTPRouteOptions{Public: true})
}

func (s *APIHTTPServer) hLoginPOST(h *HTTPHandler) {
	var loginData LoginData
	if err := h.JSONRequestObject(&loginData); err != nil {
		return
	}

	if _, err := s.LogIn(h, &loginData); err != nil {
		return
	}

	// We could try to be smart to avoid potential errors if someone tries to
	// login multiple times during the same second. But it probably will never
	// happen, and if it does, the user will just retry.
	now := time.Now().UTC()
	newAPIKey := eventline.NewAPIKey{
		Name: "evcli-" + now.Format("20060102T150405Z"),
	}

	scope := h.Context.AccountScope()

	apiKey, key, err := s.Service.CreateAPIKey(&newAPIKey, scope)
	if err != nil {
		var duplicateAPIKeyNameErr *DuplicateAPIKeyNameError

		if errors.As(err, &duplicateAPIKeyNameErr) {
			h.ReplyError(400, "duplicate_api_key_name", "%v", err)
		} else {
			h.ReplyInternalError(500, "cannot create api key: %v", err)
		}

		return
	}

	res := struct {
		APIKey *eventline.APIKey `json:"api_key"`
		Key    string            `json:"key"`
	}{
		APIKey: apiKey,
		Key:    key,
	}

	h.ReplyJSON(200, res)
}
