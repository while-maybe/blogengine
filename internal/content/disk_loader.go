package content

import (
	"bufio"
	"bytes"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
)

const maxBufferSize = 32 * 1024
const maxFileSize = 10 * 1024 * 1024

type metaData struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"` // for SEO
	Author      string `yaml:"author"`
	CreatedAt   string `yaml:"created_at"`
	ModifiedAt  string `yaml:"modified_at"`
	Draft       bool   `yaml:"draft"`
	NoIndex     bool   `yaml:"noindex"`
}

func (p *Repository) LoadLazyMetaFromDisk(paths []string) error {
	if len(paths) == 0 {
		return ErrNoFilePaths
	}

	for _, fileName := range paths {
		func() {
			cleanPath := strings.TrimSpace(fileName)

			file, err := os.Open(cleanPath)
			if err != nil {
				fmt.Printf("%v: %s: %v\n", ErrReadingFile, cleanPath, err)
			}
			defer file.Close()

			var meta metaData

			_, err = frontmatter.Parse(file, &meta)

			// fallback for files without frontmatter
			if err != nil || meta.Title == "" {
				file.Seek(0, 0)
				meta.Title = fallbackTitleScan(file)
			}

			postDate := time.Now().UTC()
			if stats, err := file.Stat(); err != nil {
				fmt.Printf("%v: %s, keeping current time\n", ErrFileStats, cleanPath)
			} else {
				postDate = stats.ModTime().UTC()
			}

			if meta.ModifiedAt != "" {
				if parsed, err := time.Parse("2006-01-02", meta.ModifiedAt); err == nil {
					postDate = parsed
				}
			}

			id := crc32.ChecksumIEEE([]byte(cleanPath))

			newPost := &Post{
				ID:          id,
				Title:       meta.Title,
				Description: meta.Description,
				Author:      meta.Author,
				CreatedAt:   time.Time{},
				ModifiedAt:  postDate,
				Path:        cleanPath,
				IsSafeHTML:  false,
			}

			p.mu.Lock()
			p.Data[newPost.ID] = newPost
			p.mu.Unlock()

			fmt.Printf(" -> Found: %q [%s]\n", cleanPath, meta.Title)
		}()
	}
	return nil
}

func fallbackTitleScan(r io.Reader) string {
	scanner := bufio.NewScanner(r)
	// if title is not within first 20 lines, it's likely not there at all
	linesScanned := 0
	for scanner.Scan() {
		linesScanned++
		if linesScanned > 20 {
			break
		}
		if _, title, found := strings.Cut(scanner.Text(), "# "); found {
			return strings.TrimSpace(title)
		}
	}
	return "Untitled Post"
}

// GetContent returns the cached content or loads it from disk if missing.
func (p *Post) GetContent() ([]byte, error) {
	// return contents directly if already cached
	p.mu.RLock()
	if p.Content != nil {
		defer p.mu.RUnlock()
		return p.Content, nil
	}
	p.mu.RUnlock()

	// load post from disk
	p.mu.Lock()
	defer p.mu.Unlock()

	// check if another goroutine wrote to cache
	if p.Content != nil {
		return p.Content, nil
	}

	// fetch it from disk
	fmt.Printf("Loading content from disk for: %s\n", p.Title)

	file, err := os.OpenInRoot(filepath.Split(p.Path))
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrReadingFile, p.Path, err)
	}
	defer file.Close()

	stats, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFileStats, err)
	}

	// Safety check for huge files
	if stats.Size() > maxFileSize { // 10MB limit
		return nil, fmt.Errorf("%w: %d bytes", ErrFileTooLarge, stats.Size())
	}
	// use file size as buffer size when smaller than max to reduce memory footprint
	postBufferSize := min(maxBufferSize, int(stats.Size()))

	// normally conversion int64 -> int would be dangerous but blog post markdown files < maxFileSize...
	bufReader := bufio.NewReaderSize(file, postBufferSize)

	// read the entire content from the buffer
	postBytes, err := io.ReadAll(bufReader)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrBufferError, p.Path, err)
	}

	// separate frontmatter data from body
	var meta metaData
	markdownBody, err := frontmatter.Parse(bytes.NewReader(postBytes), &meta)
	if err != nil {
		// no yaml detected  or parsing failed, proceed with original raw bytes
		markdownBody = postBytes
	}

	// into cache
	if p.Content, err = mdToHTML(markdownBody); err != nil {
		return []byte{}, fmt.Errorf("%w: %v", ErrContentUnavailable, err)
	}
	return p.Content, nil
}
