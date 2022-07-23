package eventline

import (
	"net/url"

	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/daemon"
	"github.com/exograd/go-daemon/pg"
	"github.com/exograd/go-daemon/dlog"
)

type ConnectorInitData struct {
	Daemon           *daemon.Daemon
	Log              *dlog.Logger
	WebHTTPServerURI *url.URL
}

type ConnectorCfg interface {
	check.Object
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
