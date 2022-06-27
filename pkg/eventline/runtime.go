package eventline

import (
	"encoding/json"
	"fmt"

	"github.com/exograd/go-daemon/check"
)

type Runtime struct {
	Name          string            `json:"name"`
	Parameters    RuntimeParameters `json:"-"`
	RawParameters json.RawMessage   `json:"parameters"`
}

type RuntimeParameters interface {
	check.Object
}

func (r *Runtime) Check(c *check.Checker) {
	runtimeNames := make([]string, 0, len(RunnerDefs))
	for name := range RunnerDefs {
		runtimeNames = append(runtimeNames, name)
	}

	if c.CheckStringValue("name", r.Name, runtimeNames) {
		c.CheckObject("parameters", r.Parameters)
	} else {
		// If the runtime name is invalid, we do not want to check the content
		// of the parameters, but we at least can check that it is there.
		c.Check("parameters", r.Parameters != nil, "missing_value",
			"missing value")
	}
}

func (pr *Runtime) MarshalJSON() ([]byte, error) {
	type Runtime2 Runtime

	r := Runtime2(*pr)

	params, err := json.Marshal(r.Parameters)
	if err != nil {
		return nil, fmt.Errorf("cannot encode parameters: %w", err)
	}

	r.RawParameters = params

	return json.Marshal(r)
}

func (pr *Runtime) UnmarshalJSON(data []byte) error {
	type Runtime2 Runtime

	r := Runtime2(*pr)
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}

	var params RuntimeParameters

	switch r.Name {
	case "local":
		params = &LocalRuntime{}
	}

	// Note that at this moment, Check has not been called yet, so the runtime
	// name may be invalid. It is better to let Check validate it so that
	// users get full validation errors.

	if params != nil && r.RawParameters != nil {
		if err := json.Unmarshal(r.RawParameters, &params); err != nil {
			return fmt.Errorf("invalid runtime parameters: %w", err)
		}
	}

	r.Parameters = params

	*pr = Runtime(r)
	return nil
}
