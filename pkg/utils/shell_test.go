package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShellEscape(t *testing.T) {
	assert := assert.New(t)

	assert.Equal(``, ShellEscape(``))
	assert.Equal(`foo`, ShellEscape(`foo`))
	assert.Equal(`été`, ShellEscape(`été`))
	assert.Equal(`foo\|bar\&baz`, ShellEscape(`foo|bar&baz`))
	assert.Equal(`\(foo\)\$`, ShellEscape(`(foo)$`))
	assert.Equal(`\\foo\\`, ShellEscape(`\foo\`))
	assert.Equal(`\"foo\ \'bar\'\"`, ShellEscape(`"foo 'bar'"`))
	assert.Equal("a\\\nb\\\tc d", ShellEscape("a\nb\tc d"))
}
