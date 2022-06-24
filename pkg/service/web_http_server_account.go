package service

import (
	"errors"
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/web"
	"github.com/exograd/go-daemon/pg"
)

func (s *WebHTTPServer) setupAccountRoutes() {
	s.route("/account", "GET",
		s.hAccountGET,
		HTTPRouteOptions{})

	s.route("/account/configuration", "GET",
		s.hAccountConfigurationGET,
		HTTPRouteOptions{})

	s.route("/account/configuration", "POST",
		s.hAccountConfigurationPOST,
		HTTPRouteOptions{})

	s.route("/account/change_password", "GET",
		s.hAccountChangePasswordGET,
		HTTPRouteOptions{})

	s.route("/account/change_password", "POST",
		s.hAccountChangePasswordPOST,
		HTTPRouteOptions{})

	s.route("/account/api_keys", "GET",
		s.hAccountAPIKeysGET,
		HTTPRouteOptions{})

	s.route("/account/api_keys/create", "GET",
		s.hAccountAPIKeysCreateGET,
		HTTPRouteOptions{})

	s.route("/account/api_keys/create", "POST",
		s.hAccountAPIKeysCreatePOST,
		HTTPRouteOptions{})

	s.route("/account/api_keys/id/{id}/delete", "POST",
		s.hAccountAPIKeysIdDeletePOST,
		HTTPRouteOptions{})
}

func (s *WebHTTPServer) hAccountGET(h *HTTPHandler) {
	account, err := s.LoadAccount(h)
	if err != nil {
		return
	}

	bodyData := struct {
		Account *eventline.Account
	}{
		Account: account,
	}

	h.ReplyView(200, &web.View{
		Title:      "Accounts",
		Menu:       NewMainMenu("account"),
		Breadcrumb: accountBreadcrumb(),
		Tabs:       accountTabs("view"),
		Body:       s.NewTemplate("account_view.html", bodyData),
	})
}

func (s *WebHTTPServer) hAccountConfigurationGET(h *HTTPHandler) {
	account, err := s.LoadAccount(h)
	if err != nil {
		return
	}

	bodyData := struct {
		Account *eventline.Account
	}{
		Account: account,
	}

	h.ReplyView(200, &web.View{
		Title:      "Account configuration",
		Menu:       NewMainMenu("account"),
		Breadcrumb: accountBreadcrumb(),
		Tabs:       accountTabs("configuration"),
		Body:       s.NewTemplate("account_configuration.html", bodyData),
	})
}

func (s *WebHTTPServer) hAccountConfigurationPOST(h *HTTPHandler) {
	var update eventline.AccountSelfUpdate
	if err := h.JSONRequestObject(&update); err != nil {
		return
	}

	err := s.Service.SelfUpdateAccount(*h.Context.AccountId, &update,
		h.Context)
	if err != nil {
		h.ReplyInternalError(500, "%v", err)
		return
	}

	h.ReplyJSONLocation(200, "/account", nil)
}

func (s *WebHTTPServer) hAccountChangePasswordGET(h *HTTPHandler) {
	h.ReplyView(200, &web.View{
		Title:      "Password change",
		Menu:       NewMainMenu("account"),
		Tabs:       accountTabs("view"),
		Breadcrumb: accountBreadcrumb(),
		Body:       s.NewTemplate("account_password_change.html", nil),
	})
}

func (s *WebHTTPServer) hAccountChangePasswordPOST(h *HTTPHandler) {
	var update eventline.AccountPasswordUpdate
	if err := h.JSONRequestObject(&update); err != nil {
		return
	}

	err := s.Service.SelfUpdateAccountPassword(*h.Context.AccountId, &update)
	if err != nil {
		h.ReplyInternalError(500, "cannot update account: %v", err)
		return
	}

	h.ReplyJSONLocation(200, "/account", nil)
}

