package docker

import (
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/check"
)

type RunnerParameters struct {
}

func NewRunnerParameters() eventline.RunnerParameters {
	return &RunnerParameters{}
}

func (r *RunnerParameters) Check(c *check.Checker) {
}
