package github

import (
	"github.com/exograd/eventline/pkg/eventline"
)

type RepositoryCreationEvent struct {
	Organization string `json:"organization"`
	Repository   string `json:"repository"`
}

func RepositoryCreationEventDef() *eventline.EventDef {
	return eventline.NewEventDef("repository_creation",
		&RepositoryCreationEvent{}, &Parameters{})
}
