package utils

import (
	"fmt"
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
