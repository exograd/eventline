package docker

import (
	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/ejson"
)

type RunnerParameters struct {
	Image       string  `json:"image"`
	CPULimit    float64 `json:"cpu_limit,omitempty"`
	MemoryLimit int     `json:"memory_limit,omitempty"` // megabytes
}

func NewRunnerParameters() eventline.RunnerParameters {
	return &RunnerParameters{}
}

func (r *RunnerParameters) ValidateJSON(v *ejson.Validator) {
	v.CheckStringNotEmpty("image", r.Image)

	// Resource limits are arbitrary, the point is to catch clearly incorrect
	// values. If your system lets you exceed these limits, please contact me,
	// I really want to hear about it!

	if r.CPULimit != 0.0 {
		v.CheckFloatMinMax("cpu_limit", r.CPULimit, 0.1, 1024.0)
	}

	if r.MemoryLimit != 0 {
		v.CheckIntMinMax("memory_limit", r.MemoryLimit, 10, 10_000_000)
	}
}
