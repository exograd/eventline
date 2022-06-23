package service

import (
	"errors"
	"fmt"

	"github.com/exograd/evgo/pkg/eventline"
	"github.com/exograd/go-daemon/pg"
)

func (s *HTTPServer) LoadAccountPage(h *HTTPHandler) (*eventline.Page, error) {
	cursor, err := h.ParseCursor(eventline.AccountSorts)
	if err != nil {
		return nil, err
	}

	var page *eventline.Page

	err = s.Pg.WithConn(func(conn pg.Conn) (err error) {
		page, err = eventline.LoadAccountPage(conn, cursor)
		if err != nil {
			err = fmt.Errorf("cannot load accounts: %w", err)
		}
		return
	})
	if err != nil {
		h.ReplyInternalError(500, "%v", err)
		return nil, err
	}

	return page, nil
}

func (s *HTTPServer) LoadAccount(h *HTTPHandler) (*eventline.Account, error) {
	var account eventline.Account

	err := s.Pg.WithConn(func(conn pg.Conn) error {
		if err := account.Load(conn, *h.Context.AccountId); err != nil {
			return fmt.Errorf("cannot load account: %w", err)
		}

		return nil
	})
	if err != nil {
		var unknownAccountErr *eventline.UnknownAccountError

		if errors.As(err, &unknownAccountErr) {
			h.ReplyError(404, "unknown_account", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return nil, err
	}

	return &account, nil
}
