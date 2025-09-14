package service

import (
	"fmt"
	"net/url"
	"path"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-oauth2c"
	"go.n16f.net/service/pkg/pg"
)

func (s *WebHTTPServer) processOAuth2Request(h *HTTPHandler) {
	// Decode the scope to obtain an identity id and session id
	state := h.QueryParameter("state")
	if state == "" {
		h.Log.Error("missing or empty oauth2 state query parameter")
		h.ReplyError(400, "missing_oauth2_state",
			"missing or empty oauth2 state query parameter")
		return
	}

	pIdentityId, pSessionId, err := DecodeOAuth2State(state)
	if err != nil {
		h.Log.Error("invalid oauth2 state: %v", err)
		h.ReplyError(400, "invalid_oauth2_state", "invalid oauth2 state")
		return
	}

	identityId := *pIdentityId
	sessionId := *pSessionId

	// Check that we have an OAuth2 code
	code := h.QueryParameter("code")
	if code == "" {
		h.Log.Error("missing or empty oauth2 code query parameter")
		h.ReplyError(400, "missing_oauth2_code",
			"missing or empty oauth2 code query parameter")
		return
	}

	err = s.Service.Pg.WithTx(func(conn pg.Conn) error {
		// Load the session referenced in the state and update the context
		var session eventline.Session
		if err := session.LoadUpdate(conn, sessionId); err != nil {
			return fmt.Errorf("cannot load session: %w", err)
		}

		h.SetContextSession(&session)
		h.SetSessionCookie(s.Service.sessionCookie(session.Id))

		// Load the identity referenced in the state
		scope := h.Context.ProjectScope()

		var identity eventline.Identity
		if err := identity.LoadForUpdate(conn, identityId, scope); err != nil {
			return fmt.Errorf("cannot load identity: %w", err)
		}

		cdef := eventline.GetConnectorDef(identity.Connector)
		idef := cdef.Identity(identity.Type)

		// Check for an OAuth2 error
		authErr := oauth2c.GetRequestError(h.Request)

		if authErr == nil {
			// Use the OAuth2 code to fetch token data and update identity
			// data.
			httpClient, err := s.Service.oauth2HTTPClient(identityId,
				&sessionId)
			if err != nil {
				return fmt.Errorf("cannot create http client: %w", err)
			}

			identityData := identity.Data.(eventline.OAuth2IdentityData)

			path := path.Join("ext", "connectors", identity.Connector,
				identity.Type)
			redirectionURI := s.Service.WebHTTPServerURI.ResolveReference(
				&url.URL{Path: path})

			err = identityData.FetchTokenData(httpClient, code,
				redirectionURI.String())
			if err != nil {
				return fmt.Errorf("cannot fetch token data: %w", err)
			}

			identity.Status = eventline.IdentityStatusReady
			identity.ErrorMessage = ""

			if idef.Refreshable {
				identityData2 :=
					identityData.(eventline.RefreshableOAuth2IdentityData)
				refreshTime := identityData2.RefreshTime()
				identity.RefreshTime = &refreshTime
			}
		} else {
			identity.Status = eventline.IdentityStatusError
			identity.ErrorMessage = authErr.Error()
		}

		// Update the identity
		if err := identity.Update(conn); err != nil {
			return fmt.Errorf("cannot update identity %q: %w", identityId, err)
		}

		return nil
	})
	if err != nil {
		h.ReplyInternalError(500, "%v", err)
		return
	}

	// Redirect the client
	h.ReplyRedirect(303, "/identities")
}
