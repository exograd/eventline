package eventline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/exograd/eventline/pkg/utils"
	"github.com/exograd/go-daemon/check"
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
		d.UseNumber()

		if err := d.Decode(&p.Default); err != nil {
			return fmt.Errorf("invalid default: %w", err)
		}
	}

	*pp = Parameter(p)
	return nil
}

func (p *Parameter) Check(c *check.Checker) {
	CheckName(c, "name", p.Name)
	c.CheckStringValue("type", p.Type, ParameterTypeValues)

	if p.Type != ParameterTypeString {
		c.Check("values", len(p.Values) == 0, "unexpected_value",
			"non-string parameters cannot have values")
	}

	if p.Default != nil {
		switch p.Type {
		case ParameterTypeNumber:
			_, ok := p.Default.(json.Number)
			c.Check("default", ok, "invalid_type",
				"default value must be a number")

		case ParameterTypeInteger:
			var err error
			number, ok := p.Default.(json.Number)
			if ok {
				_, err = number.Int64()
			}
			c.Check("default", ok && err == nil, "invalid_type",
				"default value must be an integer")

		case ParameterTypeString:
			defaultString, isString := p.Default.(string)
			c.Check("default", isString, "invalid_type",
				"default value must be a string")

			if p.Values != nil {
				c.Check("default",
					utils.StringsContain(p.Values, defaultString),
					"invalid_value",
					"default value is not one of the valid values")
			}

		case ParameterTypeBoolean:
			_, isBool := p.Default.(bool)
			c.Check("default", isBool, "invalid_type",
				"default value must be a boolean")
		}
	}
}

func (ps Parameters) CheckValues(c *check.Checker, token string, values map[string]interface{}) {
	c.WithChild(token, func() {
		for name, value := range values {
			param := ps.Parameter(name)
			if param == nil {
				// Ideally we would like to point at the key in the object,
				// but JSON pointers can only target values.
				c.AddError(name, "unknown_parameter", "unknown parameter")
				continue
			}

			values[name] = param.CheckValue(c, name, value)
		}
	})

	for _, p := range ps {
		if p.Default != nil {
			continue
		}

		if _, found := values[p.Name]; !found {
			c.AddError(token, "missing_parameter", "missing parameter %q",
				p.Name)
		}
	}
}

func (p *Parameter) CheckValue(c *check.Checker, token string, value interface{}) interface{} {
	switch p.Type {
	case ParameterTypeNumber:
		return p.checkValueNumber(c, token, value)
	case ParameterTypeInteger:
		return p.checkValueInteger(c, token, value)
	case ParameterTypeString:
		return p.checkValueString(c, token, value)
	case ParameterTypeBoolean:
		return p.checkValueBoolean(c, token, value)
	default:
		utils.Panicf("unhandled parameter type %v", p.Type)
	}

	return nil
}

func (p *Parameter) checkValueNumber(c *check.Checker, token string, value interface{}) interface{} {
	number, ok := value.(json.Number)
	if !ok {
		c.AddError(token, "invalid_number", "value is not a number")
		return nil
	}

	if f64, err := number.Float64(); err == nil {
		return f64
	} else if i64, err := number.Int64(); err == nil {
		return i64
	}

	// Not supposed to happen
	c.AddError(token, "invalid_number", "value is not a number")

	return nil
}

func (p *Parameter) checkValueInteger(c *check.Checker, token string, value interface{}) interface{} {
	number, ok := value.(json.Number)
	if !ok {
		c.AddError(token, "invalid_integer", "value is not an integer")
		return nil
	}

	i64, err := number.Int64()
	if err != nil {
		c.AddError(token, "invalid_integer", "value is not an integer")
		return nil
	}

	return i64
}

func (p *Parameter) checkValueString(c *check.Checker, token string, value interface{}) interface{} {
	s, ok := value.(string)
	c.Check(token, ok, "invalid_string", "value is not a string")

	return s
}

func (p *Parameter) checkValueBoolean(c *check.Checker, token string, value interface{}) interface{} {
	b, ok := value.(bool)
	c.Check(token, ok, "invalid_boolean", "value is not a boolean")

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
