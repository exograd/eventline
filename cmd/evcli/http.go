package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func NewHTTPClient() *http.Client {
	c := &http.Client{
		Timeout:   30 * time.Second,
		Transport: NewRoundTripper(http.DefaultTransport),
	}

	return c
}

type RoundTripper struct {
	http.RoundTripper
}

func NewRoundTripper(rt http.RoundTripper) *RoundTripper {
	return &RoundTripper{
		RoundTripper: rt,
	}
}

func (rt *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("User-Agent", app.UserAgent)

	start := time.Now()
	res, err := rt.RoundTripper.RoundTrip(req)
	d := time.Now().Sub(start)

	var statusString string
	if res == nil {
		statusString = "-"
	} else {
		statusString = strconv.Itoa(res.StatusCode)
	}

	p.Debug(2, "%s %s %s %s", req.Method, req.URL.String(), statusString,
		FormatRequestDuration(d))
	return res, err
}

func FormatRequestDuration(d time.Duration) string {
	s := d.Seconds()

	switch {
	case s < 0.001:
		return fmt.Sprintf("%dµs", d.Microseconds())

	case s < 1.0:
		return fmt.Sprintf("%dms", d.Milliseconds())

	default:
		return fmt.Sprintf("%.1fs", s)
	}
}
