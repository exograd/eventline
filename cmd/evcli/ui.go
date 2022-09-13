package main

import (
	"fmt"
	"strings"

	"github.com/peterh/liner"
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

	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	response, err := line.Prompt("Are you sure? (yes or no) ")
	if err != nil {
		line.Close()
		p.Fatal("cannot read response: %v", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))

	switch response {
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
