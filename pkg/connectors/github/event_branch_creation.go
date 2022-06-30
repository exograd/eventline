package github

import (
	"github.com/exograd/eventline/pkg/eventline"
)

type BranchCreationEvent struct {
	Organization string `json:"organization"`
	Repository   string `json:"repository"`
	Branch       string `json:"branch"`
	Revision     string `json:"revision"`
}

func BranchCreationEventDef() *eventline.EventDef {
	return eventline.NewEventDef("branch_creation",
		&BranchCreationEvent{}, &Parameters{})
}
