package ksuid

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestKSUIDParse(t *testing.T) {
	assert := assert.New(t)

	var id KSUID
	var err error

	err = id.Parse("1l12i5euax5i7oGDn5DFULPYdCM")
	if assert.NoError(err) {
		assert.Equal(KSUID{12, 82, 194, 192, 81, 235, 30, 45, 167, 101,
			37, 49, 142, 237, 93, 150, 108, 158, 91, 122}, id)
	}

	err = id.Parse("")
	assert.ErrorIs(err, ErrInvalidFormat)

	err = id.Parse("1l12i5euax5i7oGDn5DFULPYdCM2")
	assert.ErrorIs(err, ErrInvalidFormat)

	err = id.Parse("1l12i5euax5i7oGDn5DFULPYdC=")
	assert.ErrorIs(err, ErrInvalidFormat)
}

func TestKSUIDString(t *testing.T) {
	assert := assert.New(t)

	assert.Equal("1l12i5euax5i7oGDn5DFULPYdCM",
		KSUID{12, 82, 194, 192, 81, 235, 30, 45, 167, 101,
			37, 49, 142, 237, 93, 150, 108, 158, 91, 122}.String())
}

func TestKSUIDTime(t *testing.T) {
	assert := assert.New(t)

	date := time.Date(2020, 2, 1, 10, 20, 30, 0, time.UTC)
	id := GenerateWithTime(date)

	assert.Equal(date.Unix(), int64(Epoch+id.Timestamp()))
	assert.Equal(date, id.Time())
}

func TestKSUIDIsZero(t *testing.T) {
	assert := assert.New(t)

	assert.True(Zero.IsZero())
	assert.False(Generate().IsZero())
}
