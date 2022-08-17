package local

import (
	"path"

	"github.com/exograd/go-daemon/check"
)

type RunnerCfg struct {
	RootDirectory string `json:"root_directory"`
}

func (cfg *RunnerCfg) Check(c *check.Checker) {
	if c.CheckStringNotEmpty("root_directory", cfg.RootDirectory) {
		c.Check("root_directory", path.IsAbs(cfg.RootDirectory),
			"invalid_relative_path", "path must be absolute")
	}
}
