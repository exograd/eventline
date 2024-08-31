package time

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.n16f.net/program"
)

func TestNextHour(t *testing.T) {
	assert := assert.New(t)

	assert.Equal(testTime("2020-05-01 10:20:30Z"),
		NextHour(testTime("2020-05-01 10:00:00Z"), 20, 30))

	assert.Equal(testTime("2020-05-01 11:20:30Z"),
		NextHour(testTime("2020-05-01 10:20:30Z"), 20, 30))

	assert.Equal(testTime("2020-05-02 00:20:30Z"),
		NextHour(testTime("2020-05-01 23:50:00Z"), 20, 30))

	assert.Equal(testTime("2021-01-01 00:20:30Z"),
		NextHour(testTime("2020-12-31 23:20:31Z"), 20, 30))

	assert.Equal(testTime("2020-05-01 11:00:00Z"),
		NextHour(testTime("2020-05-01 10:00:00Z"), 0, 0))

	assert.Equal(testTime("2020-05-01 11:00:00Z"),
		NextHour(testTime("2020-05-01 10:20:30Z"), 0, 0))

	assert.Equal(testTime("2020-05-02 00:00:00Z"),
		NextHour(testTime("2020-05-01 23:20:30Z"), 0, 0))
}

func TestNextDay(t *testing.T) {
	assert := assert.New(t)

	assert.Equal(testTime("2020-05-01 10:20:30Z"),
		NextDay(testTime("2020-05-01 10:00:00Z"), 10, 20, 30))

	assert.Equal(testTime("2020-05-02 10:20:30Z"),
		NextDay(testTime("2020-05-01 11:00:00Z"), 10, 20, 30))

	assert.Equal(testTime("2020-05-02 05:00:30Z"),
		NextDay(testTime("2020-05-01 10:00:00Z"), 5, 0, 30))

	assert.Equal(testTime("2021-01-01 08:30:00Z"),
		NextDay(testTime("2020-12-31 12:00:00Z"), 8, 30, 0))

	assert.Equal(testTime("2020-05-02 00:00:00Z"),
		NextDay(testTime("2020-05-01 00:00:00Z"), 0, 0, 0))
}

func TestNextWeekDay(t *testing.T) {
	assert := assert.New(t)

	// Same day same time
	assert.Equal(testTime("2021-08-09 10:20:30Z"),
		NextWeekDay(testTime("2021-08-02 10:20:30Z"),
			WeekDayMonday, 10, 20, 30))

	// Same day before time
	assert.Equal(testTime("2021-08-02 10:20:30Z"),
		NextWeekDay(testTime("2021-08-02 10:00:00Z"),
			WeekDayMonday, 10, 20, 30))

	// Same day after time
	assert.Equal(testTime("2021-08-09 10:20:30Z"),
		NextWeekDay(testTime("2021-08-02 11:00:00Z"),
			WeekDayMonday, 10, 20, 30))

	// Week day after current one
	assert.Equal(testTime("2021-08-03 10:20:30Z"),
		NextWeekDay(testTime("2021-08-02 10:00:00Z"),
			WeekDayTuesday, 10, 20, 30))

	assert.Equal(testTime("2021-08-03 10:20:30Z"),
		NextWeekDay(testTime("2021-08-02 11:00:00Z"),
			WeekDayTuesday, 10, 20, 30))

	// Week day before current one
	assert.Equal(testTime("2021-08-07 10:20:30Z"),
		NextWeekDay(testTime("2021-08-01 10:00:00Z"),
			WeekDaySaturday, 10, 20, 30))

	assert.Equal(testTime("2021-08-07 10:20:30Z"),
		NextWeekDay(testTime("2021-08-01 11:00:00Z"),
			WeekDaySaturday, 10, 20, 30))

	// Next week day with year change
	assert.Equal(testTime("2021-01-01 10:20:30Z"),
		NextWeekDay(testTime("2020-12-30 10:00:00Z"),
			WeekDayFriday, 10, 20, 30))
}

func testTime(s string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05Z07:00", s)
	if err != nil {
		program.Panicf("invalid time %q: %v", s, err)
	}

	return t
}
