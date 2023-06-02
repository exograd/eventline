package eventline

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/galdor/go-ejson"
	"github.com/stretchr/testify/require"
)

func TestFilterSerialization(t *testing.T) {
	require := require.New(t)

	re := regexp.MustCompile("^a")

	ptr := ejson.NewPointer("foo")

	f1 := &Filter{
		Path:      ptr,
		IsEqualTo: "bar",
		Matches:   "^a", // required for the test to pass
		MatchesRE: re,
	}

	data, err := json.Marshal(f1)
	require.NoError(err)

	var f2 Filter
	err = json.Unmarshal(data, &f2)
	require.NoError(err)

	v := ejson.NewValidator()
	f2.ValidateJSON(v)
	require.NoError(v.Error())

	require.Equal(*f1, f2)
}
