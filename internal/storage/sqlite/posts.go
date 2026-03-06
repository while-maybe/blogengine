package sqlite

import (
	"blogengine/internal/storage"
	"blogengine/internal/utils"
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidAuthorOrBlog = errors.New("author id and blog id must be > 0")
	ErrPostSlug            = errors.New("slug must be between 5 and 100 chars (lower case letters, digits and '-')")
	ErrPostTitle           = errors.New("title must be between 5 and 100 chars")
	ErrPostDescription     = errors.New("description can only be nil OR less than 500 chars")
	ErrMissingBlogSlug     = errors.New("could not get blog slug")
	ErrGenerateS3Key       = errors.New("could not generate post s3 key")
	ErrEncPointlessIv      = errors.New("encryption iv was given for plain content")
	ErrEncMissingIV        = errors.New("encrypted content must have iv")
	ErrEncSettingsConflict = errors.New("given encryption settings are not compatible with each other")
	ErrCreatingPost        = errors.New("could not create post")
	ErrInvalidPublicID     = errors.New("public id must not be empty")
)

func (s *Store) CreatePost(ctx context.Context, p storage.CreatePostParams) (*storage.Post, error) {
	if p.AuthorID < 1 || p.BlogID < 1 {
		return nil, fmt.Errorf("%w: %w", ErrCreatingPost, ErrInvalidAuthorOrBlog)
	}
	if err := validatePostDetails(p.Slug, p.Title, p.Description); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreatingPost, err)
	}
	publicID, err := utils.GeneratePublicID(12)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreatingPost, err)
	}
	s3Key, err := s.genPostS3Key(ctx, p.BlogID, publicID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreatingPost, err)
	}
	if err := validatePostEncryptionSettings(p.IsEncrypted, p.EncryptionIV); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreatingPost, err)
	}

	query := `INSERT INTO posts (blog_id, author_id, public_id, slug, title, description, s3_key, is_encrypted, encryption_iv, requires_auth, is_listed, allow_comments, published_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				RETURNING id, blog_id, author_id, public_id, slug, title, description, s3_key, is_encrypted, encryption_iv, requires_auth, is_listed, allow_comments, published_at, created_at`

	var post storage.Post
	if err := s.db.GetContext(ctx, &post, query, p.BlogID, p.AuthorID, publicID, p.Slug, p.Title, p.Description, s3Key, p.IsEncrypted, p.EncryptionIV, p.RequiresAuth, p.IsListed, p.AllowComments, p.PublishedAt); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreatingPost, err)
	}
	return &post, nil
}

func validatePostDetails(slug *string, title string, description *string) error {
	if err := validatePostSlug(slug); err != nil {
		return err
	}
	if len(title) < minTitleLen || len(title) > maxTitleLen {
		return ErrPostTitle
	}
	if description != nil && len(*description) > maxDescriptionLen {
		return ErrPostDescription
	}
	return nil
}

func validatePostSlug(slug *string) error {
	if slug == nil {
		return nil
	}
	if len(*slug) < minSlugLen || len(*slug) > maxSlugLen || !validSlug.MatchString(*slug) {
		return ErrPostSlug
	}
	return nil
}

func (s *Store) genPostS3Key(ctx context.Context, blogID int64, publicID string) (string, error) {
	if blogID < 1 {
		return "", ErrInvalidAuthorOrBlog
	}
	if publicID == "" {
		return "", ErrInvalidPublicID
	}
	query := `SELECT slug 
				FROM blogs
				WHERE id = ? AND deleted_at IS NULL
				LIMIT 1`

	var blogSlug string
	if err := s.db.GetContext(ctx, &blogSlug, query, blogID); err != nil {
		return "", fmt.Errorf("%w: %w", ErrGenerateS3Key, err)
	}
	s3Key := strings.Join([]string{blogSlug, publicID}, "/")

	return s3Key, nil
}

func postIdentifier(slug *string, title string) string {
	if slug != nil {
		return *slug
	}
	return utils.Slugify(title)
}

func validatePostEncryptionSettings(isEncrypted bool, encIV *string) error {
	if !isEncrypted && encIV != nil {
		return ErrEncPointlessIv
	}
	if isEncrypted && encIV == nil {
		return ErrEncMissingIV
	}
	return nil
}
