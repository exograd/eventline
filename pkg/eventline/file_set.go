package eventline

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"time"
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

func (s *FileSet) AddPrefix(prefix string) {
	files := make(map[string]*FileSetFile, len(s.Files))

	for filePath, file := range s.Files {
		files[path.Join(prefix, filePath)] = file
	}

	s.Files = files
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

func (s *FileSet) TarArchive(buf *bytes.Buffer) error {
	now := time.Now().UTC()

	w := tar.NewWriter(buf)

	for filePath, file := range s.Files {
		typeFlag := tar.TypeReg

		header := tar.Header{
			Typeflag: byte(typeFlag),
			Name:     filePath,
			Size:     int64(len(file.Content)),
			Mode:     int64(file.Mode),
			ModTime:  now,
		}

		if err := w.WriteHeader(&header); err != nil {
			return fmt.Errorf("cannot write header: %w", err)
		}

		if _, err := w.Write(file.Content); err != nil {
			return fmt.Errorf("cannot write content: %w", err)
		}
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("cannot close archive: %w", err)
	}

	return nil
}
