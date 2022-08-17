package ssh

import (
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/check"
)

var hostKeyAlgorithms = []string{
	"ssh-dss",
	"ssh-rsa",
	"ecdsa-sha2-nistp256",
	"ecdsa-sha2-nistp384",
	"ecdsa-sha2-nistp521",
	"ssh-ed25519",
}

type RunnerParameters struct {
	Host             string `json:"host"`
	Port             int    `json:"port"`
	User             string `json:"user"`
	HostKey          []byte `json:"host_key,omitempty"`
	HostKeyAlgorithm string `json:"host_key_algorithm,omitempty"`
}

func NewRunnerParameters() eventline.RunnerParameters {
	return &RunnerParameters{
		Port: 22,
		User: "root",
	}
}

func (r *RunnerParameters) Check(c *check.Checker) {
	c.CheckStringNotEmpty("host", r.Host)
	c.CheckIntMinMax("port", r.Port, 1, 65535)
	c.CheckStringNotEmpty("user", r.User)

	if r.HostKeyAlgorithm == "" {
		c.Check("host_key", r.HostKey == nil, "invalid_value",
			"cannot set a host key without a host key algorithm")
	} else {
		c.CheckStringValue("host_key_algorithm", r.HostKeyAlgorithm,
			hostKeyAlgorithms)
	}
}
