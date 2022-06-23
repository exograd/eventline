package service

import (
	"errors"
	"fmt"

	"github.com/exograd/evgo/pkg/eventline"
	"github.com/exograd/evgo/pkg/web"
	"github.com/exograd/go-daemon/pg"
)

func (s *WebHTTPServer) setupProjectRoutes() {
	s.route("/projects", "GET",
		s.hProjectsGET,
		HTTPRouteOptions{})

	s.route("/projects/create", "GET",
		s.hProjectsCreateGET,
		HTTPRouteOptions{Admin: true})

	s.route("/projects/create", "POST",
		s.hProjectsCreatePOST,
		HTTPRouteOptions{Admin: true})

	s.route("/projects/dialog", "GET",
		s.hProjectsDialogGET,
		HTTPRouteOptions{})

	s.route("/projects/id/{id}/select", "POST",
		s.hProjectsIdSelectPOST,
		HTTPRouteOptions{})

	s.route("/projects/id/{id}/configuration", "GET",
		s.hProjectsIdConfigurationGET,
		HTTPRouteOptions{})

	s.route("/projects/id/{id}/configuration", "POST",
		s.hProjectsIdConfigurationPOST,
		HTTPRouteOptions{Admin: true})

	s.route("/projects/id/{id}/delete", "POST",
		s.hProjectsIdDeletePOST,
		HTTPRouteOptions{Admin: true})
}

func (s *WebHTTPServer) hProjectsGET(h *HTTPHandler) {
	page, err := s.LoadProjectPage(h)
	if err != nil {
		return
	}

	breadcrumb := projectsBreadcrumb()

	bodyData := struct {
		Page *eventline.Page
	}{
		Page: page,
	}

	h.ReplyView(200, &web.View{
		Title:      "Projects",
		Menu:       NewMainMenu(""),
		Breadcrumb: breadcrumb,
		Body:       s.NewTemplate("projects.html", bodyData),
	})
}

func (s *WebHTTPServer) hProjectsCreateGET(h *HTTPHandler) {
	breadcrumb := projectsBreadcrumb()
	breadcrumb.AddEntry(&web.BreadcrumbEntry{
		Label: "Creation",
	})

	h.ReplyView(200, &web.View{
		Title:      "Project creation",
		Menu:       NewMainMenu(""),
		Breadcrumb: breadcrumb,
		Body:       s.NewTemplate("project_creation.html", nil),
	})
}

func (s *WebHTTPServer) hProjectsCreatePOST(h *HTTPHandler) {
	var newProject eventline.NewProject
	if err := h.JSONRequestObject(&newProject); err != nil {
		return
	}

	project, err := s.CreateProject(h, &newProject)
	if err != nil {
		return
	}

	extra := map[string]interface{}{
		"project_id": project.Id.String(),
	}

	h.ReplyJSONLocation(201, "/projects", extra)
}

func (s *WebHTTPServer) hProjectsDialogGET(h *HTTPHandler) {
	var projects eventline.Projects

	err := s.Pg.WithConn(func(conn pg.Conn) (err error) {
		if err = projects.LoadAll(conn); err != nil {
			err = fmt.Errorf("cannot load projects: %w", err)
		}
		return
	})
	if err != nil {
		h.ReplyInternalError(500, "%v", err)
		return
	}

	contentData := struct {
		Projects eventline.Projects
	}{
		Projects: projects,
	}

	content := s.NewTemplate("project_dialog.html", contentData)

	h.ReplyContent(200, content)
}

func (s *WebHTTPServer) hProjectsIdSelectPOST(h *HTTPHandler) {
	projectId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	err = s.Service.SelectAccountProject(projectId, h.Context)
	if err != nil {
		var unknownProjectErr *eventline.UnknownProjectError

		if errors.As(err, &unknownProjectErr) {
			h.ReplyError(404, "unknown_project", "%v", err)
		} else {
			h.ReplyInternalError(500, "cannot select project: %v", err)
		}

		return
	}

	h.ReplyJSONLocation(200, "/jobs", nil)
}

