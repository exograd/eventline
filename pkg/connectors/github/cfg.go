package github

import "github.com/exograd/go-daemon/check"

type ConnectorCfg struct {
	WebhookKey string `json:"webhook_key"`
}

func (cfg *ConnectorCfg) Check(c *check.Checker) {
	c.CheckStringNotEmpty("webhook_key", cfg.WebhookKey)
}
