package sqlite

import "errors"

var (
	// blogs
	ErrBlogSlug                  = errors.New("slug must be between 5 and 100 chars (lower case letters, digits and '-')")
	ErrBlogTitle                 = errors.New("title must be between 5 and 100 chars")
	ErrBlogDescription           = errors.New("description can only be nil OR less than 500 chars")
	ErrBlogVisibility            = errors.New("visibility must be 'private' or 'public'")
	ErrBlogRegistrationMode      = errors.New("unknown registration mode")
	ErrBlogRegistrationLimit     = errors.New("registration limit must be valid")
	ErrCreateBlog                = errors.New("could not create blog")
	ErrLimitOffset               = errors.New("offset must be >= 0 and limit > 0")
	ErrAllPublicBlogs            = errors.New("could not get public blog list")
	ErrInvalidBlogID             = errors.New("blog id must be > 0")
	ErrGetBlogByID               = errors.New("could not get blog by id")
	ErrGetBlogBySlug             = errors.New("could not get blog by slug")
	ErrUpdateBlog                = errors.New("could not update blog")
	ErrNegativeIDs               = errors.New("given id(s) must be > 0")
	ErrUpdateBlogVisibility      = errors.New("could not update blog visibility")
	ErrUpdateBlogRegistration    = errors.New("could not update blog registration")
	ErrRegistrationValuesForMode = errors.New("registration mode incompatible with provided limit")
	ErrDeleteBlog                = errors.New("could not delete blog")
)
