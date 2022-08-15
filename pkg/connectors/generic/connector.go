package generic

import (
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/dlog"
)

type ConnectorCfg struct {
	Secret string `json:"secret"`
}

type Connector struct {
	Def *eventline.ConnectorDef
	Cfg *ConnectorCfg
	Log *dlog.Logger
}

func NewConnector() *Connector {
	def := eventline.NewConnectorDef("generic")

	def.AddIdentity(PasswordIdentityDef())
	def.AddIdentity(APIKeyIdentityDef())
	def.AddIdentity(SSHKeyIdentityDef())
	def.AddIdentity(OAuth2IdentityDef())
	def.AddIdentity(GPGKeyIdentityDef())

	return &Connector{
		Def: def,
	}
}

func (cfg *ConnectorCfg) Check(c *check.Checker) {
}

func (c *Connector) Name() string {
	return "generic"
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
