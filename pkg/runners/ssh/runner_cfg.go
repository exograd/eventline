package ssh

import (
	"path"

	"github.com/galdor/go-ejson"
)

type RunnerCfg struct {
	RootDirectory string `json:"root_directory"`
}

func (cfg *RunnerCfg) ValidateJSON(v *ejson.Validator) {
	if v.CheckStringNotEmpty("root_directory", cfg.RootDirectory) {
		v.Check("root_directory", path.IsAbs(cfg.RootDirectory),
			"invalid_relative_path", "path must be absolute")
	}
}
