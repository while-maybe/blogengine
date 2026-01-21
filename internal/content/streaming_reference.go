//go:build ignore

package content

// This file is preserved for reference to show how streaming was handled before switching to caching.

// func (p *Post) GetContentReader() (io.ReadCloser, error) {
// 	// if cached, return a reader over cached content
// 	p.mu.RLock()
// 	if p.Content != nil {
// 		defer p.mu.RUnlock()
// 		return io.NopCloser(bytes.NewReader(p.Content)), nil
// 	}
// 	p.mu.RUnlock()

// 	// load post from disk
// 	p.mu.Lock()
// 	defer p.mu.Unlock()

// 	// check if another goroutine wrote to cache
// 	if p.Content != nil {
// 		return io.NopCloser(bytes.NewReader(p.Content)), nil
// 	}

// 	// fetch it from disk
// 	fmt.Printf("Loading content from disk for: %s\n", p.Title)

// 	file, err := os.OpenInRoot(filepath.Split(p.Path))
// 	if err != nil {
// 		return nil, fmt.Errorf("could not open file %s: %w", p.Path, err)
// 	}

// 	stats, err := file.Stat()
// 	if err != nil {
// 		file.Close()
// 		return nil, fmt.Errorf("could not stat file %s: %w", p.Path, err)
// 	}

// 	if stats.Size() > maxFileSize { // 10MB limit
// 		file.Close()
// 		return nil, fmt.Errorf("file too large %s: %d bytes", p.Path, stats.Size())
// 	}

// 	bufferSize := min(maxBufferSize, int(stats.Size()))

// 	return struct {
// 		io.Reader
// 		io.Closer
// 	}{
// 		Reader: bufio.NewReaderSize(file, bufferSize),
// 		Closer: file,
// 	}, nil
// }

// func (p *Post) ServeContent(w http.ResponseWriter) error {
// 	// stream large files directly
// 	contentReader, err := p.GetContentReader()
// 	if err != nil {
// 		return fmt.Errorf("could not get content reader: %w", err)
// 	}
// 	defer contentReader.Close()

// 	// stream directly from file to HTTP response
// 	_, err = io.Copy(w, contentReader)
// 	return err
// }
