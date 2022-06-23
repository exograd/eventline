package utils

import (
	"fmt"
	"os"
)

func Abort(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
