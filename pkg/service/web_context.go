package service

import (
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/utils"
	"go.n16f.net/program"
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
		program.Panicf("unsupported date format %q", format)
	}

	return
}

func (ctx *WebContext) FormatDuration(d time.Duration) (s string) {
	return utils.FormatDuration(d)
}
