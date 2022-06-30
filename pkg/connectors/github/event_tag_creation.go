package github

import (
	"github.com/exograd/eventline/pkg/eventline"
)

type TagCreationEvent struct {
	Organization string `json:"organization"`
	Repository   string `json:"repository"`
	Tag          string `json:"tag"`
	Revision     string `json:"revision"`
}

func TagCreationEventDef() *eventline.EventDef {
	return eventline.NewEventDef("tag_creation",
		&TagCreationEvent{}, &Parameters{})
}
