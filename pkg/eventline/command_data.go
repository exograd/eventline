package eventline

import "github.com/galdor/go-ejson"

type CommandData struct {
	Parameters []*Parameter `json:"parameters,omitempty"`
	Pipelines  []string     `json:"pipelines"` //names
}

func (d *CommandData) ValidateJSON(v *ejson.Validator) {
	v.CheckObjectArray("parameters", d.Parameters)
}
