package github

import (
	"github.com/exograd/eventline/pkg/eventline"
)

type TagDeletionEvent struct {
	Organization string `json:"organization"`
	Repository   string `json:"repository"`
	Tag          string `json:"tag"`
	Revision     string `json:"revision"`
}

func TagDeletionEventDef() *eventline.EventDef {
	return eventline.NewEventDef("tag_deletion",
		&TagDeletionEvent{}, &Parameters{})
}
