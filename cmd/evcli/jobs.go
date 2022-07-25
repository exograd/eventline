package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/utils"
)

func FindJobFiles(filesOrDirPaths []string, recursive bool) ([]string, error) {
	var filePaths []string

	var fn func(string, int) error
	fn = func(currentPath string, depth int) error {
		info, err := os.Stat(currentPath)
		if err != nil {
			return fmt.Errorf("cannot stat %q: %w", currentPath, err)
		}

		if info.IsDir() {
			if recursive || depth == 0 {
				entries, err := ioutil.ReadDir(currentPath)
				if err != nil {
					return fmt.Errorf("cannot list directory %q: %w",
						currentPath, err)
				}

				for _, entry := range entries {
					fullPath := path.Join(currentPath, entry.Name())
					if err := fn(fullPath, depth+1); err != nil {
						return err
					}
				}
			}
		} else {
			ext := path.Ext(currentPath)
			if ext == ".yml" || ext == ".yaml" {
				filePaths = append(filePaths, currentPath)
			}
		}

		return nil
	}

	for _, fileOrDirPath := range filesOrDirPaths {
		if err := fn(fileOrDirPath, 0); err != nil {
			return nil, err
		}
	}

	return filePaths, nil
}

func LoadJobFile(filePath string) (*eventline.JobSpec, error) {
	p.Debug(1, "loading job file %s", filePath)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read %q: %w", filePath, err)
	}

	var spec eventline.JobSpec
	if err := spec.ParseYAML(data); err != nil {
		return nil, fmt.Errorf("cannot decode data: %w", err)
	}

	dirPath := filepath.Dir(filePath)
	if err := LoadSteps(&spec, dirPath); err != nil {
		return nil, err
	}

	return &spec, nil
}

func LoadSteps(spec *eventline.JobSpec, dirPath string) error {
	for i, step := range spec.Steps {
		switch {
		case step.Script != nil:
			if err := LoadScriptStep(step, dirPath); err != nil {
				return fmt.Errorf("cannot load script for "+
					"step %d: %w", i+1, err)
			}
		}
	}

	return nil
}

func LoadScriptStep(step *eventline.Step, dirPath string) error {
	filePath := path.Join(dirPath, step.Script.Path)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("cannot read %q: %w", filePath, err)
	}

	step.Script.Content = string(data)

	return nil
}
func ExportJob(spec *eventline.JobSpec, dirPath string) (string, error) {
	for _, step := range spec.Steps {
		if script := step.Script; script != nil {
			scriptPath := path.Join(dirPath, script.Path)

			err := os.WriteFile(scriptPath, []byte(script.Content), 0700)
			if err != nil {
				return "", fmt.Errorf("cannot write %q: %w", scriptPath, err)
			}

			script.Content = ""
		}
	}

	specData, err := utils.YAMLEncode(spec)
	if err != nil {
		return "", fmt.Errorf("cannot encode job specification: %w", err)
	}

	filePath := path.Join(dirPath, spec.Name+".yaml")
	if err := os.WriteFile(filePath, specData, 0600); err != nil {
		return "", fmt.Errorf("cannot write %q: %w", filePath, err)
	}

	return filePath, nil
}
