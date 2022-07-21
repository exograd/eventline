package utils

import (
	"fmt"
	"math"
	"time"
)

func FormatRelativeDate(t time.Time, now time.Time) (s string) {
	seconds := now.Unix() - t.Unix()

	absSeconds := seconds
	if absSeconds < 0 {
		absSeconds = -seconds
	}

	switch {
	case absSeconds == 0:
		s = "now"
	case absSeconds == 1:
		s = "1 second"
	case absSeconds < 60:
		s = fmt.Sprintf("%d seconds", absSeconds)
	case absSeconds < 120:
		s = "1 minute"
	case absSeconds < 3600:
		s = fmt.Sprintf("%d minutes", absSeconds/60)
	case absSeconds < 7200:
		s = "1 hour"
	case absSeconds < 86400:
		s = fmt.Sprintf("%d hours", absSeconds/3600)
	case absSeconds < 172800:
		s = "1 day"
	default:
		s = fmt.Sprintf("%d days", absSeconds/86400)
	}

	if seconds > 0 {
		s += " ago"
	}

	return
}

func FormatDuration(d time.Duration) string {
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
