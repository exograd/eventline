package eventline

import "github.com/exograd/go-daemon/check"

type CommandData struct {
	Parameters []*Parameter `json:"parameters,omitempty"`
	Pipelines  []string     `json:"pipelines"` //names
}

func (d *CommandData) Check(c *check.Checker) {
	c.CheckObjectArray("parameters", d.Parameters)
}
