package utils

import (
	"bytes"
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

func YAMLValueToJSONValue(yamlValue interface{}) (interface{}, error) {
	// For some reason, go-yaml will return objects as map[string]interface{}
	// if all keys are strings, and as map[interface{}]interface{} if not. So
	// we have to handle both.

	var jsonValue interface{}

	switch v := yamlValue.(type) {
	case []interface{}:
		array := make([]interface{}, len(v))

		for i, yamlElement := range v {
			jsonElement, err := YAMLValueToJSONValue(yamlElement)
			if err != nil {
				return nil, err
			}

			array[i] = jsonElement
		}

		jsonValue = array

	case map[interface{}]interface{}:
		object := make(map[string]interface{})

		for key, yamlEntry := range v {
			keyString, ok := key.(string)
			if !ok {
				return nil,
					fmt.Errorf("object key \"%v\" is not a string", key)
			}

			jsonEntry, err := YAMLValueToJSONValue(yamlEntry)
			if err != nil {
				return nil, err
			}

			object[keyString] = jsonEntry
		}

		jsonValue = object

	case map[string]interface{}:
		object := make(map[string]interface{})

		for key, yamlEntry := range v {
			jsonEntry, err := YAMLValueToJSONValue(yamlEntry)
			if err != nil {
				return nil, err
			}

			object[key] = jsonEntry
		}

		jsonValue = object

	default:
		jsonValue = yamlValue
	}

	return jsonValue, nil
}

func YAMLEncode(jsonValue interface{}) ([]byte, error) {
	// Infortunately go-yaml has no way to handle json tags, and it has been
	// an issue for more than three years
	// (https://github.com/go-yaml/yaml/issues/424).

	jsonData, err := json.Marshal(jsonValue)
	if err != nil {
		return nil, err
	}

	var yamlValue interface{}
	if err := yaml.Unmarshal(jsonData, &yamlValue); err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	if err := encoder.Encode(yamlValue); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
