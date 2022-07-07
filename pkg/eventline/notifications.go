package eventline

import (
	"github.com/exograd/go-daemon/check"
)

type NotificationsCfg struct {
	SMTPServer    *SMTPServerCfg `json:"smtp_server"`
	FromAddress   string         `json:"from_address"`
	SubjectPrefix string         `json:"subject_prefix"`
}

type SMTPServerCfg struct {
	Address  string `json:"address"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func (cfg *NotificationsCfg) Check(c *check.Checker) {
	c.CheckObject("smtp_server", cfg.SMTPServer)
	c.CheckStringNotEmpty("from_address", cfg.FromAddress)
}

func (cfg *SMTPServerCfg) Check(c *check.Checker) {
	c.CheckStringNotEmpty("address", cfg.Address)
}

func DefaultNotificationsCfg() *NotificationsCfg {
	return &NotificationsCfg{
		SMTPServer: &SMTPServerCfg{
			Address: "localhost:25",
		},

		FromAddress:   "no-reply@localhost",
		SubjectPrefix: "[eventline] ",
	}
}
