package eventline

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	shebangRE      = regexp.MustCompile(`^#!\s*(\S+)\s*(.+)?`)
	shebangSpaceRE = regexp.MustCompile(`\s`)
)

type Shebang struct {
	Interpreter string
	Argument    string
}

func (s *Shebang) Parse(data string) error {
	parts := strings.SplitN(data, "\n", 2)
	line := strings.TrimRight(parts[0], " \n\r\t")

	matches := shebangRE.FindAllStringSubmatch(line, -1)
	if len(matches) < 1 {
		return fmt.Errorf("invalid format")
	}

	groups := matches[0][1:]

	s.Interpreter = groups[0]

	if len(groups) >= 2 {
		arg := groups[1]

		if shebangSpaceRE.MatchString(arg) {
			return fmt.Errorf("invalid space character in argument")
		}

		s.Argument = arg
	} else {
		s.Argument = ""
	}

	return nil
}

func StartsWithShebang(data string) bool {
	return shebangRE.MatchString(data)
}
