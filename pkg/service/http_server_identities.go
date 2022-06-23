package service

import (
	"errors"
	"fmt"

	"github.com/exograd/evgo/pkg/eventline"
	"github.com/exograd/go-daemon/pg"
)

func (s *HTTPServer) LoadIdentity(h *HTTPHandler, identityId eventline.Id) (*eventline.Identity, error) {
	scope := h.Context.ProjectScope()

	var identity eventline.Identity

	err := s.Pg.WithConn(func(conn pg.Conn) error {
		if err := identity.Load(conn, identityId, scope); err != nil {
			return fmt.Errorf("cannot load identity: %w", err)
		}

		return nil
	})
	if err != nil {
		var unknownIdentityErr *eventline.UnknownIdentityError

		if errors.As(err, &unknownIdentityErr) {
			h.ReplyError(404, "unknown_identity", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return nil, err
	}

	return &identity, nil
}
