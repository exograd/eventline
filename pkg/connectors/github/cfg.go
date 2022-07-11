package github

import (
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/check"
)

type ConnectorCfg struct {
	Enabled       bool   `json:"enabled"`
	WebhookSecret string `json:"webhook_secret,omitempty"`
}

func (c *Connector) DefaultCfg() eventline.ConnectorCfg {
	return &ConnectorCfg{}
}

func (cfg *ConnectorCfg) Check(c *check.Checker) {
	if cfg.Enabled {
		c.CheckStringNotEmpty("webhook_secret", cfg.WebhookSecret)
	}
}
