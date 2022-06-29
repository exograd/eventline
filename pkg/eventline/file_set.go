package eventline

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
)

type FileSet struct {
	Files map[string]*FileSetFile
}

type FileSetFile struct {
	Content []byte
	Mode    os.FileMode
}

func NewFileSet() *FileSet {
	return &FileSet{
		Files: make(map[string]*FileSetFile),
	}
}

func (s *FileSet) AddFile(filePath string, content []byte, mode os.FileMode) {
	s.Files[filePath] = &FileSetFile{
		Content: content,
		Mode:    mode,
	}
}

func (s *FileSet) Write(rootPath string) error {
	if err := os.RemoveAll(rootPath); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("cannot delete directory %q: %w", rootPath, err)
		}
	}

	if err := os.MkdirAll(rootPath, 0700); err != nil {
		return fmt.Errorf("cannot create directory %q: %w", rootPath, err)
	}

	for filePath, file := range s.Files {
		fullFilePath := path.Join(rootPath, filePath)

		dirPath := path.Dir(fullFilePath)
		if err := os.MkdirAll(dirPath, 0700); err != nil {
			return fmt.Errorf("cannot create directory %q: %w", dirPath, err)
		}

		err := os.WriteFile(fullFilePath, file.Content, file.Mode)
		if err != nil {
			os.RemoveAll(rootPath) // best effort
			return fmt.Errorf("cannot write %q: %w", fullFilePath, err)
		}
	}

	return nil
}
