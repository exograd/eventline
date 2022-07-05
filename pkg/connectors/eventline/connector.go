package eventline

import (
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-log"
)

type ConnectorCfg struct {
}

type Connector struct {
	Def *eventline.ConnectorDef
	Cfg *ConnectorCfg
	Log *log.Logger
}

func NewConnector() *Connector {
	def := eventline.NewConnectorDef("eventline")

	def.AddIdentity(APIKeyIdentityDef())

	return &Connector{
		Def: def,
	}
}

func (cfg *ConnectorCfg) Check(c *check.Checker) {
}

func (c *Connector) Name() string {
	return "eventline"
}

func (c *Connector) Definition() *eventline.ConnectorDef {
	return c.Def
}

func (c *Connector) DefaultCfg() eventline.ConnectorCfg {
	return &ConnectorCfg{}
}

func (c *Connector) Init(ccfg eventline.ConnectorCfg, initData eventline.ConnectorInitData) error {
	c.Cfg = ccfg.(*ConnectorCfg)
	c.Log = initData.Log

	return nil
}

func (c *Connector) Terminate() {
}
