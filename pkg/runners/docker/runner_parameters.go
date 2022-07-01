package docker

import (
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/check"
)

type RunnerParameters struct {
	Image string `json:"image"`
}

func NewRunnerParameters() eventline.RunnerParameters {
	return &RunnerParameters{}
}

func (r *RunnerParameters) Check(c *check.Checker) {
	c.CheckStringNotEmpty("image", r.Image)
}
