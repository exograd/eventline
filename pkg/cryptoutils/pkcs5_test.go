package cryptoutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPadPKCS5(t *testing.T) {
	assert := assert.New(t)

	assert.Equal([]byte("\x04\x04\x04\x04"),
		PadPKCS5([]byte(""), 4))
	assert.Equal([]byte("a\x03\x03\x03"),
		PadPKCS5([]byte("a"), 4))
	assert.Equal([]byte("ab\x02\x02"),
		PadPKCS5([]byte("ab"), 4))
	assert.Equal([]byte("abc\x01"),
		PadPKCS5([]byte("abc"), 4))
	assert.Equal([]byte("abcd\x04\x04\x04\x04"),
		PadPKCS5([]byte("abcd"), 4))
	assert.Equal([]byte("abcde\x03\x03\x03"),
		PadPKCS5([]byte("abcde"), 4))
	assert.Equal([]byte("abcdefgh\x04\x04\x04\x04"),
		PadPKCS5([]byte("abcdefgh"), 4))
}

func TestUnpadPKCS5(t *testing.T) {
	assert := assert.New(t)

	assertEqual := func(expected, data []byte) {
		t.Helper()

		data2, err := UnpadPKCS5(data, 4)
		if assert.NoError(err) {
			assert.Equal(expected, data2)
		}
	}

	assertEqual([]byte(""),
		[]byte("\x04\x04\x04\x04"))
	assertEqual([]byte("a"),
		[]byte("a\x03\x03\x03"))
	assertEqual([]byte("ab"),
		[]byte("ab\x02\x02"))
	assertEqual([]byte("abc"),
		[]byte("abc\x01"))
	assertEqual([]byte("abcd"),
		[]byte("abcd\x04\x04\x04\x04"))
	assertEqual([]byte("abcde"),
		[]byte("abcde\x03\x03\x03"))
	assertEqual([]byte("abcdefgh"),
		[]byte("abcdefgh\x04\x04\x04\x04"))
}
