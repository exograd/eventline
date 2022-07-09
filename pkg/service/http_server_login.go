package service

import (
	"errors"

	"github.com/exograd/eventline/pkg/eventline"
)

func (s *HTTPServer) LogIn(h *HTTPHandler, loginData *LoginData) (*eventline.Session, error) {
	session, err := s.Service.LogIn(loginData, h.Context)
	if err != nil {
		var unknownUsernameErr *eventline.UnknownUsernameError

		if errors.As(err, &unknownUsernameErr) {
			h.ReplyError(403, "unknown_username", "%v", err)
		} else if errors.Is(err, ErrWrongPassword) {
			h.ReplyError(403, "wrong_password", "%v", err)
		} else {
			h.ReplyInternalError(500, "cannot log in: %v", err)
		}

		return nil, err
	}

	return session, nil
}
