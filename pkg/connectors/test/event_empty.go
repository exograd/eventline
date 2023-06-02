package test

import (
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/galdor/go-ejson"
)

type EmptyEvent struct {
}

type EmptyParameters struct {
}

func EmptyEventDef() *eventline.EventDef {
	return eventline.NewEventDef("empty", &EmptyEvent{}, &EmptyParameters{})
}

func (e *EmptyEvent) ValidateJSON(v *ejson.Validator) {
}

func (p *EmptyParameters) ValidateJSON(v *ejson.Validator) {
}
