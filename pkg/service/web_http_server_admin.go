package service

import (
	"errors"
	"fmt"

	"github.com/exograd/evgo/pkg/eventline"
	"github.com/exograd/evgo/pkg/web"
	"github.com/exograd/go-daemon/pg"
)

func (s *WebHTTPServer) setupAdminRoutes() {
	s.route("/admin", "GET",
		s.hAdminGET,
		HTTPRouteOptions{Admin: true})

	s.route("/admin/accounts", "GET",
		s.hAdminAccountsGET,
		HTTPRouteOptions{Admin: true})

	s.route("/admin/accounts/create", "GET",
		s.hAdminAccountsCreateGET,
		HTTPRouteOptions{Admin: true})

	s.route("/admin/accounts/create", "POST",
		s.hAdminAccountsCreatePOST,
		HTTPRouteOptions{Admin: true})

	s.route("/admin/accounts/id/{id}/edit", "GET",
		s.hAdminAccountsIdEditGET,
		HTTPRouteOptions{Admin: true})

	s.route("/admin/accounts/id/{id}/edit", "POST",
		s.hAdminAccountsIdEditPOST,
		HTTPRouteOptions{Admin: true})

	s.route("/admin/accounts/id/{id}/change_password", "GET",
		s.hAdminAccountsIdChangePasswordGET,
		HTTPRouteOptions{Admin: true})

	s.route("/admin/accounts/id/{id}/change_password", "POST",
		s.hAdminAccountsIdChangePasswordPOST,
		HTTPRouteOptions{Admin: true})

	s.route("/admin/accounts/id/{id}/delete", "POST",
		s.hAdminAccountsIdDeletePOST,
		HTTPRouteOptions{Admin: true})
}

func (s *WebHTTPServer) hAdminGET(h *HTTPHandler) {
	h.ReplyRedirect(302, "/admin/accounts")
}

func (s *WebHTTPServer) hAdminAccountsGET(h *HTTPHandler) {
	page, err := s.LoadAccountPage(h)
	if err != nil {
		return
	}

	breadcrumb := adminAccountsBreadcrumb()

	bodyData := struct {
		Page *eventline.Page
	}{
		Page: page,
	}

	h.ReplyView(200, &web.View{
		Title:      "Accounts",
		Menu:       NewMainMenu("admin"),
		Breadcrumb: breadcrumb,
		Tabs:       adminTabs("accounts"),
		Body:       s.NewTemplate("admin_accounts.html", bodyData),
	})
}

func (s *WebHTTPServer) hAdminAccountsCreateGET(h *HTTPHandler) {
	breadcrumb := adminAccountsBreadcrumb()
	breadcrumb.AddEntry(&web.BreadcrumbEntry{
		Label: "Creation",
	})

	h.ReplyView(200, &web.View{
		Title:      "Account creation",
		Menu:       NewMainMenu("admin"),
		Breadcrumb: breadcrumb,
		Tabs:       adminTabs("accounts"),
		Body:       s.NewTemplate("admin_account_creation.html", nil),
	})
}

func (s *WebHTTPServer) hAdminAccountsCreatePOST(h *HTTPHandler) {
	var newAccount eventline.NewAccount
	if err := h.JSONRequestObject(&newAccount); err != nil {
		return
	}

	account, err := s.Service.CreateAccount(&newAccount)
	if err != nil {
		var duplicateUsernameErr *DuplicateUsernameError

		if errors.As(err, &duplicateUsernameErr) {
			h.ReplyError(400, "duplicate_username", "%v", err)
		} else {
			h.ReplyInternalError(500, "cannot create account: %v", err)
		}

		return
	}

	extra := map[string]interface{}{
		"account_id": account.Id.String(),
	}

	h.ReplyJSONLocation(201, "/admin/accounts", extra)
}

