package storage

import (
	"context"
	"io"
)

type Provider interface {
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Save(ctx context.Context, key string, body io.ReadSeeker) error
	Exists(ctx context.Context, key string) bool
}
