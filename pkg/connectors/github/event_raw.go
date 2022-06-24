package github

import (
	"github.com/exograd/eventline/pkg/eventline"
)

type RawEvent struct {
	EventType  string      `json:"event_type"`
	DeliveryId string      `json:"delivery_id"`
	Message    interface{} `json:"message"`
}

func RawEventDef() *eventline.EventDef {
	return eventline.NewEventDef("raw", &RawEvent{}, &Parameters{})
}