func (s *WebHTTPServer) hAdminAccountsIdEditGET(h *HTTPHandler) {
	accountId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	var account eventline.Account

	err = s.Pg.WithConn(func(conn pg.Conn) error {
		if err := account.Load(conn, accountId); err != nil {
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

		return
	}

	breadcrumb := adminAccountBreadcrumb(&account)
	breadcrumb.AddEntry(&web.BreadcrumbEntry{
		Label: "Edition",
	})

	bodyData := struct {
		Account *eventline.Account
	}{
		Account: &account,
	}

	h.ReplyView(200, &web.View{
		Title:      "Account edition",
		Menu:       NewMainMenu("admin"),
		Breadcrumb: breadcrumb,
		Tabs:       adminTabs("accounts"),
		Body:       s.NewTemplate("admin_account_edition.html", bodyData),
	})
}

func (s *WebHTTPServer) hAdminAccountsIdEditPOST(h *HTTPHandler) {
	accountId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	var update eventline.AccountUpdate
	if err := h.JSONRequestObject(&update); err != nil {
		return
	}

	if _, err := s.Service.UpdateAccount(accountId, &update); err != nil {
		var unknownAccountErr *eventline.UnknownAccountError
		var duplicateUsernameErr *DuplicateUsernameError

		if errors.As(err, &unknownAccountErr) {
			h.ReplyError(404, "unknown_account", "%v", err)
		} else if errors.As(err, &duplicateUsernameErr) {
			h.ReplyError(400, "duplicate_username", "%v", err)
		} else {
			h.ReplyInternalError(500, "cannot update account: %v", err)
		}

		return
	}

	h.ReplyJSONLocation(200, "/admin/accounts", nil)
}

func (s *WebHTTPServer) hAdminAccountsIdChangePasswordGET(h *HTTPHandler) {
	accountId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	var account eventline.Account

	err = s.Pg.WithConn(func(conn pg.Conn) error {
		if err := account.Load(conn, accountId); err != nil {
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

		return
	}

	breadcrumb := adminAccountBreadcrumb(&account)
	breadcrumb.AddEntry(&web.BreadcrumbEntry{
		Label: "Password change",
	})

	bodyData := struct {
		Account *eventline.Account
	}{
		Account: &account,
	}

	h.ReplyView(200, &web.View{
		Title:      "Account password change",
		Menu:       NewMainMenu("admin"),
		Breadcrumb: breadcrumb,
		Tabs:       adminTabs("accounts"),
		Body: s.NewTemplate("admin_account_password_change.html",
			bodyData),
	})
}

func (s *WebHTTPServer) hAdminAccountsIdChangePasswordPOST(h *HTTPHandler) {
	accountId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	var update eventline.AccountPasswordUpdate
	if err := h.JSONRequestObject(&update); err != nil {
		return
	}

	_, err = s.Service.UpdateAccountPassword(accountId, &update)
	if err != nil {
		var unknownAccountErr *eventline.UnknownAccountError

		if errors.As(err, &unknownAccountErr) {
			h.ReplyError(404, "unknown_account", "%v", err)
		} else {
			h.ReplyInternalError(500, "cannot update account: %v", err)
		}

		return
	}

	h.ReplyJSONLocation(200, "/admin/accounts", nil)
}

func (s *WebHTTPServer) hAdminAccountsIdDeletePOST(h *HTTPHandler) {
	accountId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	if err := s.Service.DeleteAccount(accountId); err != nil {
		return
	}

	h.ReplyEmpty(204)
}

func adminAccountsBreadcrumb() *web.Breadcrumb {
	breadcrumb := web.NewBreadcrumb()

	breadcrumb.AddEntry(&web.BreadcrumbEntry{
		Label: "Accounts",
		URI:   "/admin/accounts",
	})

	return breadcrumb
}

func adminAccountBreadcrumb(account *eventline.Account) *web.Breadcrumb {
	breadcrumb := adminAccountsBreadcrumb()

	breadcrumb.AddEntry(&web.BreadcrumbEntry{
		Label: account.Username,
		URI:   "/admin/accounts/id/" + account.Id.String(),
	})

	return breadcrumb
}

func adminTabs(selectedTab string) *web.Tabs {
	tabs := web.NewTabs()
	tabs.SelectedTab = selectedTab

	tabs.AddTab(&web.Tab{
		Id:    "accounts",
		Icon:  "account-group-outline",
		Label: "Accounts",
		URI:   "/admin/accounts",
	})

	return tabs
}
