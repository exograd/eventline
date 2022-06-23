package time

import (
	"time"

	"github.com/exograd/evgo/pkg/utils"
)

type WeekDay string

const (
	WeekDayMonday    WeekDay = "monday"
	WeekDayTuesday   WeekDay = "tuesday"
	WeekDayWednesday WeekDay = "wednesday"
	WeekDayThursday  WeekDay = "thursday"
	WeekDayFriday    WeekDay = "friday"
	WeekDaySaturday  WeekDay = "saturday"
	WeekDaySunday    WeekDay = "sunday"
)

var WeekDayValues = []WeekDay{
	WeekDayMonday,
	WeekDayTuesday,
	WeekDayWednesday,
	WeekDayThursday,
	WeekDayFriday,
	WeekDaySaturday,
	WeekDaySunday,
}

func (wd WeekDay) Number() (n int) {
	// Follow the convention of the Go standard time module

	switch wd {
	case WeekDayMonday:
		n = 1
	case WeekDayTuesday:
		n = 2
	case WeekDayWednesday:
		n = 3
	case WeekDayThursday:
		n = 4
	case WeekDayFriday:
		n = 5
	case WeekDaySaturday:
		n = 6
	case WeekDaySunday:
		n = 0

	default:
		utils.Panicf("unhandled week day %q", string(wd))
	}

	return
}

func NextHour(now time.Time, m, s int) time.Time {
	var h int

	nm := now.Minute()
	ns := now.Second()

	if nm < m || nm == m && ns < s {
		h = now.Hour()
	} else {
		h = now.Hour() + 1
	}

	return time.Date(now.Year(), now.Month(), now.Day(), h, m, s, 0, time.UTC)
}

func NextDay(now time.Time, h, m, s int) time.Time {
	var d int

	nh := now.Hour()
	nm := now.Minute()
	ns := now.Second()

	if nh < h || nh == h && (nm < m || nm == m && ns < s) {
		d = now.Day()
	} else {
		d = now.Day() + 1
	}

	return time.Date(now.Year(), now.Month(), d, h, m, s, 0, time.UTC)
}

func NextWeekDay(now time.Time, day WeekDay, h, m, s int) time.Time {
	nwd := int(now.Weekday())
	wd := day.Number()

	nh := now.Hour()
	nm := now.Minute()
	ns := now.Second()

	timeBefore := nh < h || nh == h && (nm < m || nm == m && ns < s)

	var d int

	if nwd == wd {
		d = now.Day()
		if !timeBefore {
			d += 7
		}
	} else if nwd < wd {
		d = now.Day() + wd - nwd
	} else if nwd > wd {
		d = now.Day() + 7 - nwd + wd
	}

	return time.Date(now.Year(), now.Month(), d, h, m, s, 0, time.UTC)
}
