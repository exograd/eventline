package eventline

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCursorParse(t *testing.T) {
	assert := assert.New(t)

	sorts := DefaultSorts

	assertEqual := func(ec Cursor, s string, as *AccountSettings) {
		t.Helper()

		uri, err := url.Parse(s)
		if assert.NoError(err) {
			var c Cursor

			assert.NoError(c.ParseQuery(uri.Query(), sorts, as), s)
			assert.Equal(ec, c, s)
		}
	}

	assertEqual(Cursor{}, "/", nil)

	assertEqual(Cursor{
		After: "1owDs3KXz5IM19fEK6H50T0hTiW",
	}, "/?after=MW93RHMzS1h6NUlNMTlmRUs2SDUwVDBoVGlX", nil)

	assertEqual(Cursor{
		Before: "1owDs3KXz5IM19fEK6H50T0hTiW",
	}, "/?before=MW93RHMzS1h6NUlNMTlmRUs2SDUwVDBoVGlX", nil)

	assertEqual(Cursor{
		Size: 42,
	}, "/?size=42", nil)

	assertEqual(Cursor{
		Size: 5,
	}, "/", &AccountSettings{PageSize: 5})

	assertEqual(Cursor{
		Size: 42,
	}, "/?size=42", &AccountSettings{PageSize: 5})

	assertEqual(Cursor{
		Order: OrderDesc,
	}, "/?order=desc", nil)

	assertEqual(Cursor{
		Order: OrderAsc,
	}, "/?order=asc", nil)

	assertEqual(Cursor{
		Before: "1owDs3KXz5IM19fEK6H50T0hTiW",
		Size:   42,
	}, "/?before=MW93RHMzS1h6NUlNMTlmRUs2SDUwVDBoVGlX&size=42", nil)

	assertEqual(Cursor{
		Sort: "id",
	}, "/?sort=id", nil)
}
