package eventline

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONFields(t *testing.T) {
	assert := assert.New(t)

	type testP struct {
		S string `json:"s"`
	}

	type testOA struct {
		S string `json:"s"`
	}

	type testOM struct {
		I int `json:"i"`
	}

	type testObj struct {
		B  bool              `json:"b"`
		I  int               `json:"i"`
		F  float64           `json:"f"`
		P  *testP            `json:"p"`
		A  []int             `json:"a,omitempty"`
		OA []testOA          `json:"oa,omitempty"`
		OM map[string]testOM `json:"om,omitempty"`
	}

	assertEqual := func(expectedFields map[string]string, obj testObj) {
		fields, err := JSONFields(obj)
		if assert.NoError(err) {
			assert.Equal(expectedFields, fields)
		}
	}

	assertEqual(map[string]string{
		"b":   "true",
		"i":   "42",
		"f":   "3.14",
		"p":   "null",
		"a/0": "1",
		"a/1": "2",
		"a/2": "3",
	}, testObj{
		B:  true,
		I:  42,
		F:  3.14,
		P:  nil,
		A:  []int{1, 2, 3},
		OA: nil,
		OM: nil,
	})

	assertEqual(map[string]string{
		"b":      "false",
		"i":      "-123",
		"f":      "0.01",
		"p/s":    "foo",
		"oa/0/s": "foo",
		"oa/1/s": "bar",
		"om/a/i": "1",
		"om/b/i": "2",
	}, testObj{
		B:  false,
		I:  -123,
		F:  0.01,
		P:  &testP{S: "foo"},
		OA: []testOA{testOA{S: "foo"}, testOA{S: "bar"}},
		OM: map[string]testOM{"a": testOM{I: 1}, "b": testOM{I: 2}},
	})
}
