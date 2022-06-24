package time

import (
	"github.com/exograd/eventline/pkg/eventline"
)

type TickEvent struct {
}

func TickEventDef() *eventline.EventDef {
	return eventline.NewEventDef("tick", &TickEvent{}, &Parameters{})
}
