package eventline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/exograd/eventline/pkg/utils"
	"go.n16f.net/ejson"
	"go.n16f.net/program"
)

var (
	reParameterLabelRE = regexp.MustCompile(`_+`)
)

type ParameterType string

const (
	ParameterTypeNumber  ParameterType = "number"
	ParameterTypeInteger ParameterType = "integer"
	ParameterTypeString  ParameterType = "string"
	ParameterTypeBoolean ParameterType = "boolean"
)

var ParameterTypeValues = []ParameterType{
	ParameterTypeNumber,
	ParameterTypeInteger,
	ParameterTypeString,
	ParameterTypeBoolean,
}

type Parameter struct {
	Name        string          `json:"name"`
	Type        ParameterType   `json:"type"`
	Values      []string        `json:"values,omitempty"`
	Default     interface{}     `json:"-"`
	RawDefault  json.RawMessage `json:"default,omitempty"`
	Description string          `json:"description,omitempty"`
	Environment string          `json:"environment,omitempty"`
}

type Parameters []*Parameter

func (pp *Parameter) MarshalJSON() ([]byte, error) {
	type Parameter2 Parameter

	p := Parameter2(*pp)
	def, err := json.Marshal(p.Default)
	if err != nil {
		return nil, fmt.Errorf("cannot encode parameters: %w", err)
	}

	p.RawDefault = def

	return json.Marshal(p)
}

func (pp *Parameter) UnmarshalJSON(data []byte) error {
	type Parameter2 Parameter

	p := Parameter2(*pp)
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}

	if p.RawDefault != nil {
		d := json.NewDecoder(bytes.NewReader(p.RawDefault))
		d.DisallowUnknownFields()
		d.UseNumber()

		if err := d.Decode(&p.Default); err != nil {
			return fmt.Errorf("invalid default: %w", err)
		}
	}

	*pp = Parameter(p)
	return nil
}

func (p *Parameter) ValidateJSON(v *ejson.Validator) {
	CheckName(v, "name", p.Name)
	v.CheckStringValue("type", p.Type, ParameterTypeValues)

	if p.Type != ParameterTypeString {
		v.Check("values", len(p.Values) == 0, "unexpected_value",
			"non-string parameters cannot have values")
	}

	if p.Default != nil {
		switch p.Type {
		case ParameterTypeNumber:
			_, ok := p.Default.(json.Number)
			v.Check("default", ok, "invalid_type",
				"default value must be a number")

		case ParameterTypeInteger:
			var err error
			number, ok := p.Default.(json.Number)
			if ok {
				_, err = number.Int64()
			}
			v.Check("default", ok && err == nil, "invalid_type",
				"default value must be an integer")

		case ParameterTypeString:
			defaultString, isString := p.Default.(string)
			v.Check("default", isString, "invalid_type",
				"default value must be a string")

			if p.Values != nil {
				v.Check("default",
					utils.StringsContain(p.Values, defaultString),
					"invalid_value",
					"default value is not one of the valid values")
			}

		case ParameterTypeBoolean:
			_, isBool := p.Default.(bool)
			v.Check("default", isBool, "invalid_type",
				"default value must be a boolean")
		}
	}
}

func (ps Parameters) CheckValues(v *ejson.Validator, token string, values map[string]interface{}) {
	v.WithChild(token, func() {
		for name, value := range values {
			param := ps.Parameter(name)
			if param == nil {
				// Ideally we would like to point at the key in the object,
				// but JSON pointers can only target values.
				v.AddError(name, "unknown_parameter", "unknown parameter")
				continue
			}

			values[name] = param.CheckValue(v, name, value)
		}
	})

	for _, p := range ps {
		if _, found := values[p.Name]; !found {
			if p.Default == nil {
				v.AddError(token, "missing_parameter", "missing parameter %q",
					p.Name)
			} else {
				values[p.Name] = p.Default
			}
		}
	}
}

func (p *Parameter) CheckValue(v *ejson.Validator, token string, value interface{}) interface{} {
	switch p.Type {
	case ParameterTypeNumber:
		return p.checkValueNumber(v, token, value)
	case ParameterTypeInteger:
		return p.checkValueInteger(v, token, value)
	case ParameterTypeString:
		return p.checkValueString(v, token, value)
	case ParameterTypeBoolean:
		return p.checkValueBoolean(v, token, value)
	default:
		program.Panicf("unhandled parameter type %v", p.Type)
	}

	return nil
}

func (p *Parameter) checkValueNumber(v *ejson.Validator, token string, value interface{}) interface{} {
	number, ok := value.(json.Number)
	if !ok {
		v.AddError(token, "invalid_number", "value is not a number")
		return nil
	}

	if f64, err := number.Float64(); err == nil {
		return f64
	} else if i64, err := number.Int64(); err == nil {
		return i64
	}

	// Not supposed to happen
	v.AddError(token, "invalid_number", "value is not a number")

	return nil
}

func (p *Parameter) checkValueInteger(v *ejson.Validator, token string, value interface{}) interface{} {
	number, ok := value.(json.Number)
	if !ok {
		v.AddError(token, "invalid_integer", "value is not an integer")
		return nil
	}

	i64, err := number.Int64()
	if err != nil {
		v.AddError(token, "invalid_integer", "value is not an integer")
		return nil
	}

	return i64
}

func (p *Parameter) checkValueString(v *ejson.Validator, token string, value interface{}) interface{} {
	s, ok := value.(string)
	v.Check(token, ok, "invalid_string", "value is not a string")

	if p.Values != nil {
		v.CheckStringValue(token, s, p.Values)
	}

	return s
}

func (p *Parameter) checkValueBoolean(v *ejson.Validator, token string, value interface{}) interface{} {
	b, ok := value.(bool)
	v.Check(token, ok, "invalid_boolean", "value is not a boolean")

	return b
}

func (ps Parameters) Parameter(name string) *Parameter {
	for _, p := range ps {
		if p.Name == name {
			return p
		}
	}

	return nil
}

func (p *Parameter) Label() string {
	// Used in the form template
	label := reParameterLabelRE.ReplaceAllString(p.Name, " ")
	return utils.Capitalize(label)
}

func (p *Parameter) ValueString(value interface{}) (s string) {
	switch p.Type {
	case ParameterTypeNumber:
		return fmt.Sprintf("%v", value)

	case ParameterTypeInteger:
		return fmt.Sprintf("%v", value)

	case ParameterTypeString:
		return value.(string)

	case ParameterTypeBoolean:
		if value.(bool) == true {
			return "true"
		} else {
			return "false"
		}

	default:
		program.Panicf("unhandled parameter type %v", p.Type)
	}

	return
}
