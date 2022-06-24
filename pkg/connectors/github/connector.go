package github

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/daemon"
	"github.com/exograd/go-daemon/dhttp"
	"github.com/exograd/go-daemon/pg"
	"github.com/exograd/go-log"
	"github.com/google/go-github/v45/github"
)

type ConnectorCfg struct {
	WebhookKey string `json:"webhook_key"`
}

type Connector struct {
	Def    *eventline.ConnectorDef
	Cfg    *ConnectorCfg
	Daemon *daemon.Daemon
	Log    *log.Logger

	webHTTPServerURI *url.URL
	webhookKey       []byte
}

func NewConnector() *Connector {
	def := eventline.NewConnectorDef("github")

	def.AddIdentity(TokenIdentityDef())
	def.AddIdentity(OAuth2IdentityDef())

	def.AddEvent(RawEventDef())

	return &Connector{
		Def: def,
	}
}

func (cfg *ConnectorCfg) Check(c *check.Checker) {
	c.CheckStringNotEmpty("webhook_key", cfg.WebhookKey)
}

func (c *Connector) Name() string {
	return "github"
}

func (c *Connector) Definition() *eventline.ConnectorDef {
	return c.Def
}

func (c *Connector) DefaultCfg() eventline.ConnectorCfg {
	return &ConnectorCfg{}
}

func (c *Connector) Init(ccfg eventline.ConnectorCfg, initData eventline.ConnectorInitData) error {
	c.Cfg = ccfg.(*ConnectorCfg)
	c.Daemon = initData.Daemon
	c.Log = initData.Log

	c.webHTTPServerURI = initData.WebHTTPServerURI

	webhookKey, err := hex.DecodeString(c.Cfg.WebhookKey)
	if err != nil {
		return fmt.Errorf("invalid webhook key: %w", err)
	}
	c.webhookKey = webhookKey

	return nil
}

func (c *Connector) Terminate() {
}

func (c *Connector) Subscribe(conn pg.Conn, sctx *eventline.SubscriptionContext) error {
	params := sctx.Subscription.Parameters.(*Parameters)

	hookId, err := c.MaybeCreateHook(conn, params, sctx.Identity)
	if err != nil {
		return fmt.Errorf("cannot create hook: %w", err)
	}

	s := Subscription{
		Id:           sctx.Subscription.Id,
		Organization: params.Organization,
		Repository:   params.Repository,
		HookId:       *hookId,
	}

	if err := s.Insert(conn); err != nil {
		return fmt.Errorf("cannot insert subscription: %w", err)
	}

	return nil
}

func (c *Connector) Unsubscribe(conn pg.Conn, sctx *eventline.SubscriptionContext) error {
	params := sctx.Subscription.Parameters.(*Parameters)

	var subscription Subscription
	err := subscription.LoadForUpdate(conn, sctx.Subscription.Id)
	if err != nil {
		return fmt.Errorf("cannot load subscription: %w", err)
	}

	err = c.MaybeDeleteHook(conn, params, sctx.Identity, subscription.HookId)
	if err != nil {
		return fmt.Errorf("cannot delete hook: %w", err)
	}

	if err := subscription.Delete(conn); err != nil {
		return fmt.Errorf("cannot delete subscription: %w", err)
	}

	return nil
}

func (c *Connector) NewClient(identity *eventline.Identity) (*github.Client, error) {
	header := make(http.Header)

	if identity != nil {
		switch idata := identity.Data.(type) {
		case *TokenIdentity:
			credentials := []byte(idata.Username + ":" + idata.Token)
			header.Add("Authorization",
				"Basic "+base64.StdEncoding.EncodeToString(credentials))

		case *OAuth2Identity:
			header.Add("Authorization", "token "+idata.AccessToken)

		default:
			return nil, fmt.Errorf("unsupported identity")
		}
	}

	httpClientCfg := dhttp.ClientCfg{
		Log:         c.Log,
		LogRequests: true,
		Header:      header,
	}

	httpClient, err := dhttp.NewClient(httpClientCfg)
	if err != nil {
		return nil, fmt.Errorf("cannot create http client: %w", err)
	}

	return github.NewClient(httpClient.Client), nil
}
