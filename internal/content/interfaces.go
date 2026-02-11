package content

import (
	"context"
	"io"

	"github.com/gofrs/uuid/v5"
)

// PostService defines read-access to blog posts
type PostService interface {
	GetAll() []*Post
	Get(id uint32) (*Post, error)
}

// MediaService defines access to binary assets
type MediaService interface {
	Retrieve(id uuid.UUID) (io.ReadCloser, error)
	Obfuscate(path string) (uuid.UUID, error)
	GetRelativePath(id uuid.UUID) (string, error)
}

// ImageProcessorService defines source image file processing
type ImageProcessorService interface {
	Enqueue(ctx context.Context, job ImageJob) error
}
