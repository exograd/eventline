package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/exograd/evgo/pkg/eventline"
	"github.com/exograd/go-daemon/pg"
)

func (s *HTTPServer) LoadProjectPage(h *HTTPHandler) (*eventline.Page, error) {
	cursor, err := h.ParseCursor(eventline.ProjectSorts)
	if err != nil {
		return nil, err
	}

	var page *eventline.Page

	err = s.Pg.WithConn(func(conn pg.Conn) (err error) {
		page, err = eventline.LoadProjectPage(conn, cursor)
		if err != nil {
			err = fmt.Errorf("cannot load projects: %w", err)
		}
		return
	})
	if err != nil {
		h.ReplyInternalError(500, "%v", err)
		return nil, err
	}

	return page, nil
}

func (s *HTTPServer) LoadProject(h *HTTPHandler, projectId eventline.Id) (*eventline.Project, error) {
	var project eventline.Project

	err := s.Pg.WithConn(func(conn pg.Conn) error {
		if err := project.Load(conn, projectId); err != nil {
			return fmt.Errorf("cannot load project: %w", err)
		}

		return nil
	})
	if err != nil {
		var unknownProjectErr *eventline.UnknownProjectError

		if errors.As(err, &unknownProjectErr) {
			h.ReplyError(404, "unknown_project", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return nil, err
	}

	return &project, nil
}

func (s *HTTPServer) LoadProjectByName(h *HTTPHandler, projectName string) (*eventline.Project, error) {
	var project eventline.Project

	err := s.Pg.WithConn(func(conn pg.Conn) error {
		if err := project.LoadByName(conn, projectName); err != nil {
			return fmt.Errorf("cannot load project: %w", err)
		}

		return nil
	})
	if err != nil {
		var unknownProjectNameErr *eventline.UnknownProjectNameError

		if errors.As(err, &unknownProjectNameErr) {
			h.ReplyError(404, "unknown_project", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return nil, err
	}

	return &project, nil
}

func (s *HTTPServer) CreateProject(h *HTTPHandler, newProject *eventline.NewProject) (*eventline.Project, error) {
	project, err := s.Service.CreateProject(newProject, h.Context.AccountId)
	if err != nil {
		var duplicateProjectNameErr *DuplicateProjectNameError

		if errors.As(err, &duplicateProjectNameErr) {
			h.ReplyError(400, "duplicate_project_name", "%v", err)
		} else {
			h.ReplyInternalError(500, "cannot create project: %v", err)
		}

		return nil, err
	}

	return project, nil
}

func (s *HTTPServer) UpdateProject(h *HTTPHandler, projectId eventline.Id, newProject *eventline.NewProject) (*eventline.Project, error) {
	var project eventline.Project

	err := s.Pg.WithTx(func(conn pg.Conn) error {
		if err := project.LoadForUpdate(conn, projectId); err != nil {
			return fmt.Errorf("cannot load project: %w", err)
		}

		if newProject.Name != project.Name {
			exists, err := eventline.ProjectNameExists(conn, newProject.Name)
			if err != nil {
				return fmt.Errorf("cannot check project name existence: %w",
					err)
			} else if exists {
				return &DuplicateProjectNameError{Name: newProject.Name}
			}
		}

		now := time.Now().UTC()

		project.Name = newProject.Name
		project.UpdateTime = now

		if err := project.Update(conn); err != nil {
			return fmt.Errorf("cannot update project: %w", err)
		}

		return nil
	})
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

		return nil, err
	}

	return &project, nil
}

func (s *HTTPServer) DeleteProject(h *HTTPHandler, projectId eventline.Id) error {
	if err := s.Service.DeleteProject(projectId, h.Context); err != nil {
		var unknownProjectErr *eventline.UnknownProjectError

		if errors.As(err, &unknownProjectErr) {
			h.ReplyError(404, "unknown_project", "%v", err)
		} else {
			h.ReplyInternalError(500, "cannot delete project: %v", err)
		}

		return fmt.Errorf("cannot delete project: %w", err)
	}

	return nil
}
