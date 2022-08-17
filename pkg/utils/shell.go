package utils

import (
	"strings"
)

// According to POSIX, the following character must be quoted:
//
// |  &  ;  <  >  (  )  $  `  \  "  '  <space>  <tab>  <newline>
var shellEscapeReplacer = strings.NewReplacer(
	`|`, `\|`,
	`&`, `\&`,
	`;`, `\;`,
	`<`, `\<`,
	`>`, `\>`,
	`(`, `\(`,
	`)`, `\)`,
	`$`, `\$`,
	"`", "\\`",
	`\`, `\\`,
	`"`, `\"`,
	`'`, `\'`,
	` `, `\ `,
	"\t", "\\\t",
	"\n", "\\\n",
)

func ShellEscape(s string) string {
	return shellEscapeReplacer.Replace(s)
}
