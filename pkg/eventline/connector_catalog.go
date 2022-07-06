package eventline

import (
	"github.com/exograd/eventline/pkg/utils"
	"github.com/exograd/go-daemon/check"
)

var Connectors = map[string]Connector{}

func ValidateConnectorName(name string) error {
	if _, found := Connectors[name]; !found {
		return &UnknownConnectorDefError{Name: name}
	}

	return nil
}

func ConnectorExists(name string) (exists bool) {
	_, exists = Connectors[name]
	return
}

func GetConnector(name string) Connector {
	c, found := Connectors[name]
	if !found {
		utils.Panicf("unknown connector %q", name)
	}

	return c
}

func GetConnectorDef(name string) *ConnectorDef {
	c := GetConnector(name)
	return c.Definition()
}

func CheckConnectorName(c *check.Checker, token string, cname string) bool {
	if c.CheckStringNotEmpty(token, cname) {
		return c.Check(token, ConnectorExists(cname),
			"unknown_connector", "unknown connector %q", cname)
	}

	return true
}

func IdentityExists(cname, itype string) bool {
	c, found := Connectors[cname]
	if !found {
		return false
	}

	cdef := c.Definition()

	return cdef.IdentityExists(itype)
}

func CheckIdentityName(c *check.Checker, token string, cname, itype string) {
	c.CheckStringNotEmpty(token, itype)

	if itype != "" {
		c.Check(token, IdentityExists(cname, itype),
			"unknown_identity", "unknown identity %q in connector %q",
			itype, cname)
	}
}

func EventExists(cname, name string) bool {
	c, found := Connectors[cname]
	if !found {
		return false
	}

	cdef := c.Definition()

	return cdef.EventExists(name)
}

func EventDefExists(ref EventRef) bool {
	return EventExists(ref.Connector, ref.Event)
}

func GetEventDef(ref EventRef) *EventDef {
	cdef := GetConnectorDef(ref.Connector)
	return cdef.Event(ref.Event)
}

func CheckEventName(c *check.Checker, token string, cname, name string) bool {
	if c.CheckStringNotEmpty(token, name) == false {
		return false
	}

	return c.Check(token, EventExists(cname, name),
		"unknown_event", "unknown event %q in connector %q", name, cname)
}
