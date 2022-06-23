package eventline

import (
	"regexp"

	"github.com/exograd/go-daemon/check"
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

func CheckName(c *check.Checker, token interface{}, name string) {
	c.CheckStringLengthMinMax(token, name, MinNameLength, MaxNameLength)

	if name != "" {
		c.CheckStringMatch2(token, name, NameRE, "invalid_format",
			"names must only contain lower case alphanumeric characters, "+
				"'-' or '_', and must start with an alphanumeric character")
	}

}

func CheckLabel(c *check.Checker, token string, label string) {
	c.CheckStringLengthMinMax(token, label, MinLabelLength, MaxLabelLength)
}

func CheckDescription(c *check.Checker, token string, label string) {
	c.CheckStringLengthMinMax(token, label,
		MinDescriptionLength, MaxDescriptionLength)
}
