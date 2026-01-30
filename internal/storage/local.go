package storage

import (
	"io"
	"os"
	"path/filepath"
)

type LocalStore struct {
	basePath string
}

func NewLocalStorage(basePath string) *LocalStore {
	return &LocalStore{basePath: basePath}
}

func (l *LocalStore) Open(path string) (io.ReadCloser, error) {
	return os.OpenInRoot(l.basePath, path)
}

// Exists takes a path and returns true if the file exists and can be opened
func (l *LocalStore) Exists(path string) bool {
	path = filepath.Clean(path)

	f, err := os.OpenInRoot(l.basePath, path)
	if err != nil {
		return false
	}

	defer f.Close() // overkill to consider errors if only checking existence
	return true
}
