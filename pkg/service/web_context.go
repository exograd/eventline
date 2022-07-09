package service

import (
	"fmt"
	"math"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/utils"
)

// The set of information we inject into every rendered piece of web content.
// It is available in templates as .Context.
type WebContext struct {
	Product          string
	Version          string
	VersionHash      string
	PublicPage       bool
	LoggedIn         bool
	AccountSettings  *eventline.AccountSettings
	ProjectIdChecked bool // true if we have performed project id detection
	ProjectId        *eventline.Id
	ProjectName      string
}

func (ctx *WebContext) FormatDate(t time.Time) (s string) {
	format := eventline.DateFormatAbsolute

	if ctx.AccountSettings != nil && ctx.AccountSettings.DateFormat != "" {
		format = ctx.AccountSettings.DateFormat
	}

	return ctx.formatDate(t, format)
}

func (ctx *WebContext) FormatAltDate(t time.Time) (s string) {
	format := eventline.DateFormatRelative

	if ctx.AccountSettings != nil && ctx.AccountSettings.DateFormat != "" {
		if ctx.AccountSettings.DateFormat == eventline.DateFormatAbsolute {
			format = eventline.DateFormatRelative
		} else {
			format = eventline.DateFormatAbsolute
		}
	}

	return ctx.formatDate(t, format)
}

func (ctx *WebContext) formatDate(t time.Time, format eventline.DateFormat) (s string) {
	switch format {
	case eventline.DateFormatAbsolute:
		s = t.Format("2006-01-02 15:04:05Z07:00")

	case eventline.DateFormatRelative:
		s = utils.FormatRelativeDate(t, time.Now().UTC())

	default:
		utils.Panicf("unsupported date format %q", format)
	}

	return
}

func (ctx *WebContext) FormatDuration(d time.Duration) (s string) {
	us := d.Microseconds()

	switch {
	case us == 0:
		return fmt.Sprintf("%dns", d.Nanoseconds())

	case us < 1_000:
		return fmt.Sprintf("%dµs", us)

	case us < 1_000_000:
		return fmt.Sprintf("%.0fms", math.Ceil(float64(us)/1_000.0))

	case us < 60_000_000:
		return fmt.Sprintf("%.0fs", math.Ceil(float64(us)/1_000_000.0))

	case us < 3_600_000_000:
		m := us / 60_000_000
		s := math.Ceil(float64(us%60_000_000) / 1_000_000.0)
		if s > 0 {
			return fmt.Sprintf("%dm%.0fs", m, s)
		} else {
			return fmt.Sprintf("%dm", m)
		}

	default:
		h := us / 3_600_000_000
		m := math.Ceil(float64(us%3_600_000_000) / 60_000_000.0)
		if m > 0.0 {
			return fmt.Sprintf("%dh%.0fm", h, m)
		} else {
			return fmt.Sprintf("%dh", h)
		}
	}
}
