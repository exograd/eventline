package eventline

import "go.n16f.net/ejson"

type CommandData struct {
	Parameters []*Parameter `json:"parameters,omitempty"`
	Pipelines  []string     `json:"pipelines"` //names
}

func (d *CommandData) ValidateJSON(v *ejson.Validator) {
	v.CheckObjectArray("parameters", d.Parameters)
}
