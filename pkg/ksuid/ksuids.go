package ksuid

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

type KSUIDs []KSUID

func (ids KSUIDs) Strings() []string {
	ss := make([]string, len(ids))

	for i, u := range ids {
		ss[i] = u.String()
	}

	return ss
}

func (pids *KSUIDs) Parse(ss []string) error {
	ids := make(KSUIDs, len(ss))

	for i, s := range ss {
		if err := ids[i].Parse(s); err != nil {
			return err
		}
	}

	*pids = ids
	return nil
}

func (ids KSUIDs) MarshalJSON() ([]byte, error) {
	return json.Marshal(ids.Strings())
}

func (ids *KSUIDs) UnmarshalJSON(data []byte) error {
	var ss []string
	if err := json.Unmarshal(data, &ss); err != nil {
		return ErrInvalidFormat
	}

	return ids.Parse(ss)
}

// sql.Scanner interface
func (ids *KSUIDs) Scan(src interface{}) error {
	if src == nil {
		*ids = nil
		return nil
	}

	switch v := src.(type) {
	case []string:
		return ids.Parse(v)

	case string:
		if len(v) < 2 || v[0] != '{' || v[len(v)-1] != '}' {
			return fmt.Errorf("invalid format")
		}
		parts := strings.Split(v[1:len(v)-1], ",")
		return ids.Parse(parts)

	default:
		return fmt.Errorf("invalid value of type %T", v)
	}
}

// sql/driver.Valuer interface
func (id KSUIDs) Value() (driver.Value, error) {
	return id.Strings(), nil
}
