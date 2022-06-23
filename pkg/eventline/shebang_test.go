package eventline

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShebangParse(t *testing.T) {
	assert := assert.New(t)

	var s Shebang

	assert.Error(s.Parse(""))
	assert.Error(s.Parse("#foo\n"))
	assert.Error(s.Parse("#!\n"))
	assert.Error(s.Parse("#!/bin/sh -e -u \n"))

	if assert.NoError(s.Parse("#!foo\n")) {
		assert.Equal("foo", s.Interpreter)
		assert.Equal("", s.Argument)
	}

	if assert.NoError(s.Parse("#! \t  foo\n")) {
		assert.Equal("foo", s.Interpreter)
		assert.Equal("", s.Argument)
	}

	if assert.NoError(s.Parse("#!/bin/sh -eu\n")) {
		assert.Equal("/bin/sh", s.Interpreter)
		assert.Equal("-eu", s.Argument)
	}

	if assert.NoError(s.Parse("#!/bin/sh\necho hello\n")) {
		assert.Equal("/bin/sh", s.Interpreter)
		assert.Equal("", s.Argument)
	}

	if assert.NoError(s.Parse("#!/bin/sh -eu\necho hello")) {
		assert.Equal("/bin/sh", s.Interpreter)
		assert.Equal("-eu", s.Argument)
	}
}
