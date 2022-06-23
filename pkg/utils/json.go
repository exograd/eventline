package utils

import (
	"bytes"
	"encoding/json"
)

func JSONEncode(value interface{}) ([]byte, error) {
	var buf bytes.Buffer

	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(value); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
