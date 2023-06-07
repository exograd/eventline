package eventline

import (
	"fmt"
	"strings"

	"github.com/exograd/eventline/pkg/utils"
)

var TemplateFuncMap = map[string]interface{}{
	"add": func(a, b int) int {
		return a + b
	},

	"sub": func(a, b int) int {
		return a - b
	},

	"toSentence": utils.ToSentence,

	"join": strings.Join,

	"stringMember": func(s string, ss []string) bool {
		for _, s2 := range ss {
			if s == s2 {
				return true
			}
		}

		return false
	},

	"quoteString": func(s string) string {
		return fmt.Sprintf("%q", s)
	},
}
