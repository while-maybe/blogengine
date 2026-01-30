package storage

import "io"

type Provider interface {
	Open(path string) (io.ReadCloser, error)
	Exists(path string) bool
}
