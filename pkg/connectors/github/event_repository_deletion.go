package github

import (
	"github.com/exograd/eventline/pkg/eventline"
)

type RepositoryDeletionEvent struct {
	Organization string `json:"organization"`
	Repository   string `json:"repository"`
}

func RepositoryDeletionEventDef() *eventline.EventDef {
	return eventline.NewEventDef("repository_deletion",
		&RepositoryDeletionEvent{}, &Parameters{})
}
