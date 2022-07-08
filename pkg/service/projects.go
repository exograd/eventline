package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/pg"
)

type ProjectConfiguration struct {
	Project                     *eventline.NewProject                  `json:"project"`
	ProjectSettings             *eventline.ProjectSettings             `json:"project_settings"`
	ProjectNotificationSettings *eventline.ProjectNotificationSettings `json:"project_notification_settings"`
}

func (cfg *ProjectConfiguration) Check(c *check.Checker) {
	c.CheckObject("project", cfg.Project)
	c.CheckObject("project_settings", cfg.ProjectSettings)
	c.CheckObject("project_notification_settings", cfg.ProjectNotificationSettings)
}

type DuplicateProjectNameError struct {
	Name string
}

func (err DuplicateProjectNameError) Error() string {
	return fmt.Sprintf("duplicate project name %q", err.Name)
}

func (s *Service) CreateProject(newProject *eventline.NewProject, accountId *eventline.Id) (*eventline.Project, error) {
	var project *eventline.Project

	err := s.Daemon.Pg.WithTx(func(conn pg.Conn) (err error) {
		project, err = s.createProject(conn, newProject, accountId)
		return
	})
	if err != nil {
		return nil, err
	}

	return project, nil
}

func (s *Service) createProject(conn pg.Conn, newProject *eventline.NewProject, accountId *eventline.Id) (*eventline.Project, error) {
	now := time.Now().UTC()

	exists, err := eventline.ProjectNameExists(conn, newProject.Name)
	if err != nil {
		return nil, fmt.Errorf("cannot check project name existence: %w", err)
	} else if exists {
		return nil, &DuplicateProjectNameError{Name: newProject.Name}
	}

	project := &eventline.Project{
		Id:           eventline.GenerateId(),
		Name:         newProject.Name,
		CreationTime: now,
		UpdateTime:   now,
	}
	if err := project.Insert(conn); err != nil {
		return nil, fmt.Errorf("cannot insert project: %w", err)
	}

	settings := DefaultProjectSettings(project)
	if err := settings.Insert(conn); err != nil {
		return nil, fmt.Errorf("cannot insert project settings: %w", err)
	}

	notificationSettings := DefaultProjectNotificationSettings(project)
	if err := notificationSettings.Insert(conn); err != nil {
		return nil, fmt.Errorf("cannot insert project notification "+
			"settings: %w", err)
	}

	return project, nil
}

func (s *Service) MaybeCreateDefaultProject(conn pg.Conn, account *eventline.Account) (*eventline.Project, error) {
	var existingProject eventline.Project
	err := existingProject.LoadByName(conn, "main")
	if err == nil {
		return &existingProject, nil
	}

	if err != nil {
		var unknownProjectNameErr *eventline.UnknownProjectNameError
		if !errors.As(err, &unknownProjectNameErr) {
			return nil, fmt.Errorf("cannot load project: %w", err)
		}
	}

	newProject := eventline.NewProject{
		Name: "main",
	}

	s.Log.Info("creating default %q project", newProject.Name)

	project, err := s.createProject(conn, &newProject, &account.Id)
	if err != nil {
		return nil, fmt.Errorf("cannot create project: %w", err)
	}

	return project, nil
}

func (s *Service) DeleteProject(projectId eventline.Id, hctx *HTTPContext) error {
	scope := eventline.NewProjectScope(projectId)

	return s.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		var project eventline.Project

		if err := project.LoadForUpdate(conn, projectId); err != nil {
			return fmt.Errorf("cannot load project: %w", err)
		}

		if err := s.TerminateSubscriptions(conn, scope); err != nil {
			return err
		}

		if err := eventline.DeleteJobs(conn, scope); err != nil {
			return fmt.Errorf("cannot delete jobs: %w", err)
		}

		if err := s.DeleteIdentities(conn, scope); err != nil {
			return err
		}

		err := eventline.UpdateAccountsForProjectDeletion(conn, projectId)
		if err != nil {
			return fmt.Errorf("cannot update accounts: %w", err)
		}

		err = eventline.UpdateSessionsForProjectDeletion(conn, projectId)
		if err != nil {
			return fmt.Errorf("cannot update sessions: %w", err)
		}

		if err := project.Delete(conn); err != nil {
			return err
		}

		if hctx.ProjectId != nil && *hctx.ProjectId == projectId {
			hctx.ProjectId = nil
			hctx.ProjectName = ""
		}

		return nil
	})
}

