package main

import (
	"fmt"

	"go.n16f.net/program"
)

func cmdVersion(p *program.Program) {
	fmt.Printf("evcli %s\n", buildId)
}
