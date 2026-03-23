package seeder

import (
	"blogengine/internal/storage"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/frontmatter"
	"gopkg.in/yaml.v3"
)

var (
	ErrReadToBuffer         = errors.New("could not read to buffer")
	ErrUnmarshallToManifest = errors.New("could not unmarshall data to blog manifest")
	ErrNoBlogTitle          = errors.New("blog manifest missing title")
	ErrNoBlogOwner          = errors.New("blog manifest missing owner")
	ErrParseToFrontmatter   = errors.New("could not parse post frontmatter")
	ErrNoPostTitle          = errors.New("post missing title")
	ErrReadingFile          = errors.New("could not read from file")
)

const maxChunkReadSize = 32 * 1024 // 32KB

func readFileToBytes(path string) ([]byte, error) {
	file, err := os.OpenInRoot(filepath.Split(path))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var buf bytes.Buffer
	buf.Grow(maxChunkReadSize)
	if _, err = buf.ReadFrom(bufio.NewReaderSize(file, maxChunkReadSize)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func ParseBlogManifest(path string) (*BlogManifest, error) {

	data, err := readFileToBytes(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadingFile, err)
	}

	var manifest BlogManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnmarshallToManifest, err)
	}

	if manifest.Title == "" {
		return nil, ErrNoBlogTitle
	}
	if manifest.Owner == "" {
		return nil, ErrNoBlogOwner
	}

	// apply defaults
	if manifest.Visibility == "" {
		manifest.Visibility = storage.VisibilityPublic
	}
	if manifest.RegistrationMode == "" {
		manifest.RegistrationMode = storage.RegistrationOpen
	}
	return &manifest, nil
}

func ParsePostFile(path string) (*PostFrontmatter, []byte, error) {
	data, err := readFileToBytes(path)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrReadingFile, err)
	}

	var fm PostFrontmatter
	body, err := frontmatter.Parse(bytes.NewReader(data), &fm)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrParseToFrontmatter, err)
	}

	if fm.Title == "" {
		return nil, nil, ErrNoPostTitle
	}
	return &fm, body, nil
}
