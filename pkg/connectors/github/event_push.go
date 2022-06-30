package github

import (
	"github.com/exograd/eventline/pkg/eventline"
)

type PushEvent struct {
	Organization string `json:"organization"`
	Repository   string `json:"repository"`
	Branch       string `json:"branch"`
	OldRevision  string `json:"old_revision,omitempty"`
	NewRevision  string `json:"new_revision"`
}

func PushEventDef() *eventline.EventDef {
	return eventline.NewEventDef("push",
		&PushEvent{}, &Parameters{})
}
