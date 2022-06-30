package github

import (
	"github.com/exograd/eventline/pkg/eventline"
)

type BranchDeletionEvent struct {
	Organization string `json:"organization"`
	Repository   string `json:"repository"`
	Branch       string `json:"branch"`
	Revision     string `json:"revision"`
}

func BranchDeletionEventDef() *eventline.EventDef {
	return eventline.NewEventDef("branch_deletion",
		&BranchDeletionEvent{}, &Parameters{})
}
