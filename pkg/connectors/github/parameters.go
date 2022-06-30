package github

import (
	"strings"

	"github.com/exograd/go-daemon/check"
)

type Parameters struct {
	Organization string `json:"organization"`
	Repository   string `json:"repository,omitempty"` // optional
}

func (p *Parameters) Check(c *check.Checker) {
	c.CheckStringNotEmpty("organization", p.Organization)
}

func (p *Parameters) Target() string {
	if p.Repository == "" {
		return p.Organization
	} else {
		return p.Organization + ":" + p.Repository
	}
}

func (p *Parameters) ParseTarget(s string) {
	idx := strings.IndexByte(s, ':')
	if idx == -1 {
		p.Organization = s
		p.Repository = ""
	} else {
		p.Organization = s[:idx]
		p.Repository = s[idx+1:]
	}
}
