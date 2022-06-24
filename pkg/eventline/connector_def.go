package eventline

import (
	"fmt"

	"github.com/exograd/eventline/pkg/utils"
)

type UnknownConnectorDefError struct {
	Name string
}

func (err UnknownConnectorDefError) Error() string {
	return fmt.Sprintf("unknown connector %q", err.Name)
}

type ConnectorDef struct {
	Name string

	Identities map[string]*IdentityDef
	Events     map[string]*EventDef

	Worker WorkerBehaviour
}

func NewConnectorDef(name string) *ConnectorDef {
	return &ConnectorDef{
		Name: name,

		Identities: make(map[string]*IdentityDef),
		Events:     make(map[string]*EventDef),
	}
}

func (c *ConnectorDef) AddIdentity(idef *IdentityDef) {
	idef.DataDef = idef.Data.Def()

	c.Identities[idef.Type] = idef
}

func (c *ConnectorDef) IdentityExists(typeName string) (exists bool) {
	_, exists = c.Identities[typeName]
	return
}

func (c *ConnectorDef) Identity(typeName string) *IdentityDef {
	def, found := c.Identities[typeName]
	if !found {
		utils.Panicf("unknown identity %q in connector %q", typeName, c.Name)
	}

	return def
}

func (c *ConnectorDef) ValidateIdentityType(typeName string) error {
	if _, found := c.Identities[typeName]; !found {
		return &UnknownIdentityDefError{Connector: c.Name, Type: typeName}
	}

	return nil
}

func (c *ConnectorDef) AddEvent(edef *EventDef) {
	c.Events[edef.Name] = edef
}

func (c *ConnectorDef) EventExists(typeName string) (exists bool) {
	_, exists = c.Events[typeName]
	return
}

func (c *ConnectorDef) Event(typeName string) *EventDef {
	def, found := c.Events[typeName]
	if !found {
		utils.Panicf("unknown event %q in connector %q", typeName, c.Name)
	}

	return def
}

func (c *ConnectorDef) ValidateEventName(name string) error {
	if _, found := c.Events[name]; !found {
		return &UnknownEventDefError{Connector: c.Name, Name: name}
	}

	return nil
}
