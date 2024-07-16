package time

import (
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/ejson"
	"go.n16f.net/log"
	"go.n16f.net/service/pkg/pg"
)

type ConnectorCfg struct {
	Secret string `json:"secret"`
}

type Connector struct {
	Def *eventline.ConnectorDef
	Cfg *ConnectorCfg
	Log *log.Logger
}

func NewConnector() *Connector {
	def := eventline.NewConnectorDef("time")

	def.Worker = NewWorker()

	def.AddEvent(TickEventDef())

	return &Connector{
		Def: def,
	}
}

func (cfg *ConnectorCfg) ValidateJSON(v *ejson.Validator) {
}

func (c *Connector) Name() string {
	return "time"
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

func (c *Connector) Subscribe(conn pg.Conn, sctx *eventline.SubscriptionContext) error {
	params := sctx.Subscription.Parameters.(*Parameters)

	s := Subscription{
		Id:       sctx.Subscription.Id,
		NextTick: params.FirstTick(),
	}

	if err := s.Insert(conn); err != nil {
		return fmt.Errorf("cannot insert subscription: %w", err)
	}

	return nil
}

func (c *Connector) Unsubscribe(conn pg.Conn, sctx *eventline.SubscriptionContext) error {
	if err := DeleteSubscription(conn, sctx.Subscription.Id); err != nil {
		return fmt.Errorf("cannot delete subscription: %w", err)
	}

	return nil
}
