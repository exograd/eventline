package eventline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/exograd/go-daemon/pg"
)

type ExecutionContext struct {
	Event      *Event                  `json:"event,omitempty"`
	Parameters map[string]interface{}  `json:"parameters,omitempty"`
	Identities map[string]IdentityData `json:"identities,omitempty"`
}

func (ctx *ExecutionContext) Load(conn pg.Conn, je *JobExecution) error {
	scope := NewProjectScope(je.ProjectId)

	if je.EventId != nil {
		var event Event
		if err := event.Load(conn, *je.EventId, scope); err != nil {
			return fmt.Errorf("cannot load event: %w", err)
		}

		ctx.Event = &event
	}

	ctx.Parameters = je.Parameters

	var identities Identities
	err := identities.LoadByNames(conn, je.JobSpec.IdentityNames(), scope)
	if err != nil {
		return fmt.Errorf("cannot load identities: %w", err)
	}

	return nil
}

func (ctx *ExecutionContext) Write(w io.Writer) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")

	return e.Encode(ctx)
}

func (ctx *ExecutionContext) WriteFile(filePath string) error {
	var buf bytes.Buffer

	if err := ctx.Write(&buf); err != nil {
		return fmt.Errorf("cannot encode context: %w", err)
	}

	return os.WriteFile(filePath, buf.Bytes(), 0600)
}
