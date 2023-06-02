package postgresql

import (
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/dlog"
	"github.com/galdor/go-ejson"
)

type ConnectorCfg struct {
}

type Connector struct {
	Def *eventline.ConnectorDef
	Cfg *ConnectorCfg
	Log *dlog.Logger
}

func NewConnector() *Connector {
	def := eventline.NewConnectorDef("postgresql")

	def.AddIdentity(PasswordIdentityDef())

	return &Connector{
		Def: def,
	}
}

func (cfg *ConnectorCfg) ValidateJSON(v *ejson.Validator) {
}

func (c *Connector) Name() string {
	return "postgresql"
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
