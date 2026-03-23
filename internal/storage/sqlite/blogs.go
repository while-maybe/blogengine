package sqlite

import (
	"blogengine/internal/storage"
	"context"
	"fmt"
	"regexp"
)

const (
	maxRegistrationQueue = 1_000
	minSlugLen           = 5
	maxSlugLen           = 100
	minTitleLen          = 5
	maxTitleLen          = 100
	maxDescriptionLen    = 500
)

var (
	validSlug = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

func (s *Store) CreateBlog(ctx context.Context, p storage.CreateBlogParams) (*storage.Blog, error) {
	if err := validateBlog(p.Slug, p.Title, p.Description); err != nil {
		return nil, err
	}
	if err := validateBlogVisibility(p.Visibility); err != nil {
		return nil, err
	}
	if err := validateBlogRegistration(p.RegistrationMode, p.RegistrationLimit); err != nil {
		return nil, err
	}

	query := `INSERT INTO blogs (owner_id, slug, title, description, visibility, registration_mode, registration_limit)
				VALUES (?, ?, ?, ?, ?, ?, ?)
				RETURNING id, owner_id, slug, title, description, visibility, registration_mode, registration_limit, created_at`

	var blog storage.Blog
	if err := s.db.GetContext(ctx, &blog, query, p.OwnerID, p.Slug, p.Title, p.Description, p.Visibility, p.RegistrationMode, p.RegistrationLimit); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateBlog, mapSqlError(err))
	}

	return &blog, nil
}

func (s *Store) GetPublicBlogs(ctx context.Context, offset, limit int64) ([]*storage.Blog, error) {
	if offset < 0 || limit <= 0 {
		return nil, ErrLimitOffset
	}

	query := `SELECT b.id, b.owner_id, b.slug, b.title, b.description, b.visibility, b.registration_mode, b.registration_limit, b.created_at, b.updated_at,
		u.username AS owner_name
				FROM blogs AS b
				JOIN users AS u ON u.id = b.owner_id
				WHERE b.visibility = 'public' AND b.deleted_at IS NULL
				ORDER BY b.created_at DESC
				LIMIT ?
				OFFSET ?`

	blogs := make([]*storage.Blog, 0)
	if err := s.db.SelectContext(ctx, &blogs, query, limit, offset); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAllPublicBlogs, mapSqlError(err))
	}
	return blogs, nil
}

func (s *Store) GetBlogByID(ctx context.Context, blogID int64) (*storage.Blog, error) {
	if blogID < 1 {
		return nil, ErrInvalidBlogID
	}

	query := `SELECT id, owner_id, slug, title, description, visibility, registration_mode, registration_limit, created_at, updated_at
				FROM blogs
				WHERE id = ? AND deleted_at IS NULL
				LIMIT 1`

	var blog storage.Blog
	if err := s.db.GetContext(ctx, &blog, query, blogID); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGetBlogByID, mapSqlError(err))
	}
	return &blog, nil
}

// GetBlogsByUserID returns all blogs including the private blogs. It is meant to list a user's own blogs (hence no privacy differentiation)
func (s *Store) GetBlogsByUserID(ctx context.Context, ownerID, offset, limit int64) ([]*storage.Blog, error) {
	if offset < 0 || limit <= 0 {
		return nil, fmt.Errorf("%w: %w", ErrBlogsByUserID, ErrLimitOffset)
	}
	if ownerID < 1 {
		return nil, fmt.Errorf("%w: %w", ErrBlogsByUserID, ErrInvalidOwnerID)
	}

	query := `SELECT id, owner_id, slug, title, description, visibility, registration_mode, registration_limit, created_at, updated_at
				FROM blogs
				WHERE owner_id = ? AND deleted_at IS NULL
				ORDER BY created_at DESC
				LIMIT ?
				OFFSET ?`

	blogs := make([]*storage.Blog, 0)
	if err := s.db.SelectContext(ctx, &blogs, query, ownerID, limit, offset); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBlogsByUserID, err)
	}
	return blogs, nil
}

func (s *Store) GetBlogBySlug(ctx context.Context, slug string) (*storage.Blog, error) {
	if err := validateBlogSlug(slug); err != nil {
		return nil, err
	}

	query := `SELECT id, owner_id, slug, title, description, visibility, registration_mode, registration_limit, created_at, updated_at
				FROM blogs
				WHERE slug = ? AND deleted_at IS NULL
				LIMIT 1`

	var blog storage.Blog
	if err := s.db.GetContext(ctx, &blog, query, slug); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGetBlogBySlug, mapSqlError(err))
	}
	return &blog, nil
}

