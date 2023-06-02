package eventline

import (
	"regexp"

	"github.com/galdor/go-ejson"
)

const (
	MinNameLength = 1
	MaxNameLength = 100

	MinLabelLength = 1
	MaxLabelLength = 100

	MinDescriptionLength = 1
	MaxDescriptionLength = 500
)

var (
	NameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9\-_]*$`)
)

func CheckName(v *ejson.Validator, token interface{}, name string) {
	v.CheckStringLengthMinMax(token, name, MinNameLength, MaxNameLength)

	if name != "" {
		v.CheckStringMatch2(token, name, NameRE, "invalid_format",
			"names must only contain lower case alphanumeric characters, "+
				"'-' or '_', and must start with an alphanumeric character")
	}

}

func CheckLabel(v *ejson.Validator, token string, label string) {
	v.CheckStringLengthMinMax(token, label, MinLabelLength, MaxLabelLength)
}

func CheckDescription(v *ejson.Validator, token string, label string) {
	v.CheckStringLengthMinMax(token, label,
		MinDescriptionLength, MaxDescriptionLength)
}
