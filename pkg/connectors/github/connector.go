package github

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/log"
	"go.n16f.net/service/pkg/pg"
	"go.n16f.net/service/pkg/shttp"
	"github.com/google/go-github/v45/github"
)

type Connector struct {
	Def *eventline.ConnectorDef
	Cfg *ConnectorCfg
	Pg  *pg.Client
	Log *log.Logger

	webHTTPServerURI *url.URL
}

func NewConnector() *Connector {
	def := eventline.NewConnectorDef("github")

	def.AddIdentity(TokenIdentityDef())
	def.AddIdentity(OAuth2IdentityDef())

	def.AddEvent(RawEventDef())
	def.AddEvent(RepositoryCreationEventDef())
	def.AddEvent(RepositoryDeletionEventDef())
	def.AddEvent(TagCreationEventDef())
	def.AddEvent(TagDeletionEventDef())
	def.AddEvent(BranchCreationEventDef())
	def.AddEvent(BranchDeletionEventDef())
	def.AddEvent(PushEventDef())

	return &Connector{
		Def: def,
	}
}

func (c *Connector) Name() string {
	return "github"
}

func (c *Connector) Definition() *eventline.ConnectorDef {
	return c.Def
}

func (c *Connector) Enabled() bool {
	return c.Cfg.Enabled
}

func (c *Connector) Init(ccfg eventline.ConnectorCfg, initData eventline.ConnectorInitData) error {
	c.Cfg = ccfg.(*ConnectorCfg)
	c.Pg = initData.Pg
	c.Log = initData.Log

	c.webHTTPServerURI = initData.WebHTTPServerURI

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

	httpClientCfg := shttp.ClientCfg{
		Log:         c.Log,
		LogRequests: true,
		Header:      header,
	}

	httpClient, err := shttp.NewClient(httpClientCfg)
	if err != nil {
		return nil, fmt.Errorf("cannot create http client: %w", err)
	}

	return github.NewClient(httpClient.Client), nil
}
