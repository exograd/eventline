package docker

import "github.com/exograd/go-daemon/check"

type RunnerCfg struct {
	URI               string `json:"uri,omitempty"`
	CACertificatePath string `json:"ca_certificate_path,omitempty"`
	CertificatePath   string `json:"certificate_path,omitempty"`
	PrivateKeyPath    string `json:"private_key_path,omitempty"`
}

func (cfg *RunnerCfg) Check(c *check.Checker) {
}