func (s *WebHTTPServer) hAccountAPIKeysGET(h *HTTPHandler) {
	scope := h.Context.AccountScope()

	cursor, err := h.ParseCursor(eventline.APIKeySorts)
	if err != nil {
		return
	}

	var page *eventline.Page

	err = s.Pg.WithConn(func(conn pg.Conn) (err error) {
		page, err = eventline.LoadAPIKeyPage(conn, cursor, scope)
		if err != nil {
			err = fmt.Errorf("cannot load api keys: %w", err)
		}
		return
	})
	if err != nil {
		h.ReplyInternalError(500, "%v", err)
		return
	}

	bodyData := struct {
		Page *eventline.Page
	}{
		Page: page,
	}

	h.ReplyView(200, &web.View{
		Title:      "API keys",
		Menu:       NewMainMenu("account"),
		Tabs:       accountTabs("api-keys"),
		Breadcrumb: apiKeysBreadcrumb(),
		Body:       s.NewTemplate("account_api_keys.html", bodyData),
	})
}

func (s *WebHTTPServer) hAccountAPIKeysCreateGET(h *HTTPHandler) {
	breadcrumb := apiKeysBreadcrumb()
	breadcrumb.AddEntry(&web.BreadcrumbEntry{
		Label: "Creation",
	})

	h.ReplyView(200, &web.View{
		Title:      "API key creation",
		Menu:       NewMainMenu("account"),
		Tabs:       accountTabs("api-keys"),
		Breadcrumb: breadcrumb,
		Body:       s.NewTemplate("account_api_key_creation.html", nil),
	})
}

func (s *WebHTTPServer) hAccountAPIKeysCreatePOST(h *HTTPHandler) {
	scope := h.Context.AccountScope()

	var newKey eventline.NewAPIKey
	if err := h.JSONRequestObject(&newKey); err != nil {
		return
	}

	apiKey, key, err := s.Service.CreateAPIKey(&newKey, scope)
	if err != nil {
		var duplicateAPIKeyNameErr *DuplicateAPIKeyNameError

		if errors.As(err, &duplicateAPIKeyNameErr) {
			h.ReplyError(400, "duplicate_api_key_name", "%v", err)
		} else {
			h.ReplyInternalError(500, "cannot create api key: %v", err)
		}

		return
	}

	extra := map[string]interface{}{
		"api_key_id":   apiKey.Id.String(),
		"api_key_name": apiKey.Name,
		"key":          key,
	}

	h.ReplyJSONLocation(201, "/account/api_keys", extra)
}

func (s *WebHTTPServer) hAccountAPIKeysIdDeletePOST(h *HTTPHandler) {
	scope := h.Context.AccountScope()

	keyId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	if err := s.Service.DeleteAPIKey(keyId, scope); err != nil {
		var unknownAPIKeyErr *eventline.UnknownAPIKeyError

		if errors.As(err, &unknownAPIKeyErr) {
			h.ReplyError(404, "unknown_api_key", "%v", err)
		} else {
			h.ReplyInternalError(500, "cannot delete api key: %v", err)
		}

		return
	}

	h.ReplyEmpty(204)
}

func accountBreadcrumb() *web.Breadcrumb {
	breadcrumb := web.NewBreadcrumb()

	breadcrumb.AddEntry(&web.BreadcrumbEntry{
		Label: "Account",
		URI:   "/account",
	})

	return breadcrumb
}

func apiKeysBreadcrumb() *web.Breadcrumb {
	breadcrumb := accountBreadcrumb()

	breadcrumb.AddEntry(&web.BreadcrumbEntry{
		Label: "API keys",
		URI:   "/account/api_keys",
	})

	return breadcrumb
}

func accountTabs(selectedTab string) *web.Tabs {
	tabs := web.NewTabs()
	tabs.SelectedTab = selectedTab

	tabs.AddTab(&web.Tab{
		Id:    "view",
		Icon:  "account-outline",
		Label: "Account",
		URI:   "/account",
	})

	tabs.AddTab(&web.Tab{
		Id:    "configuration",
		Icon:  "cog-outline",
		Label: "Configuration",
		URI:   "/account/configuration",
	})

	tabs.AddTab(&web.Tab{
		Id:    "api-keys",
		Icon:  "key-outline",
		Label: "API keys",
		URI:   "/account/api_keys",
	})

	return tabs
}
