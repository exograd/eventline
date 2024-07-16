package github

import (
	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/ejson"
)

type ConnectorCfg struct {
	Enabled       bool   `json:"enabled"`
	WebhookSecret string `json:"webhook_secret,omitempty"`
}

func (c *Connector) DefaultCfg() eventline.ConnectorCfg {
	return &ConnectorCfg{}
}

func (cfg *ConnectorCfg) ValidateJSON(v *ejson.Validator) {
	if cfg.Enabled {
		v.CheckStringNotEmpty("webhook_secret", cfg.WebhookSecret)
	}
}
