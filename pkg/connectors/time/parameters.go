package time

import (
	"time"

	"github.com/exograd/evgo/pkg/utils"
	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/djson"
)

type Parameters struct {
	Oneshot  *OneshotParameters `json:"oneshot,omitempty"`
	Periodic *int               `json:"periodic,omitempty"` // seconds
	Hourly   *HourlyParameters  `json:"hourly,omitempty"`
	Daily    *DailyParameters   `json:"daily,omitempty"`
	Weekly   *WeeklyParameters  `json:"weekly,omitempty"`
}

type OneshotParameters time.Time

type HourlyParameters struct {
	Minute int `json:"minute,omitempty"`
	Second int `json:"second,omitempty"`
}

type DailyParameters struct {
	Hour   int `json:"hour,omitempty"`
	Minute int `json:"minute,omitempty"`
	Second int `json:"second,omitempty"`
}

type WeeklyParameters struct {
	Day    WeekDay `json:"day"`
	Hour   int     `json:"hour,omitempty"`
	Minute int     `json:"minute,omitempty"`
	Second int     `json:"second,omitempty"`
}

func (p *Parameters) Check(c *check.Checker) {
	n := 0
	if p.Oneshot != nil {
		n += 1
	}
	if p.Periodic != nil {
		n += 1
	}
	if p.Hourly != nil {
		n += 1
	}
	if p.Daily != nil {
		n += 1
	}
	if p.Weekly != nil {
		n += 1
	}
	c.Check(djson.Pointer{}, n == 1, "invalid_value",
		"parameters must contain a single member")

	c.CheckOptionalObject("oneshot", p.Oneshot)
	if p.Periodic != nil {
		c.CheckIntMinMax("periodic", int(*p.Periodic), 30, 86400)
	}
	c.CheckOptionalObject("hourly", p.Hourly)
	c.CheckOptionalObject("daily", p.Daily)
	c.CheckOptionalObject("weekly", p.Weekly)
}

func (p *HourlyParameters) Check(c *check.Checker) {
	c.CheckIntMinMax("minute", p.Minute, 0, 59)
	c.CheckIntMinMax("second", p.Second, 0, 59)
}

func (p *DailyParameters) Check(c *check.Checker) {
	c.CheckIntMinMax("hour", p.Hour, 0, 23)
	c.CheckIntMinMax("minute", p.Minute, 0, 59)
	c.CheckIntMinMax("second", p.Second, 0, 59)
}

func (p *Parameters) FirstTick() (tick time.Time) {
	now := time.Now().UTC()

	switch {
	case p.Oneshot != nil:
		tick = time.Time(*p.Oneshot)

	case p.Periodic != nil:
		tick = now

	case p.Hourly != nil:
		tick = NextHour(now, p.Hourly.Minute, p.Hourly.Second)

	case p.Daily != nil:
		tick = NextDay(now, p.Daily.Hour, p.Daily.Minute, p.Daily.Second)

	case p.Weekly != nil:
		tick = NextWeekDay(now, p.Weekly.Day, p.Weekly.Hour, p.Weekly.Minute,
			p.Weekly.Second)

	default:
		utils.Panicf("unhandled tick parameters %#v", p)
	}

	return
}

func (p *Parameters) NextTick(expectedTick time.Time) (tick time.Time) {
	switch {
	case p.Oneshot != nil:
		tick = time.Time(*p.Oneshot)

	case p.Periodic != nil:
		tick = expectedTick.Add(time.Duration(*p.Periodic) * time.Second)

	case p.Hourly != nil:
		tick = NextHour(expectedTick, p.Hourly.Minute, p.Hourly.Second)

	case p.Daily != nil:
		tick = NextDay(expectedTick, p.Daily.Hour, p.Daily.Minute,
			p.Daily.Second)

	case p.Weekly != nil:
		tick = NextWeekDay(expectedTick, p.Weekly.Day, p.Weekly.Hour,
			p.Weekly.Minute, p.Weekly.Second)

	default:
		utils.Panicf("unhandled tick parameters %#v", p)
	}

	return
}
