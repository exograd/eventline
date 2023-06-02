package eventline

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/galdor/go-ejson"
)

type EventRef struct {
	Connector string
	Event     string
}

func (ref EventRef) String() string {
	return ref.Connector + "/" + ref.Event
}

func (ref *EventRef) Parse(s string) error {
	idx := strings.IndexByte(s, '/')
	if idx == -1 {
		return fmt.Errorf("invalid format")
	}

	if idx == 0 {
		return fmt.Errorf("empty connector name")
	}

	if idx == len(s)-1 {
		return fmt.Errorf("empty event name")
	}

	ref.Connector = s[:idx]
	ref.Event = s[idx+1:]

	return nil
}

func (ref EventRef) MarshalJSON() ([]byte, error) {
	return json.Marshal(ref.String())
}

func (ref *EventRef) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	if err := ref.Parse(s); err != nil {
		return fmt.Errorf("invalid event reference %q: %w", s, err)
	}

	return nil
}

func CheckEventRef(v *ejson.Validator, token string, ref EventRef) bool {
	if CheckConnectorName(v, token, ref.Connector) == false {
		return false
	}

	return CheckEventName(v, token, ref.Connector, ref.Event)
}