func (s *Store) UpdateBlog(ctx context.Context, p storage.UpdateBlogParams) (*storage.Blog, error) {
	if p.BlogID < 1 || p.OwnerID < 1 {
		return nil, ErrNegativeIDs
	}

	if err := validateBlog(p.Slug, p.Title, p.Description); err != nil {
		return nil, err
	}

	query := `UPDATE blogs SET slug = ?, title = ?, description = ?
				WHERE id = ? AND owner_id = ? AND deleted_at IS NULL
				RETURNING id, owner_id, slug, title, description, visibility, registration_mode, registration_limit, created_at, updated_at`

	var blog storage.Blog
	if err := s.db.GetContext(ctx, &blog, query, p.Slug, p.Title, p.Description, p.BlogID, p.OwnerID); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUpdateBlog, mapSqlError(err))
	}

	return &blog, nil
}

func (s *Store) UpdateBlogVisibility(ctx context.Context, blogID, ownerID int64, visibility storage.Visibility) error {
	if blogID < 1 || ownerID < 1 {
		return ErrNegativeIDs
	}
	if err := validateBlogVisibility(visibility); err != nil {
		return err
	}

	query := `UPDATE blogs SET visibility = ?
				WHERE id = ? AND owner_id = ? AND deleted_at IS NULL`

	result, err := s.db.ExecContext(ctx, query, visibility, blogID, ownerID)
	if err != nil {
		return ErrUpdateBlogVisibility
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrBlogVisibility, mapSqlError(err))
	}
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (s *Store) UpdateBlogRegistration(ctx context.Context, p storage.UpdateBlogRegistrationParams) error {
	if p.BlogID < 1 || p.OwnerID < 1 {
		return ErrNegativeIDs
	}
	if err := validateBlogRegistration(p.RegistrationMode, p.RegistrationLimit); err != nil {
		return err
	}

	query := `UPDATE blogs SET registration_mode = ?, registration_limit = ?
				WHERE id = ? AND owner_id = ? AND deleted_at IS NULL`

	result, err := s.db.ExecContext(ctx, query, p.RegistrationMode, p.RegistrationLimit, p.BlogID, p.OwnerID)
	if err != nil {
		return ErrUpdateBlogRegistration
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUpdateBlogRegistration, mapSqlError(err))
	}
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (s *Store) DeleteBlog(ctx context.Context, blogID, ownerID int64) error {
	if blogID < 1 || ownerID < 1 {
		return ErrNegativeIDs
	}

	query := `UPDATE blogs SET deleted_at = CURRENT_TIMESTAMP
				WHERE id = ? AND owner_id = ? AND deleted_at IS NULL`

	result, err := s.db.ExecContext(ctx, query, blogID, ownerID)
	if err != nil {
		return ErrDeleteBlog
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDeleteBlog, mapSqlError(err))
	}
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func validateBlogSlug(slug string) error {
	if len(slug) < minSlugLen || len(slug) > maxSlugLen || !validSlug.MatchString(slug) {
		return ErrBlogSlug
	}
	return nil
}

func validateBlog(slug, title string, description *string) error {
	if err := validateBlogSlug(slug); err != nil {
		return err
	}
	if len(title) < minTitleLen || len(title) > maxTitleLen {
		return ErrBlogTitle
	}
	if description != nil && len(*description) > maxDescriptionLen {
		return ErrBlogDescription
	}
	return nil
}

func validateBlogVisibility(visibility storage.Visibility) error {
	if !visibility.IsValid() {
		return ErrBlogVisibility
	}
	return nil
}

func validateBlogRegistration(registrationMode storage.RegistrationMode, registrationLimit *int64) error {
	if !registrationMode.IsValid() {
		return ErrBlogRegistrationMode
	}

	if registrationMode == storage.RegistrationLimited {
		if registrationLimit == nil {
			return fmt.Errorf("%w: required for limited registration", ErrBlogRegistrationLimit)
		}
		if *registrationLimit < 1 || *registrationLimit > maxRegistrationQueue {
			return fmt.Errorf("%w: min 1, max %d", ErrBlogRegistrationLimit, maxRegistrationQueue)
		}
	}

	if registrationMode != storage.RegistrationLimited && registrationLimit != nil {
		return ErrRegistrationValuesForMode
	}
	return nil
}
