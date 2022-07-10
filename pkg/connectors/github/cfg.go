package github

import (
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/check"
)

type ConnectorCfg struct {
	Enabled    bool   `json:"enabled"`
	WebhookKey string `json:"webhook_key,omitempty"`
}

func (c *Connector) DefaultCfg() eventline.ConnectorCfg {
	return &ConnectorCfg{}
}

func (cfg *ConnectorCfg) Check(c *check.Checker) {
	if cfg.Enabled {
		c.CheckStringNotEmpty("webhook_key", cfg.WebhookKey)
	}
}
