package docker

import (
	"net/url"

	"github.com/exograd/go-daemon/check"
)

type RunnerCfg struct {
	URI               string `json:"uri,omitempty"`
	CACertificatePath string `json:"ca_certificate_path,omitempty"`
	CertificatePath   string `json:"certificate_path,omitempty"`
	PrivateKeyPath    string `json:"private_key_path,omitempty"`
}

func (cfg *RunnerCfg) Check(c *check.Checker) {
	if cfg.URI != "" {
		uri, err := url.Parse(cfg.URI)
		if err == nil {
			if uri.Scheme != "unix" && uri.Scheme != "tcp" {
				c.AddError("uri", "invalid_uri_scheme",
					"uri scheme must be either unix or tcp")
			}
		} else {
			c.AddError("uri", "invalid_uri_format",
				"string must be a valid uri")
		}
	}
}
