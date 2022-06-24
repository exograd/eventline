package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/utils"
	"github.com/exograd/eventline/pkg/web"
	"github.com/exograd/go-daemon/dhttp"
	"github.com/exograd/go-daemon/pg"
	"github.com/exograd/go-log"
)

var (
	ErrAuthenticationRequired = errors.New("authentication required")
	ErrAdminRoleRequired      = errors.New("admin role required")
	ErrInvalidSessionCookie   = errors.New("invalid session cookie")
	ErrUnknownAPIKey          = errors.New("unknown api key")
	ErrUnknownAccount         = errors.New("unknown account")
	ErrUnknownSession         = errors.New("unknown session")
	ErrMissingProjectId       = errors.New("missing project id")
	ErrInvalidProjectId       = errors.New("invalid project id")
)

type contextKey struct{}

var (
	contextKeyHandler contextKey = struct{}{}
)

type HTTPInterface string

const (
	APIHTTPInterface HTTPInterface = "api"
	WebHTTPInterface HTTPInterface = "web"
)

type HTTPServer struct {
	Log *log.Logger

	Pg *pg.Client // shortcut to avoid s.Service.Daemon.Pg

	Service *Service
	Server  *dhttp.Server
}

type HTTPRouteFunc func(*HTTPHandler)

type HTTPHandler struct {
	*dhttp.Handler

	Service      *Service
	Interface    HTTPInterface
	RouteOptions HTTPRouteOptions
	Context      *HTTPContext
}

func (h *HTTPHandler) RedirectionTarget() string {
	location := "/"

	if target := h.QueryParameter("target"); target != "" {
		// Only consider the path to avoid malicious external redirections
		if targetURI, err := url.Parse(target); err == nil {
			location = targetURI.Path
		}
	}

	return location
}

