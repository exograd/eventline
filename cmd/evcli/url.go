package main

import (
	"bytes"
	"net/url"
)

// The url package has an extremely confusing interface. One could believe
// setting RawPath is enough for it to be used during encoding, but this is
// not the case. Instead, *both* must be set, and (*url.URL).EscapedPath will
// check that RawPath is a valid encoding of RawPath. If this is not the case,
// RawPath will be ignored.
//
// In most cases, the problem is not apparent. But if the path contains
// segments with slash or space characters, it is almost guaranteed to misuse
// Path and RawPath and double-encode these characters.
//
// The problem was signaled years ago on
// https://github.com/golang/go/issues/17340 but was of course ignored and
// buried.
//
// As usual with standard library issues, the only thing we can do is add
// utils to work around it.

func NewURL(pathSegments ...string) *url.URL {
	var buf bytes.Buffer
	buf.WriteByte('/')
	for i, s := range pathSegments {
		if i > 0 {
			buf.WriteByte('/')
		}
		buf.WriteString(url.PathEscape(s))
	}
	rawPath := buf.String()

	path, _ := url.PathUnescape(rawPath)

	return &url.URL{
		Path:    path,
		RawPath: rawPath,
	}
}
