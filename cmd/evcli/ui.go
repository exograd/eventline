package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Color int

var (
	ColorBlack   = Color(0)
	ColorRed     = Color(1)
	ColorGreen   = Color(2)
	ColorYellow  = Color(3)
	ColorBlue    = Color(4)
	ColorMagenta = Color(5)
	ColorCyan    = Color(6)
	ColorWhite   = Color(7)
)

func Confirm(prompt string) bool {
	if skipConfirmations {
		return true
	}

	fmt.Printf("%s\n[yn] ", prompt)

	r := bufio.NewReader(os.Stdin)
	line, _, err := r.ReadLine()
	if err != nil {
		p.Fatal("cannot read stdin: %v", err)
	}

	response := strings.ToLower(strings.TrimSpace(string(line)))

	switch response {
	case "y":
		fallthrough
	case "yes":
		return true
	}

	return false
}

func Colorize(color Color, text string) string {
	if !colorOutput {
		return text
	}

	return fmt.Sprintf("\033[%dm%s\033[0m", 30+int(color), text)
}
