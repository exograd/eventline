package main

import (
	"fmt"

	"github.com/exograd/go-program"
)

func cmdVersion(p *program.Program) {
	fmt.Printf("evcli %s\n", buildId)
}
