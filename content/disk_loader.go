package content

import (
	"bufio"
	"fmt"
	"hash/crc32"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxBufferSize = 32 * 1024
const maxFileSize = 10 * 1024 * 1024

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

			scanner := bufio.NewScanner(file)

			var title string
			for scanner.Scan() {
				line := scanner.Text()

				// Skip YAML separator
				if strings.HasPrefix(line, "---") {
					continue
				}

				var found bool
				if title, found = extractTitle(line); found {
					break
				}
			}

			id := crc32.ChecksumIEEE([]byte(cleanPath))

			modified := time.Now().UTC()

			stats, err := file.Stat()
			if err != nil {
				fmt.Printf("%v: %s, defaulting to current time\n", ErrFileStats, cleanPath)
			} else {
				modified = stats.ModTime().UTC()
			}

			// TODO - investigate collecting size to prevent huge files to reside in the cache

			newPost := &Post{
				ID:         id,
				Author:     rand.IntN(1_000),
				Title:      title,
				CreatedAt:  time.Time{},
				ModifiedAt: modified,
				Path:       cleanPath,
				IsSafeHTML: false,
			}

			p.mu.Lock()
			p.Data[newPost.ID] = newPost
			p.mu.Unlock()
			// contents[newPost.id] = mdToHTML(fileName, io.ReadAll())
			fmt.Printf(" -> Found: %q\n", cleanPath)
		}()
	}
	return nil
}

// extractTitle is a helper to grab title from "# Title" or "Title: ..."
func extractTitle(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return "", false
	}

	if _, title, found := strings.Cut(line, "# "); found {
		return strings.TrimSpace(title), true
	}
	return strings.TrimSpace(line), false
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
	// use file size as buffer when smaller than max to reduce memory footprint
	postBufferSize := min(maxBufferSize, int(stats.Size()))

	// normally conversion int64 -> int would be dangerous but blog post markdown files < maxFileSize...
	bufReader := bufio.NewReaderSize(file, postBufferSize)

	postBytes, err := io.ReadAll(bufReader)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrBufferError, p.Path, err)
	}

	// into cache
	if p.Content, err = mdToHTML(postBytes); err != nil {
		return []byte{}, fmt.Errorf("%w: %v", ErrContentUnavailable, err)
	}
	return p.Content, nil
}
