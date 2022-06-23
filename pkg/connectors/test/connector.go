package test

import "github.com/exograd/evgo/pkg/eventline"

func ConnectorDef() *eventline.ConnectorDef {
	def := eventline.NewConnectorDef("test")

	def.AddIdentity(PasswordIdentityDef())

	def.AddEvent(EmptyEventDef())

	return def
}
