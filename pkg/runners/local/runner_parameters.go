package local

import (
	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/ejson"
)

type RunnerParameters struct {
}

func NewRunnerParameters() eventline.RunnerParameters {
	return &RunnerParameters{}
}

func (r *RunnerParameters) ValidateJSON(v *ejson.Validator) {
}