func (h *HTTPHandler) ParseCursor(sorts eventline.Sorts) (*eventline.Cursor, error) {
	query := h.Request.URL.Query()

	var cursor eventline.Cursor
	err := cursor.ParseQuery(query, sorts, h.Context.AccountSettings)
	if err != nil {
		var invalidQueryParameterErr *dhttp.InvalidQueryParameterError

		if errors.As(err, &invalidQueryParameterErr) {
			h.ReplyError(400, "invalid_query_parameter", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return nil, err
	}

	return &cursor, nil
}

func (h *HTTPHandler) TimestampQueryParameter(name string) (*time.Time, error) {
	s := h.QueryParameter(name)
	if s == "" {
		return nil, nil
	}

	i64, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		err = fmt.Errorf("invalid timestamp %q", s)
		h.ReplyError(400, "invalid_query_parameter", "%v", err)
		return nil, err
	}

	t := time.Unix(i64, 0).UTC()

	return &t, nil
}

type HTTPRouteOptions struct {
	Public  bool
	Admin   bool
	Project bool
}

type HTTPContext struct {
	// If authenticated
	AccountId       *eventline.Id
	AccountRole     *eventline.AccountRole
	AccountSettings *eventline.AccountSettings

	// If authenticated on the web interface
	Session *eventline.Session

	// If there is a current project
	ProjectIdChecked bool // true if we have performed project id detection
	ProjectId        *eventline.Id
	ProjectName      string
}

func (ctx *HTTPContext) AccountScope() eventline.Scope {
	if ctx.AccountId == nil {
		return eventline.NewGlobalScope()
	}

	return eventline.NewAccountScope(*ctx.AccountId)
}

func (ctx *HTTPContext) ProjectScope() eventline.Scope {
	if ctx.ProjectId == nil {
		return eventline.NewGlobalScope()
	}

	return eventline.NewProjectScope(*ctx.ProjectId)
}

func (ctx *HTTPContext) AccountProjectScope() eventline.Scope {
	if ctx.ProjectId != nil && ctx.AccountId != nil {
		return eventline.NewAccountProjectScope(*ctx.AccountId, *ctx.ProjectId)
	} else if ctx.ProjectId != nil {
		return eventline.NewProjectScope(*ctx.ProjectId)
	} else if ctx.AccountId != nil {
		return eventline.NewAccountScope(*ctx.AccountId)
	}

	return eventline.NewGlobalScope()
}

func (s *Service) WrapRoute(fn HTTPRouteFunc, options HTTPRouteOptions, iface HTTPInterface) dhttp.RouteFunc {
	return func(dh *dhttp.Handler) {
		// Initialize the HTTP context and handler
		hctx := HTTPContext{}

		h := &HTTPHandler{
			Handler: dh,

			Service:      s,
			Interface:    iface,
			RouteOptions: options,
			Context:      &hctx,
		}

		ctx := h.Request.Context()
		ctx = context.WithValue(ctx, contextKeyHandler, h)
		dh.Request = h.Request.WithContext(ctx)

		// Look for a session cookie and load a session if there is one
		switch iface {
		case APIHTTPInterface:
			if err := h.maybeAuthAPIKey(); err != nil {
				return
			}

		case WebHTTPInterface:
			if err := h.maybeAuthSession(); err != nil {
				return
			}
		}

		// Check the account role if necessary
		if err := h.maybeCheckAdmin(); err != nil {
			return
		}

		// Check that there is a current project if necessary
		if err := h.maybeCheckProject(); err != nil {
			return
		}

		// Load project data if there is a project id
		if err := h.maybeLoadProjectData(); err != nil {
			return
		}

		fn(h)
	}
}

func (h *HTTPHandler) maybeAuthAPIKey() error {
	auth := h.Request.Header.Get("Authorization")
	parts := strings.SplitN(auth, " ", 2)
	if strings.ToLower(parts[0]) != "bearer" || len(parts) != 2 {
		if !h.RouteOptions.Public {
			h.ReplyAuthError(401, "authentication_required",
				"authentication required")
			return ErrAuthenticationRequired
		}

		return nil
	}

	key := parts[1]

	return h.loadAPIKey(key)
}

func (h *HTTPHandler) loadAPIKey(key string) error {
	keyHash := eventline.HashAPIKey(key)

	var apiKey eventline.APIKey
	var account eventline.Account

	err := h.Service.Daemon.Pg.WithConn(func(conn pg.Conn) error {
		if err := apiKey.LoadUpdateByKeyHash(conn, keyHash); err != nil {
			return fmt.Errorf("cannot load api key: %w", err)
		}

		if err := account.Load(conn, apiKey.AccountId); err != nil {
			return fmt.Errorf("cannot load account: %w", err)
		}

		return nil
	})
	if err != nil {
		var unknownAPIKeyError *eventline.UnknownAPIKeyError
		var unknownAccountError *eventline.UnknownAccountError

		if errors.As(err, &unknownAPIKeyError) {
			h.ReplyAuthError(403, "unknown_api_key", "unknown api key")
			return ErrUnknownAPIKey
		}

		if errors.As(err, &unknownAccountError) {
			h.ReplyAuthError(403, "unknown_account", "unknown account")
			return ErrUnknownAccount
		}

		h.ReplyInternalError(500, "%v", err)
		return err
	}

	h.Context.AccountId = &account.Id
	h.Context.AccountRole = &account.Role
	h.Context.AccountSettings = account.Settings

	projectIdString := h.Request.Header.Get("X-Eventline-Project-Id")
	if projectIdString != "" {
		var projectId eventline.Id
		if err := projectId.Parse(projectIdString); err != nil {
			h.ReplyError(400, "invalid_project_id",
				"invalid project id: %v", err)
			return ErrInvalidProjectId
		}

		h.Context.ProjectId = &projectId
	}

	h.Context.ProjectIdChecked = true

	return nil
}

func (h *HTTPHandler) maybeAuthSession() error {
	cookie, err := h.Handler.Request.Cookie("session_id")
	if err == http.ErrNoCookie {
		if !h.RouteOptions.Public {
			h.ReplyAuthError(401, "authentication_required",
				"authentication required")
			return ErrAuthenticationRequired
		}

		return nil
	} else if err != nil {
		h.DeleteCookie()
		h.ReplyAuthError(400, "invalid_session_cookie", "invalid session cookie")
		return ErrInvalidSessionCookie
	}

	var sessionId eventline.Id
	if err := sessionId.Parse(cookie.Value); err != nil {
		h.DeleteCookie()
		h.ReplyAuthError(400, "invalid_session_cookie", "invalid session cookie")
		return ErrInvalidSessionCookie
	}

	return h.LoadSession(sessionId)
}

func (h *HTTPHandler) LoadSession(sessionId eventline.Id) error {
	var session eventline.Session
	err := h.Service.Daemon.Pg.WithConn(func(conn pg.Conn) error {
		return session.LoadUpdate(conn, sessionId)
	})
	if err != nil {
		var unknownSessionErr *eventline.UnknownSessionError
		if errors.As(err, &unknownSessionErr) {
			h.DeleteCookie()
			h.ReplyAuthError(403, "unknown_session", "unknown session")
			return ErrUnknownSession
		}

		h.ReplyInternalError(500, "%v", err)
		return err
	}

	h.SetContextSession(&session)

	// Send back the cookie to update its expiration date
	h.SetSessionCookie(sessionCookie(sessionId))

	return nil
}

func (h *HTTPHandler) SetContextSession(session *eventline.Session) {
	h.Context.AccountId = &session.AccountId
	h.Context.AccountRole = &session.AccountRole
	h.Context.AccountSettings = session.AccountSettings

	h.Context.Session = session

	h.Context.ProjectId = session.Data.ProjectId
	h.Context.ProjectIdChecked = true
}

func (h *HTTPHandler) maybeCheckAdmin() error {
	if !h.RouteOptions.Admin {
		return nil
	}

	if h.Context.AccountRole == nil {
		// Can happen if a route has options Public and Admin at the same
		// time, which does not make any sense.
		utils.Panicf("missing account role in admin route")
	}

	accountRole := *h.Context.AccountRole

	if accountRole != eventline.AccountRoleAdmin {
		h.ReplyError(403, "permission_denied", "admin role required")
		return ErrAdminRoleRequired
	}

	return nil
}

func (h *HTTPHandler) maybeCheckProject() error {
	if !h.RouteOptions.Project {
		return nil
	}

	if h.Context.ProjectId == nil {
		h.ReplyError(401, "missing_project_id", "you need to select a project")
		return ErrMissingProjectId
	}

	return nil
}

func (h *HTTPHandler) maybeLoadProjectData() error {
	if h.Context.ProjectId == nil {
		return nil
	}

	projectId := *h.Context.ProjectId

	var project eventline.Project

	err := h.Service.Daemon.Pg.WithConn(func(conn pg.Conn) (err error) {
		err = project.Load(conn, projectId)
		if err != nil {
			err = fmt.Errorf("cannot load project %q: %w", projectId, err)
		}
		return
	})
	if err != nil {
		var unknownProjectErr *eventline.UnknownProjectError

		// An unknown project id on the API is clearly a client error. On the
		// web interface, it means the project in the session is invalid,
		// which is not supposed to be possible, so it is an internal error.

		isAPI := h.Interface == APIHTTPInterface

		if isAPI && errors.As(err, &unknownProjectErr) {
			h.ReplyError(400, "unknown_project", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return err
	}

	h.Context.ProjectName = project.Name

	return nil
}

func (h *HTTPHandler) IdRouteVariable(name string) (id eventline.Id, err error) {
	value := h.RouteVariable(name)

	if err = id.Parse(value); err != nil {
		err = fmt.Errorf("invalid route variable: invalid id: %w", err)
		h.ReplyError(400, "invalid_route_variable", "%v", err)
	}

	return
}

func (h *HTTPHandler) SetSessionCookie(cookie *http.Cookie) {
	header := h.ResponseWriter.Header()
	header.Set("Set-Cookie", cookie.String())
}

func (h *HTTPHandler) ReplyContent(status int, content web.Content) {
	ctx := WebContext{
		VersionHash:      h.Service.BuildIdHash,
		PublicPage:       h.RouteOptions.Public,
		LoggedIn:         h.Context.Session != nil,
		AccountSettings:  h.Context.AccountSettings,
		ProjectIdChecked: h.Context.ProjectIdChecked,
		ProjectId:        h.Context.ProjectId,
		ProjectName:      h.Context.ProjectName,
	}

	data, err := content.Render(&ctx)
	if err != nil {
		panic(fmt.Errorf("cannot render content: %w", err))
	}

	h.Reply(status, bytes.NewReader(data))
}

func (h *HTTPHandler) ReplyView(status int, view *web.View) {
	view.RootTemplate = h.Service.WebHTTPServer.Template

	h.ReplyContent(status, view)
}

func (h *HTTPHandler) ReplyJSONLocation(status int, uri string, extra map[string]interface{}) {
	data := map[string]interface{}{
		"location": uri,
	}

	for k, v := range extra {
		data[k] = v
	}

	h.ReplyJSON(status, data)
}

func (h *HTTPHandler) ReplyAuthError(status int, code string, format string, args ...interface{}) {
	switch h.Interface {
	case WebHTTPInterface:
		target := url.URL{
			Path:        h.Request.URL.Path,
			RawQuery:    h.Request.URL.RawQuery,
			RawFragment: h.Request.URL.RawFragment,
		}

		message := []byte(fmt.Sprintf(format, args...))
		encodedMessage := base64.URLEncoding.EncodeToString(message)

		query := url.Values{}
		query.Add("error_code", code)
		query.Add("error_message", encodedMessage)
		query.Add("target", target.String())

		uri := url.URL{
			Path:     "/login",
			RawQuery: query.Encode(),
		}

		h.DeleteCookie()

		h.ReplyRedirect(302, uri.String())

	default:
		h.ReplyError(status, code, format, args...)
	}
}

func (h *HTTPHandler) DeleteCookie() {
	cookie := expiredCookie()
	http.SetCookie(h.ResponseWriter, cookie)
}
