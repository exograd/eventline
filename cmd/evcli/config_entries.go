package main

import (
	"fmt"
	"strings"
)

var ConfigEntries map[string]ConfigEntry

type ConfigEntry struct {
	Name string
	Get  func(*Config) string
	Set  func(*Config, string) error
}

func init() {
	entries := []ConfigEntry{
		ConfigEntry{
			Name: "api.endpoint",
			Get:  func(c *Config) string { return c.API.Endpoint },
			Set: func(c *Config, s string) error {
				return setString(s, &c.API.Endpoint)
			},
		},
		ConfigEntry{
			Name: "api.key",
			Get:  func(c *Config) string { return c.API.Key },
			Set: func(c *Config, s string) error {
				return setString(s, &c.API.Key)
			},
		},
		ConfigEntry{
			Name: "interface.color",
			Get:  func(c *Config) string { return fmtBool(c.Interface.Color) },
			Set: func(c *Config, s string) error {
				return setBool(s, &c.Interface.Color)
			},
		},
		ConfigEntry{
			Name: "misc.disable_update_check",
			Get: func(c *Config) string {
				return fmtBool(c.Misc.DisableUpdateCheck)
			},
			Set: func(c *Config, s string) error {
				return setBool(s, &c.Misc.DisableUpdateCheck)
			},
		},
	}

	ConfigEntries = make(map[string]ConfigEntry)
	for _, e := range entries {
		ConfigEntries[e.Name] = e
	}
}

func fmtBool(b bool) string {
	if b {
		return "true"
	} else {
		return "false"
	}
}

func setString(s string, ps *string) error {
	*ps = s
	return nil
}

func setBool(s string, pb *bool) error {
	s = strings.ToLower(strings.TrimSpace(s))

	if s == "true" {
		*pb = true
	} else if s == "false" {
		*pb = false
	} else {
		return fmt.Errorf("%q is not a valid boolean", s)
	}

	return nil
}
