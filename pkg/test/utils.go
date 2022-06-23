package test

import (
	"github.com/google/uuid"
)

func RandomName(prefix, suffix string) string {
	var name string

	if prefix != "" {
		name += prefix + "-"
	}

	// We use UUIDs and not KSUIDs because KSUIDs can contain upper case
	// letters, which are forbidden in names (see pkg/eventline/names.go).
	name += uuid.NewString()

	if suffix != "" {
		name += "-" + suffix
	}

	return name
}
