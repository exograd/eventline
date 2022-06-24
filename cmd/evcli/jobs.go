package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/exograd/eventline/pkg/eventline"
)

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

func LoadJobFiles(dirPath string, ignoreSet *IgnoreSet) ([]*eventline.JobSpec, error) {
	filePaths, err := FindJobFiles(dirPath, ignoreSet)
	if err != nil {
		return nil, fmt.Errorf("cannot find files: %w", err)
	}

	var specs []*eventline.JobSpec

	for _, filePath := range filePaths {
		spec, err := LoadJobFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("cannot load %q: %w", filePath, err)
		}

		specs = append(specs, spec)
	}

	return specs, nil
}

func LoadSteps(spec *eventline.JobSpec, dirPath string) error {
	for i, step := range spec.Steps {
		switch {
		case step.Script != nil:
			if err := LoadScriptStep(step, dirPath); err != nil {
				return fmt.Errorf("cannot load script for "+
					"step %d: %w", i+1, err)
			}

		case step.Bundle != nil:
			if err := LoadBundleStep(step, dirPath); err != nil {
				return fmt.Errorf("cannot load bundle for "+
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

func LoadBundleStep(step *eventline.Step, dirPath string) error {
	var ignoreSet IgnoreSet
	for _, pattern := range step.Bundle.Exclude {
		if err := ignoreSet.AddPattern(pattern); err != nil {
			return fmt.Errorf("invalid exclusion pattern %q: %w", pattern, err)
		}
	}

	fileDirPath := path.Join(dirPath, step.Bundle.Path)

	files, err := FindBundleFiles(fileDirPath, &ignoreSet)
	if err != nil {
		return fmt.Errorf("cannot find files: %w", err)
	}

	for filePath, info := range files {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("cannot read %q: %w", filePath, err)
		}

		bundleFile := eventline.StepBundleFile{
			Name:    filePath[len(fileDirPath)+1:], // relative name
			Mode:    info.Mode(),
			Content: string(data),
		}

		step.Bundle.Files = append(step.Bundle.Files, bundleFile)
	}

	return nil
}

func FindJobFiles(dirPath string, ignoreSet *IgnoreSet) ([]string, error) {
	return findJobFiles(dirPath, dirPath, ignoreSet)
}

func findJobFiles(dirPath, curDirPath string, ignoreSet *IgnoreSet) ([]string, error) {
	var filePaths []string

	entries, err := os.ReadDir(curDirPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read directory %s: %w", curDirPath, err)
	}

	for _, e := range entries {
		fileName := e.Name()
		if fileName[0] == '.' {
			continue
		}

		if e.IsDir() {
			subDirPath := path.Join(curDirPath, fileName)
			filePaths2, err := findJobFiles(dirPath, subDirPath,
				ignoreSet)
			if err != nil {
				return nil, err
			}

			filePaths = append(filePaths, filePaths2...)
		} else {
			ext := strings.ToLower(filepath.Ext(fileName))
			if ext != ".yaml" && ext != ".yml" {
				continue
			}

			filePath := path.Join(curDirPath, fileName)

			relPath := filePath[len(dirPath):]
			if match, why := ignoreSet.Match(relPath); match {
				p.Debug(2, "ignoring job file %s (%s)", filePath, why)
				continue
			}

			filePaths = append(filePaths, filePath)
		}
	}

	return filePaths, nil
}

func FindBundleFiles(dirPath string, ignoreSet *IgnoreSet) (map[string]os.FileInfo, error) {
	return findBundleFiles(dirPath, dirPath, ignoreSet)
}

func findBundleFiles(dirPath, curDirPath string, ignoreSet *IgnoreSet) (map[string]os.FileInfo, error) {
	files := make(map[string]os.FileInfo)

	entries, err := os.ReadDir(curDirPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read directory %s: %w", curDirPath, err)
	}

	for _, e := range entries {
		fileName := e.Name()
		if fileName[0] == '.' {
			continue
		}

		if e.IsDir() {
			subDirPath := path.Join(curDirPath, fileName)
			files2, err := findBundleFiles(dirPath, subDirPath, ignoreSet)
			if err != nil {
				return nil, err
			}

			for filePath, file := range files2 {
				files[filePath] = file
			}
		} else {
			filePath := path.Join(curDirPath, fileName)

			relPath := filePath[len(dirPath):]
			if match, why := ignoreSet.Match(relPath); match {
				p.Debug(2, "ignoring bundle file %s (%s)", filePath, why)
				continue
			}

			info, err := e.Info()
			if err != nil {
				return nil, fmt.Errorf("cannot stat %q: %w", filePath, err)
			}

			files[filePath] = info
		}
	}

	return files, nil
}
