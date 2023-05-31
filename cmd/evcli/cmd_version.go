package main

import (
	"fmt"

	"github.com/galdor/go-program"
)

func cmdVersion(p *program.Program) {
	fmt.Printf("evcli %s\n", buildId)
}
