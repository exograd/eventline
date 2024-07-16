package docker

import (
	"net/url"

	"go.n16f.net/ejson"
)

type RunnerCfg struct {
	URI               string                `json:"uri,omitempty"`
	CACertificatePath string                `json:"ca_certificate_path,omitempty"`
	CertificatePath   string                `json:"certificate_path,omitempty"`
	PrivateKeyPath    string                `json:"private_key_path,omitempty"`
	MountPoints       []RunnerCfgMountPoint `json:"mount_points,omitempty"`
}

type RunnerCfgMountPoint struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"read_only,omitempty"`
}

func (cfg *RunnerCfgMountPoint) ValidateJSON(v *ejson.Validator) {
	v.CheckStringNotEmpty("source", cfg.Source)
	v.CheckStringNotEmpty("target", cfg.Target)
}

func (cfg *RunnerCfg) ValidateJSON(v *ejson.Validator) {
	if cfg.URI != "" {
		uri, err := url.Parse(cfg.URI)
		if err == nil {
			if uri.Scheme != "unix" && uri.Scheme != "tcp" {
				v.AddError("uri", "invalid_uri_scheme",
					"uri scheme must be either unix or tcp")
			}
		} else {
			v.AddError("uri", "invalid_uri_format",
				"string must be a valid uri")
		}
	}
}
