package service

import (
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/service/pkg/pg"
	"go.n16f.net/uuid"
)

func (s *Service) AddFavouriteJob(conn pg.Conn, jobId uuid.UUID, scope eventline.Scope) error {
	accountProjectScope := scope.(*eventline.AccountProjectScope)
	projectScope := eventline.NewProjectScope(accountProjectScope.ProjectId)

	var job eventline.Job
	if err := job.Load(conn, jobId, projectScope); err != nil {
		return fmt.Errorf("cannot load job: %w", err)
	}

	fj := eventline.FavouriteJob{
		AccountId: accountProjectScope.AccountId,
		ProjectId: accountProjectScope.ProjectId,
		JobId:     job.Id,
	}

	return fj.Upsert(conn)
}

func (s *Service) RemoveFavouriteJob(conn pg.Conn, jobId uuid.UUID, scope eventline.Scope) error {
	accountProjectScope := scope.(*eventline.AccountProjectScope)
	projectScope := eventline.NewProjectScope(accountProjectScope.ProjectId)

	var job eventline.Job
	if err := job.Load(conn, jobId, projectScope); err != nil {
		return fmt.Errorf("cannot load job: %w", err)
	}

	return eventline.DeleteFavouriteJob(conn, jobId, scope)
}
