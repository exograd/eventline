package service

import (
	"errors"
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/web"
	"github.com/galdor/go-service/pkg/pg"
)

func (s *WebHTTPServer) setupIdentityRoutes() {
	s.route("/identities", "GET",
		s.hIdentitiesGET,
		HTTPRouteOptions{Project: true})

	s.route("/identities/create", "GET",
		s.hIdentitiesCreateGET,
		HTTPRouteOptions{Project: true})

	s.route("/identities/create", "POST",
		s.hIdentitiesCreatePOST,
		HTTPRouteOptions{Project: true})

	s.route("/identities/id/:id", "GET",
		s.hIdentitiesIdGET,
		HTTPRouteOptions{Project: true})

	s.route("/identities/id/:id/configuration", "GET",
		s.hIdentitiesIdConfigurationGET,
		HTTPRouteOptions{Project: true})

	s.route("/identities/id/:id/configuration", "POST",
		s.hIdentitiesIdConfigurationPOST,
		HTTPRouteOptions{Project: true})

	s.route("/identities/id/:id/refresh", "POST",
		s.hIdentitiesIdRefreshPOST,
		HTTPRouteOptions{Project: true})

	s.route("/identities/id/:id/delete", "POST",
		s.hIdentitiesIdDeletePOST,
		HTTPRouteOptions{Project: true})

	s.route("/identities/connector/:connector/types", "GET",
		s.hIdentitiesConnectorTypesGET,
		HTTPRouteOptions{Project: true})

	s.route("/identities/connector/:connector/type/:type/data", "GET",
		s.hIdentitiesConnectorTypeDataGET,
		HTTPRouteOptions{Project: true})
}

