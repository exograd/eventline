package eventline

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/djson"
	"github.com/stretchr/testify/require"
)

func TestFilterSerialization(t *testing.T) {
	require := require.New(t)

	re := regexp.MustCompile("^a")

	ptr := djson.NewPointer("foo")

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

	c := check.NewChecker()
	f2.Check(c)
	require.NoError(c.Error())

	require.Equal(*f1, f2)
}
