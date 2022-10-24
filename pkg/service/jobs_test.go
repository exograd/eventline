package service

import (
	"fmt"
	"testing"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/pg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckJobSpec(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	var data string
	var spec *eventline.JobSpec
	var errs check.ValidationErrors

	project := createTestProject(t, "")
	scope := eventline.NewProjectScope(project.Id)

	identity := createTestIdentity(t, "", scope)

	assertInvalid := func(data string, nbErrs int) bool {
		spec = new(eventline.JobSpec)
		require.NoError(spec.ParseYAML([]byte(data)))

		err := testService.Daemon.Pg.WithConn(func(conn pg.Conn) error {
			return testService.ValidateJobSpec(conn, spec, scope)
		})

		if !assert.Error(err) {
			return false
		}

		if !assert.ErrorAs(err, &errs) {
			return false
		}

		if len(errs) != nbErrs {
			assert.Fail(
				fmt.Sprintf("validation yielded %d errors instead of %d",
					len(errs), nbErrs), "%v", errs)
			return false
		}

		return true
	}

	assertError := func(i int, pointer, code string) bool {
		err := errs[i]

		pointerString := err.Pointer.String()

		if pointerString != pointer || err.Code != code {
			assert.Fail(fmt.Sprintf("invalid error: %v", err),
				"expected %s: %s", pointer, code)
			return false
		}

		return true
	}

	assertValid := func(data string) bool {
		spec = new(eventline.JobSpec)
		require.NoError(spec.ParseYAML([]byte(data)))

		err := testService.Daemon.Pg.WithConn(func(conn pg.Conn) error {
			return testService.ValidateJobSpec(conn, spec, scope)
		})

		return assert.NoError(err)
	}

	// Partial documents
	data = `
---
`
	if assertInvalid(data, 1) {
		// It would be nicer to indicate that the name is missing
		assertError(0, "/name", "string_too_small")
	}

	// Simple validation (Job.Check)
	data = `
---
name: "foo"
trigger:
  event: "does_not_exist/hello"
runner:
  name: "does_not_exist_either"
steps: []
`
	if assertInvalid(data, 2) {
		assertError(0, "/trigger/event", "unknown_connector")
		assertError(1, "/runner/name", "invalid_value")
	}

	// Trigger with mandatory parameters
	data = `
---
name: "foo"
trigger:
  event: "time/tick"
  parameters:
    periodic: 60
parameters:
  - name: "a"
    type: "string"
  - name: "b"
    type: "integer"
    default: 42
  - name: "c"
    type: "boolean"
`
	if assertInvalid(data, 1) {
		assertError(0, "/trigger", "invalid_trigger_with_mandatory_parameters")
	}

	// Trigger using disabled connector
	data = `
---
name: "foo"
trigger:
  event: "github/raw"
  parameters:
    organization: "exograd"
`
	if assertInvalid(data, 1) {
		assertError(0, "/trigger/event", "disabled_connector")
	}

	// Unknown identities
	data = fmt.Sprintf(`
---
name: "foo"
runner:
  name: "local"
  parameters: {}
identities: ["foo", %q, "bar"]
`, identity.Name)

	if assertInvalid(data, 2) {
		assertError(0, "/identities/0", "unknown_identity")
		assertError(1, "/identities/2", "unknown_identity")
	}

	// Invalid steps
	data = `
---
name: "foo"
runner:
  name: "local"
  parameters: {}
steps:
  - code: "uname -a"
    command:
      name: "uname"
  - code: "ps aux"
  - label: "foo"
`

	if assertInvalid(data, 2) {
		assertError(0, "/steps/0", "multiple_step_contents")
		assertError(1, "/steps/2", "missing_step_content")
	}

	// Simple oneshot trigger
	data = `
---
name: "foo"
trigger:
  event: "time/tick"
  parameters:
    oneshot: "2030-01-01T12:30:00Z"
`
	assertValid(data)

	// Simple weekly trigger
	data = `
---
name: "foo"
trigger:
  event: "time/tick"
  parameters:
    weekly:
      day: "friday"
      hour: 10
`
	assertValid(data)
}
