package utils

import (
	"strings"
	"unicode"
)

func Capitalize(s string) string {
	if s == "" {
		return s
	}

	runes := []rune(s)

	runes[0] = unicode.ToUpper(runes[0])

	return string(runes)
}

func ToSentence(s string) string {
	if s == "" {
		return s
	}

	runes := []rune(s)

	runes[0] = unicode.ToUpper(runes[0])

	last := runes[len(s)-1]
	if !strings.ContainsRune(".!?", last) {
		runes = append(runes, '.')
	}

	return string(runes)
}
