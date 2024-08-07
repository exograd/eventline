package eventline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"go.n16f.net/service/pkg/pg"
)

type ExecutionContext struct {
	Event      *Event                 `json:"event,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Identities map[string]*Identity   `json:"identities,omitempty"`
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

	now := time.Now().UTC()

	ctx.Identities = make(map[string]*Identity)
	for _, identity := range identities {
		identity.LastUseTime = &now

		if err := identity.UpdateLastUseTime(conn); err != nil {
			return fmt.Errorf("cannot update identity %q: %w",
				identity.Id, err)
		}

		ctx.Identities[identity.Name] = identity
	}

	return nil
}

func (ctx *ExecutionContext) Write(w io.Writer) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")

	return e.Encode(ctx)
}

func (ctx *ExecutionContext) Encode() ([]byte, error) {
	var buf bytes.Buffer

	if err := ctx.Write(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
