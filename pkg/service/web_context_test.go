package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWebContextFormatDuration(t *testing.T) {
	assert := assert.New(t)

	ctx := WebContext{}

	duration := func(seconds float64) time.Duration {
		return time.Duration(int64(seconds * 1e9))
	}

	assert.Equal("249ns", ctx.FormatDuration(duration(0.000000249)))
	assert.Equal("3µs", ctx.FormatDuration(duration(0.00000321)))
	assert.Equal("62µs", ctx.FormatDuration(duration(0.000062)))
	assert.Equal("2ms", ctx.FormatDuration(duration(0.0018)))
	assert.Equal("15ms", ctx.FormatDuration(duration(0.015)))
	assert.Equal("100ms", ctx.FormatDuration(duration(0.1)))
	assert.Equal("500ms", ctx.FormatDuration(duration(0.5)))
	assert.Equal("1s", ctx.FormatDuration(duration(1.0)))
	assert.Equal("2s", ctx.FormatDuration(duration(1.47)))
	assert.Equal("2s", ctx.FormatDuration(duration(1.87)))
	assert.Equal("51s", ctx.FormatDuration(duration(51.0)))
	assert.Equal("1m20s", ctx.FormatDuration(duration(80)))
	assert.Equal("15m30s", ctx.FormatDuration(duration(930)))
	assert.Equal("1h", ctx.FormatDuration(duration(3600)))
	assert.Equal("1h1m", ctx.FormatDuration(duration(3605)))
	assert.Equal("1h10m", ctx.FormatDuration(duration(4200)))
	assert.Equal("1h11m", ctx.FormatDuration(duration(4210)))
	assert.Equal("1h11m", ctx.FormatDuration(duration(4258)))
	assert.Equal("2h30m", ctx.FormatDuration(duration(9000)))
}
