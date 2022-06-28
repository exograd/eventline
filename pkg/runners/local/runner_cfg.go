package local

import "github.com/exograd/go-daemon/check"

type RunnerCfg struct {
	RootDirectory string `json:"root_directory"`
}

func (cfg *RunnerCfg) Check(c *check.Checker) {
	c.CheckStringNotEmpty("root_directory", cfg.RootDirectory)
}
