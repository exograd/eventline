package service

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/pg"
)

const (
	SessionCookieName = "session_id"
	SessionTTL        = 86_400 // seconds
)

var (
	ErrWrongPassword = errors.New("wrong password")
)

type LoginData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (data *LoginData) Check(c *check.Checker) {
	c.CheckStringNotEmpty("username", data.Username)
	c.CheckStringNotEmpty("password", data.Password)
}

func (s *Service) LogIn(data *LoginData, httpCtx *HTTPContext) (*eventline.Session, error) {
	var session *eventline.Session

	err := s.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		// Load the account and check the password
		var account eventline.Account

		err := account.LoadByUsernameForUpdate(conn, data.Username)
		if err != nil {
			return fmt.Errorf("cannot load account: %w", err)
		}

		if !account.CheckPassword(data.Password) {
			return ErrWrongPassword
		}

		// If there is no current project id, select the main project
		projectId := account.LastProjectId

		if projectId == nil {
			var project eventline.Project
			if err := project.LoadByName(conn, "main"); err == nil {
				projectId = &project.Id
			} else {
				var unknownProjectErr *eventline.UnknownProjectError
				if !errors.As(err, &unknownProjectErr) {
					return fmt.Errorf("cannot load project: %w", err)
				}
			}
		}

		// Create a new session
		sessionData := eventline.SessionData{
			ProjectId: projectId,
		}

		newSession := eventline.NewSession{
			Data: &sessionData,

			AccountRole:     account.Role,
			AccountSettings: account.Settings,
		}

		scope := eventline.NewAccountScope(account.Id)

		session, err = s.CreateSession(conn, &newSession, scope)
		if err != nil {
			return fmt.Errorf("cannot create session: %w", err)
		}

		// Update the HTTP context
		httpCtx.AccountId = &account.Id
		httpCtx.AccountRole = &account.Role
		httpCtx.AccountSettings = account.Settings

		httpCtx.ProjectId = projectId

		httpCtx.Session = session

		// Update the account
		now := time.Now().UTC()

		account.LastLoginTime = &now
		account.LastProjectId = projectId

		if err := account.UpdateForLogin(conn); err != nil {
			return fmt.Errorf("cannot update account: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (s *Service) LogOut(httpCtx *HTTPContext) error {
	return s.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		// Delete the session
		if err := httpCtx.Session.Delete(conn); err != nil {
			return fmt.Errorf("cannot delete session: %w", err)
		}

		// Update the HTTP context
		httpCtx.AccountId = nil
		httpCtx.AccountRole = nil
		httpCtx.AccountSettings = nil

		httpCtx.ProjectId = nil

		httpCtx.Session = nil

		return nil
	})
}

func (s *Service) CreateSession(conn pg.Conn, ns *eventline.NewSession, scope eventline.Scope) (*eventline.Session, error) {
	now := time.Now().UTC()

	accountScope := scope.(*eventline.AccountScope)

	session := eventline.Session{
		Id:              eventline.GenerateId(),
		AccountId:       accountScope.AccountId,
		CreationTime:    now,
		UpdateTime:      now,
		Data:            ns.Data,
		AccountRole:     ns.AccountRole,
		AccountSettings: ns.AccountSettings,
	}

	if err := session.Insert(conn); err != nil {
		return nil, err
	}

	return &session, nil
}

func sessionCookie(sessionId eventline.Id) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionId.String(),
		Path:     "/",
		MaxAge:   SessionTTL,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	}
}

func expiredCookie() *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Path:     "/",
		MaxAge:   -1,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	}
}
