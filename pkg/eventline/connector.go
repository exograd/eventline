package eventline

import (
	"net/url"

	"go.n16f.net/ejson"
	"go.n16f.net/log"
	"go.n16f.net/service/pkg/pg"
)

type ConnectorInitData struct {
	Pg               *pg.Client
	Log              *log.Logger
	WebHTTPServerURI *url.URL
}

type ConnectorCfg interface {
	ejson.Validatable
}

type Connector interface {
	Name() string
	Definition() *ConnectorDef
	DefaultCfg() ConnectorCfg

	Init(ConnectorCfg, ConnectorInitData) error
	Terminate()
}

type SubscribableConnector interface {
	Connector

	Subscribe(pg.Conn, *SubscriptionContext) error
	Unsubscribe(pg.Conn, *SubscriptionContext) error
}

// The optional aspect of the connector is related to events only. But at this
// point I do not have a better idea for a name.
type OptionalConnector interface {
	Connector

	Enabled() bool
}