func (s *Service) TerminateSubscriptions(conn pg.Conn, scope eventline.Scope) error {
	var subscriptions eventline.Subscriptions
	if err := subscriptions.LoadAllForUpdate(conn, scope); err != nil {
		return fmt.Errorf("cannot load subscriptions: %w", err)
	}

	for _, subscription := range subscriptions {
		err := s.TerminateSubscription(conn, subscription, true, scope)
		if err != nil {
			return fmt.Errorf("cannot terminate subscription %q: %w",
				subscription.Id, err)
		}
	}

	return nil
}

func (s *Service) DeleteIdentities(conn pg.Conn, scope eventline.Scope) error {
	var identities eventline.Identities
	if err := identities.LoadAllForUpdate(conn, scope); err != nil {
		return fmt.Errorf("cannot load identities: %w", err)
	}

	for _, identity := range identities {
		used, err := identity.IsUsedBySubscription(conn)
		if err != nil {
			return fmt.Errorf("cannot check identity usage: %w", err)
		}

		if used {
			// When an identity is being used by one or more subscriptions, we
			// need to keep it around until these subscriptions have been
			// terminated. We remove the project id and let the subscription
			// worker take care of it. See
			// Service.processTerminatingSubscription.

			identity.ProjectId = nil
			identity.RefreshTime = nil

			if err := identity.UpdateForProjectDeletion(conn); err != nil {
				return fmt.Errorf("cannot update identity %q: %w",
					identity.Id, err)
			}
		} else {
			// Since the identity is not used by any subscription, we can
			// delete it right away.

			if err := identity.Delete(conn); err != nil {
				return fmt.Errorf("cannot delete identity %q: %w",
					identity.Id, err)
			}
		}
	}

	return nil
}

func (s *Service) UpdateProjectConfiguration(projectId eventline.Id, cfg *ProjectConfiguration) error {
	cfg.ProjectSettings.Id = projectId
	cfg.ProjectNotificationSettings.Id = projectId

	return s.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		var project eventline.Project

		if err := project.LoadForUpdate(conn, projectId); err != nil {
			return fmt.Errorf("cannot load project: %w", err)
		}

		if cfg.Project.Name != project.Name {
			exists, err := eventline.ProjectNameExists(conn, cfg.Project.Name)
			if err != nil {
				return fmt.Errorf("cannot check project name existence: %w",
					err)
			} else if exists {
				return &DuplicateProjectNameError{Name: cfg.Project.Name}
			}
		}

		now := time.Now().UTC()

		project.Name = cfg.Project.Name
		project.UpdateTime = now

		if err := project.Update(conn); err != nil {
			return fmt.Errorf("cannot update project: %w", err)
		}

		if err := cfg.ProjectSettings.Update(conn); err != nil {
			return fmt.Errorf("cannot update project settings: %w", err)
		}

		if err := cfg.ProjectNotificationSettings.Update(conn); err != nil {
			return fmt.Errorf("cannot update project notification "+
				"settings: %w", err)
		}

		return nil
	})
}

func DefaultProjectSettings(project *eventline.Project) *eventline.ProjectSettings {
	return &eventline.ProjectSettings{
		Id:         project.Id,
		CodeHeader: "#!/bin/sh\n\nset -eu\n\n",
	}
}

func DefaultProjectNotificationSettings(project *eventline.Project) *eventline.ProjectNotificationSettings {
	return &eventline.ProjectNotificationSettings{
		Id:                     project.Id,
		OnFailedJob:            true,
		OnAbortedJob:           true,
		OnIdentityRefreshError: true,
	}
}
