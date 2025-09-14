package test

import (
	"go.n16f.net/uuid"
)

func RandomName(prefix, suffix string) string {
	var name string

	if prefix != "" {
		name += prefix + "-"
	}

	name += uuid.MustGenerate(uuid.V4).String()

	if suffix != "" {
		name += "-" + suffix
	}

	return name
}
