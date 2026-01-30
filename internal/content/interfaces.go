package content

import (
	"github.com/gofrs/uuid/v5"
	"io"
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
}
