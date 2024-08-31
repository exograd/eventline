package time

import (
	"encoding/json"
	"time"

	"go.n16f.net/ejson"
	"go.n16f.net/program"
)

type Parameters struct {
	OneshotString string            `json:"oneshot,omitempty"`
	Oneshot       *time.Time        `json:"-"`
	Periodic      *int              `json:"periodic,omitempty"` // seconds
	Hourly        *HourlyParameters `json:"hourly,omitempty"`
	Daily         *DailyParameters  `json:"daily,omitempty"`
	Weekly        *WeeklyParameters `json:"weekly,omitempty"`
}

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

func (p *Parameters) MarshalJSON() ([]byte, error) {
	type Parameters2 Parameters
	p2 := Parameters2(*p)

	if p2.Oneshot != nil {
		p2.OneshotString = p2.Oneshot.Format(time.RFC3339)
	}

	return json.Marshal(p2)
}

func (p *Parameters) ValidateJSON(v *ejson.Validator) {
	if p.OneshotString != "" {
		t, err := time.Parse(time.RFC3339, p.OneshotString)
		if v.Check("oneshot", err == nil, "invalid_datetime",
			"invalid datetime string") {
			p.Oneshot = &t
		}
	}

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
	v.Check(ejson.Pointer{}, n == 1, "invalid_value",
		"parameters must contain a single member")

	if p.Periodic != nil {
		v.CheckIntMinMax("periodic", int(*p.Periodic), 30, 86400)
	}

	v.CheckOptionalObject("hourly", p.Hourly)
	v.CheckOptionalObject("daily", p.Daily)
	v.CheckOptionalObject("weekly", p.Weekly)
}

func (p *HourlyParameters) ValidateJSON(v *ejson.Validator) {
	v.CheckIntMinMax("minute", p.Minute, 0, 59)
	v.CheckIntMinMax("second", p.Second, 0, 59)
}

func (p *DailyParameters) ValidateJSON(v *ejson.Validator) {
	v.CheckIntMinMax("hour", p.Hour, 0, 23)
	v.CheckIntMinMax("minute", p.Minute, 0, 59)
	v.CheckIntMinMax("second", p.Second, 0, 59)
}

func (p *WeeklyParameters) ValidateJSON(v *ejson.Validator) {
	v.CheckStringValue("day", p.Day, WeekDayValues)
	v.CheckIntMinMax("hour", p.Hour, 0, 23)
	v.CheckIntMinMax("minute", p.Minute, 0, 59)
	v.CheckIntMinMax("second", p.Second, 0, 59)
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
		program.Panicf("unhandled tick parameters %#v", p)
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
		program.Panicf("unhandled tick parameters %#v", p)
	}

	return
}
