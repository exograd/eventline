package test

import (
	"github.com/exograd/evgo/pkg/eventline"
	"github.com/exograd/go-daemon/check"
)

type EmptyEvent struct {
}

type EmptyParameters struct {
}

func EmptyEventDef() *eventline.EventDef {
	return eventline.NewEventDef("empty", &EmptyEvent{}, &EmptyParameters{})
}

func (e *EmptyEvent) Check(c *check.Checker) {
}

func (p *EmptyParameters) Check(c *check.Checker) {
}
