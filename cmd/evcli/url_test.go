package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewURL(t *testing.T) {
	assert := assert.New(t)

	assert.Equal("/",
		NewURL().String())

	assert.Equal("/a",
		NewURL("a").String())

	assert.Equal("/a/bcd/ef",
		NewURL("a", "bcd", "ef").String())

	assert.Equal("/a/b%20c/e%2Ff",
		NewURL("a", "b c", "e/f").String())
}