func (s *WebHTTPServer) hProjectsIdConfigurationGET(h *HTTPHandler) {
	projectId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	var project eventline.Project
	var projectSettings eventline.ProjectSettings
	var projectNotificationSettings eventline.ProjectNotificationSettings
	var accountMultiSelect web.MultiSelect

	err = s.Pg.WithConn(func(conn pg.Conn) error {
		if err := project.Load(conn, projectId); err != nil {
			return fmt.Errorf("cannot load project: %w", err)
		}

		if err := projectSettings.Load(conn, projectId); err != nil {
			return fmt.Errorf("cannot load project settings: %w", err)
		}

		err := projectNotificationSettings.Load(conn, projectId)
		if err != nil {
			return fmt.Errorf("cannot load project notification settings: %w",
				err)
		}

		var accounts eventline.Accounts
		if err := accounts.LoadAll(conn); err != nil {
			return fmt.Errorf("cannot load accounts: %w", err)
		}

		accountMultiSelect.Name =
			"/project_notification_settings/recipient_account_ids"

		accountMultiSelect.Options =
			make([]web.MultiSelectOption, len(accounts))
		for i, a := range accounts {
			accountMultiSelect.Options[i] = web.MultiSelectOption{
				Name:  a.Id.String(),
				Label: a.Username,
			}
		}

		accountMultiSelect.SelectedOptions =
			projectNotificationSettings.RecipientAccountIds.Strings()

		return nil
	})
	if err != nil {
		var unknownProjectErr *eventline.UnknownProjectError

		if errors.As(err, &unknownProjectErr) {
			h.ReplyError(404, "unknown_project", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return
	}

	breadcrumb := projectBreadcrumb(&project)
	breadcrumb.AddEntry(&web.BreadcrumbEntry{Label: "Configuration"})

	bodyData := struct {
		Project                     *eventline.Project
		ProjectSettings             *eventline.ProjectSettings
		ProjectNotificationSettings *eventline.ProjectNotificationSettings
		AccountMultiSelect          *web.MultiSelect
	}{
		Project:                     &project,
		ProjectSettings:             &projectSettings,
		ProjectNotificationSettings: &projectNotificationSettings,
		AccountMultiSelect:          &accountMultiSelect,
	}

	h.ReplyView(200, &web.View{
		Title:      "Project configuration",
		Menu:       NewMainMenu(""),
		Breadcrumb: breadcrumb,
		Body:       s.NewTemplate("project_configuration.html", bodyData),
	})
}

func (s *WebHTTPServer) hProjectsIdConfigurationPOST(h *HTTPHandler) {
	projectId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	var cfg ProjectConfiguration
	if err := h.JSONRequestObject(&cfg); err != nil {
		return
	}

	err = s.Service.UpdateProjectConfiguration(projectId, &cfg)
	if err != nil {
		var unknownProjectErr *eventline.UnknownProjectError
		var duplicateProjectNameErr *DuplicateProjectNameError

		if errors.As(err, &unknownProjectErr) {
			h.ReplyError(404, "unknown_project", "%v", err)
		} else if errors.As(err, &duplicateProjectNameErr) {
			h.ReplyError(400, "duplicate_project_name", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return
	}

	h.ReplyJSONLocation(200, "/projects", nil)
}

func (s *WebHTTPServer) hProjectsIdDeletePOST(h *HTTPHandler) {
	projectId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	if err := s.DeleteProject(h, projectId); err != nil {
		return
	}

	h.ReplyEmpty(204)
}

func projectsBreadcrumb() *web.Breadcrumb {
	breadcrumb := web.NewBreadcrumb()

	breadcrumb.AddEntry(&web.BreadcrumbEntry{
		Label: "Projects",
		URI:   "/projects",
	})

	return breadcrumb
}

func projectBreadcrumb(project *eventline.Project) *web.Breadcrumb {
	breadcrumb := projectsBreadcrumb()

	breadcrumb.AddEntry(&web.BreadcrumbEntry{
		Label: project.Name,
	})

	return breadcrumb
}
