package eventline

import (
	"encoding/json"
	"regexp"

	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/djson"
)

type Filter struct {
	Path           djson.Pointer  `json:"path"`
	IsEqualTo      interface{}    `json:"is_equal_to,omitempty"`
	IsNotEqualTo   interface{}    `json:"is_not_equal_to,omitempty"`
	Matches        string         `json:"matches,omitempty"`
	MatchesRE      *regexp.Regexp `json:"-"`
	DoesNotMatch   string         `json:"does_not_match,omitempty"`
	DoesNotMatchRE *regexp.Regexp `json:"-"`
}

type Filters []*Filter

func (f *Filter) Check(c *check.Checker) {
	var err error

	f.MatchesRE, err = regexp.Compile(f.Matches)
	c.Check("matches", err == nil,
		"invalid_regexp", "invalid regexp: %v", err)

	f.DoesNotMatchRE, err = regexp.Compile(f.DoesNotMatch)
	c.Check("does_not_match", err == nil,
		"invalid_regexp", "invalid regexp: %v", err)
}

func (pf *Filter) MarshalJSON() ([]byte, error) {
	type Filter2 Filter

	f := Filter2(*pf)

	f.Matches = f.MatchesRE.String()
	f.DoesNotMatch = f.DoesNotMatchRE.String()

	return json.Marshal(f)
}

func (f *Filter) Match(obj interface{}) bool {
	v := f.Path.Find(obj)
	if v == nil {
		return false
	}

	if v2 := f.IsEqualTo; v2 != nil && !djson.Equal(v2, v) {
		return false
	}

	if v2 := f.IsNotEqualTo; v2 != nil && djson.Equal(v2, v) {
		return false
	}

	if re := f.MatchesRE; re != nil {
		s, ok := v.(string)
		if !ok || !re.MatchString(s) {
			return false
		}
	}

	if re := f.DoesNotMatchRE; re != nil {
		s, ok := v.(string)
		if !ok || re.MatchString(s) {
			return false
		}
	}

	return true
}

func (fs Filters) Match(obj interface{}) bool {
	for _, f := range fs {
		if !f.Match(obj) {
			return false
		}
	}

	return true
}