func (s *WebHTTPServer) hIdentitiesGET(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	cursor, err := h.ParseCursor(eventline.IdentitySorts)
	if err != nil {
		return
	}

	var page *eventline.Page

	err = s.Pg.WithConn(func(conn pg.Conn) (err error) {
		page, err = eventline.LoadIdentityPage(conn, cursor, scope)
		if err != nil {
			err = fmt.Errorf("cannot load identities: %w", err)
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
		Title:      "Identities",
		Menu:       NewMainMenu("identities"),
		Breadcrumb: identitiesBreadcrumb(),
		Body:       s.NewTemplate("identities.html", bodyData),
	})
}

func (s *WebHTTPServer) hIdentitiesCreateGET(h *HTTPHandler) {
	connectorSelect := IdentityConnectorSelect("/connector")
	connectorSelect.Id = "ev-connector-select"
	connectorSelect.SelectedOption = "generic"

	breadcrumb := identitiesBreadcrumb()
	breadcrumb.AddEntry(&web.BreadcrumbEntry{Label: "Creation"})

	bodyData := struct {
		Identity        *eventline.Identity
		IdentityDataDef *eventline.IdentityDataDef
		ConnectorSelect *web.Select
	}{
		Identity:        nil,
		IdentityDataDef: nil,
		ConnectorSelect: connectorSelect,
	}

	h.ReplyView(200, &web.View{
		Title:      "Identity creation",
		Menu:       NewMainMenu("identities"),
		Breadcrumb: breadcrumb,
		Body:       s.NewTemplate("identity_configuration.html", bodyData),
	})
}

func (s *WebHTTPServer) hIdentitiesCreatePOST(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	var newIdentity eventline.NewIdentity
	if err := h.JSONRequestData(&newIdentity); err != nil {
		return
	}

	identity, err := s.Service.CreateIdentity(&newIdentity, scope)
	if err != nil {
		var duplicateIdentityNameErr *DuplicateIdentityNameError

		if errors.As(err, &duplicateIdentityNameErr) {
			h.ReplyError(400, "duplicate_identity_name", "%v", err)
		} else {
			h.ReplyInternalError(500, "cannot create identity: %v", err)
		}

		return
	}

	location, err := s.Service.IdentityRedirectionURI(identity,
		h.Context.Session.Id, "/identities")
	if err != nil {
		s.Log.Error("cannot generate redirection uri for identity %q: %v",
			identity.Id, err)
		h.ReplyInternalError(500, "%v", err)
		return
	}

	extra := map[string]interface{}{
		"identity_id": identity.Id.String(),
	}

	h.ReplyJSONLocation(201, location, extra)
}

func (s *WebHTTPServer) hIdentitiesIdGET(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	identityId, err := h.IdPathVariable("id")
	if err != nil {
		return
	}

	var identity eventline.Identity
	var jobs eventline.Jobs

	err = s.Pg.WithConn(func(conn pg.Conn) error {
		if err := identity.Load(conn, identityId, scope); err != nil {
			return fmt.Errorf("cannot load identity: %w", err)
		}

		err := jobs.LoadByIdentityName(conn, identity.Name, scope)
		if err != nil {
			return fmt.Errorf("cannot load jobs: %w", err)
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

		return
	}

	bodyData := struct {
		Identity *eventline.Identity
		Jobs     eventline.Jobs
	}{
		Identity: &identity,
		Jobs:     jobs,
	}

	h.ReplyView(200, &web.View{
		Title:      "Identity",
		Menu:       NewMainMenu("identities"),
		Breadcrumb: identityBreadcrumb(&identity),
		Tabs:       identityTabs(&identity, "view"),
		Body:       s.NewTemplate("identity_view.html", bodyData),
	})
}

func (s *WebHTTPServer) hIdentitiesIdConfigurationGET(h *HTTPHandler) {
	identityId, err := h.IdPathVariable("id")
	if err != nil {
		return
	}

	identity, err := s.LoadIdentity(h, identityId)
	if err != nil {
		return
	}

	connectorSelect := IdentityConnectorSelect("/connector")
	connectorSelect.Id = "ev-connector-select"
	connectorSelect.SelectedOption = identity.Connector

	breadcrumb := identityBreadcrumb(identity)
	breadcrumb.AddEntry(&web.BreadcrumbEntry{Label: "Configuration"})

	bodyData := struct {
		Identity        *eventline.Identity
		IdentityDataDef *eventline.IdentityDataDef
		ConnectorSelect *web.Select
	}{
		Identity:        identity,
		IdentityDataDef: identity.Data.Def(),
		ConnectorSelect: connectorSelect,
	}

	h.ReplyView(200, &web.View{
		Title:      "Identity configuration",
		Menu:       NewMainMenu("identities"),
		Breadcrumb: breadcrumb,
		Tabs:       identityTabs(identity, "configuration"),
		Body:       s.NewTemplate("identity_configuration.html", bodyData),
	})
}

func (s *WebHTTPServer) hIdentitiesIdConfigurationPOST(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	identityId, err := h.IdPathVariable("id")
	if err != nil {
		return
	}

	var newIdentity eventline.NewIdentity
	if err := h.JSONRequestData(&newIdentity); err != nil {
		return
	}

	identity, err := s.Service.UpdateIdentity(identityId, &newIdentity, scope)
	if err != nil {
		var unknownIdentityErr *eventline.UnknownIdentityError
		var duplicateIdentityNameErr *DuplicateIdentityNameError
		var identityInUseErr *IdentityInUseError

		if errors.As(err, &unknownIdentityErr) {
			h.ReplyError(404, "unknown_identity", "%v", err)
		} else if errors.As(err, &duplicateIdentityNameErr) {
			h.ReplyError(400, "duplicate_identity_name", "%v", err)
		} else if errors.As(err, &identityInUseErr) {
			h.ReplyError(400, "identity_in_use", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return
	}

	location, err := s.Service.IdentityRedirectionURI(identity,
		h.Context.Session.Id, "/identities/id/"+identity.Id.String())
	if err != nil {
		s.Log.Error("cannot generate redirection uri for identity %q: %v",
			identity.Id, err)
		h.ReplyInternalError(500, "%v", err)
		return
	}

	h.ReplyJSONLocation(200, location, nil)
}

func (s *WebHTTPServer) hIdentitiesIdRefreshPOST(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	identityId, err := h.IdPathVariable("id")
	if err != nil {
		return
	}

	if err := s.Service.RefreshIdentity(identityId, scope); err != nil {
		var unknownIdentityErr *eventline.UnknownIdentityError

		if errors.As(err, &unknownIdentityErr) {
			h.ReplyError(404, "unknown_identity", "%v", err)
		} else if errors.Is(err, ErrIdentityNotRefreshable) {
			h.ReplyError(400, "identity_not_refreshable", "%v", err)
		} else {
			h.ReplyInternalError(500, "cannot refresh identity: %v", err)
		}

		return
	}

	h.ReplyJSONLocation(200, "/identities", nil)
}

func (s *WebHTTPServer) hIdentitiesIdDeletePOST(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	identityId, err := h.IdPathVariable("id")
	if err != nil {
		return
	}

	if err := s.Service.DeleteIdentity(identityId, scope); err != nil {
		var unknownIdentityErr *eventline.UnknownIdentityError
		var identityInUseErr *IdentityInUseError

		if errors.As(err, &unknownIdentityErr) {
			h.ReplyError(404, "unknown_identity", "%v", err)
		} else if errors.As(err, &identityInUseErr) {
			h.ReplyError(400, "identity_in_use", "%v", err)
		} else {
			h.ReplyInternalError(500, "cannot delete identity: %v", err)
		}

		return
	}

	h.ReplyJSONLocation(200, "/identities", nil)
}

func (s *WebHTTPServer) hIdentitiesConnectorTypesGET(h *HTTPHandler) {
	cname := h.PathVariable("connector")
	if err := eventline.ValidateConnectorName(cname); err != nil {
		h.ReplyError(400, "unknown_connector", "%v", err)
		return
	}

	cdef := eventline.GetConnectorDef(cname)

	currentType := h.QueryParameter("current_type")

	typeSelect := IdentityTypeSelect(cdef, "/type")
	typeSelect.Id = "ev-type-select"
	typeSelect.SelectedOption = currentType

	contentData := struct {
		TypeSelect *web.Select
	}{
		TypeSelect: typeSelect,
	}

	content := s.NewTemplate("identity_configuration_types.html", contentData)

	h.ReplyContent(200, content)
}

func (s *WebHTTPServer) hIdentitiesConnectorTypeDataGET(h *HTTPHandler) {
	cname := h.PathVariable("connector")
	if err := eventline.ValidateConnectorName(cname); err != nil {
		h.ReplyError(400, "unknown_connector", "%v", err)
		return
	}

	cdef := eventline.GetConnectorDef(cname)

	itype := h.PathVariable("type")
	if err := cdef.ValidateIdentityType(itype); err != nil {
		h.ReplyError(400, "unknown_identity", "%v", err)
		return
	}

	idef := cdef.Identity(itype)

	contentData := struct {
		IdentityDataDef *eventline.IdentityDataDef
	}{
		IdentityDataDef: idef.DataDef,
	}

	content := s.NewTemplate("identity_data_form.html", contentData)

	h.ReplyContent(200, content)
}

func identitiesBreadcrumb() *web.Breadcrumb {
	breadcrumb := web.NewBreadcrumb()

	breadcrumb.AddEntry(&web.BreadcrumbEntry{
		Label: "Identities",
		URI:   "/identities",
	})

	return breadcrumb
}

func identityBreadcrumb(identity *eventline.Identity) *web.Breadcrumb {
	breadcrumb := identitiesBreadcrumb()

	breadcrumb.AddEntry(&web.BreadcrumbEntry{
		Label: identity.Name,
		URI:   "/identities/id/" + identity.Id.String(),
	})

	return breadcrumb
}

func identityTabs(identity *eventline.Identity, selectedTab string) *web.Tabs {
	tabs := web.NewTabs()
	tabs.SelectedTab = selectedTab

	tabs.AddTab(&web.Tab{
		Id:    "view",
		Icon:  "lock-outline",
		Label: "Identity",
		URI:   "/identities/id/" + identity.Id.String(),
	})

	tabs.AddTab(&web.Tab{
		Id:    "configuration",
		Icon:  "cog-outline",
		Label: "Configuration",
		URI:   "/identities/id/" + identity.Id.String() + "/configuration",
	})

	return tabs
}
