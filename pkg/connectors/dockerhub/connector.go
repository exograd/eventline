package dockerhub

import (
	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/ejson"
	"go.n16f.net/log"
)

type ConnectorCfg struct {
}

type Connector struct {
	Def *eventline.ConnectorDef
	Cfg *ConnectorCfg
	Log *log.Logger
}

func NewConnector() *Connector {
	def := eventline.NewConnectorDef("dockerhub")

	def.AddIdentity(PasswordIdentityDef())
	def.AddIdentity(TokenIdentityDef())

	return &Connector{
		Def: def,
	}
}

func (cfg *ConnectorCfg) ValidateJSON(v *ejson.Validator) {
}

func (c *Connector) Name() string {
	return "dockerhub"
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
