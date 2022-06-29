package eventline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/exograd/eventline/pkg/utils"
)

func JSONFields(value interface{}) (map[string]string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("cannot encode value: %w", err)
	}

	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()

	var obj map[string]interface{}
	if err := d.Decode(&obj); err != nil {
		return nil, fmt.Errorf("cannot decode json data: %w", err)
	}

	fields := make(map[string]string)
	jsonFields(obj, "", fields)

	return fields, nil
}

func jsonFields(value interface{}, key string, fields map[string]string) {
	switch v := value.(type) {
	case map[string]interface{}:
		for name, child := range v {
			var childKey string
			if key == "" {
				childKey = name
			} else {
				childKey = key + "/" + name
			}

			jsonFields(child, childKey, fields)
		}

	case []interface{}:
		for i, child := range v {
			var childKey string
			if key == "" {
				childKey = strconv.Itoa(i)
			} else {
				childKey = key + "/" + strconv.Itoa(i)
			}

			jsonFields(child, childKey, fields)
		}

	case bool:
		fields[key] = strconv.FormatBool(v)

	case string:
		fields[key] = v

	case json.Number:
		fields[key] = v.String()

	case nil:
		fields[key] = "null"

	default:
		utils.Panicf("unhandled json value %#v", value)
	}
}
