package content

import "errors"

var (
	ErrPostNotFound = errors.New("post not found")
	ErrReadingFile  = errors.New("could not open file")
	ErrNoFilePaths  = errors.New("must provide file path(s)")
	ErrFileStats    = errors.New("could not get stats")
	ErrFileTooLarge = errors.New("file too large")
	ErrBufferError  = errors.New("could not read from buffer")
	// markdown
	ErrMDConversion = errors.New("could not convert MD to HTML")
	// posts
	ErrRepositoryTitle = errors.New("repository must have a title")
	// disk_loader
	ErrContentUnavailable = errors.New("could not get content")
)
