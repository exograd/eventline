package github

import (
	"github.com/exograd/eventline/pkg/eventline"
)

type RawEvent struct {
	DeliveryId string      `json:"delivery_id"`
	EventType  string      `json:"event_type"`
	Event      interface{} `json:"event"`
}

func RawEventDef() *eventline.EventDef {
	return eventline.NewEventDef("raw", &RawEvent{}, &Parameters{})
}
