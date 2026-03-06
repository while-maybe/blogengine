package storage

import (
	"context"
	"errors"
	"time"
)

type Store interface {
	// store
	Close() error

	// users
	CreateUser(ctx context.Context, username, passwordHash string) (*User, error)
	GetUserByID(ctx context.Context, id int64) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	ChangeUserPassword(ctx context.Context, userID int64, newHash string) error
	DeleteUser(ctx context.Context, userID int64) error

	// comments
	CreateComment(ctx context.Context, postID, userID int64, content string) (*Comment, error)
	GetCommentByID(ctx context.Context, commentID int64) (*Comment, error)
	UpdateComment(ctx context.Context, commentID, userID int64, content string) (*Comment, error)
	DeleteComment(ctx context.Context, commentID, userID int64) error
	GetCommentsForPost(ctx context.Context, postID, offset, limit int64) ([]*Comment, error)
	GetCommentsForUserID(ctx context.Context, userID, offset, limit int64) ([]*Comment, error)

	// blogs
	CreateBlog(ctx context.Context, params CreateBlogParams) (*Blog, error)
	GetPublicBlogs(ctx context.Context, offset, limit int64) ([]*Blog, error)
	GetBlogByID(ctx context.Context, blogID int64) (*Blog, error)
	GetBlogBySlug(ctx context.Context, slug string) (*Blog, error)
	UpdateBlog(ctx context.Context, params UpdateBlogParams) (*Blog, error)
	UpdateBlogVisibility(ctx context.Context, blogID, ownerID int64, visibility Visibility) error
	UpdateBlogRegistration(ctx context.Context, params UpdateBlogRegistrationParams) error
	DeleteBlog(ctx context.Context, blogID, ownerID int64) error

	// posts
	CreatePost(ctx context.Context, params CreatePostParams) (*Post, error)
}

type Visibility string
type RegistrationMode string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"

	RegistrationOpen       RegistrationMode = "open"
	RegistrationClosed     RegistrationMode = "closed"
	RegistrationLimited    RegistrationMode = "limited"
	RegistrationInviteOnly RegistrationMode = "invite_only"
)

var (
	ErrNotFound        = errors.New("record not found")
	ErrUniqueViolation = errors.New("unique constraint violation")
	ErrCheckViolation  = errors.New("check constraint violation")
)

type User struct {
	ID           int64      `db:"id"`
	Username     string     `db:"username"`
	PasswordHash string     `db:"password_hash"`
	CreatedAt    time.Time  `db:"created_at"`
	DeletedAt    *time.Time `db:"deleted_at"`
}

type Comment struct {
	ID         int64      `db:"id"`
	PostID     int64      `db:"post_id"`
	UserID     *int64     `db:"user_id"`
	Content    string     `db:"content"`
	AuthorName string     `db:"author_name"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  *time.Time `db:"updated_at"`
	DeletedAt  *time.Time `db:"deleted_at"`
}

type Blog struct {
	ID                int64            `db:"id"`
	OwnerID           int64            `db:"owner_id"`
	Slug              string           `db:"slug"`
	Title             string           `db:"title"`
	Description       *string          `db:"description"`
	Visibility        Visibility       `db:"visibility"`
	RegistrationMode  RegistrationMode `db:"registration_mode"`
	RegistrationLimit *int64           `db:"registration_limit"`
	CreatedAt         time.Time        `db:"created_at"`
	UpdatedAt         *time.Time       `db:"updated_at"`
	DeletedAt         *time.Time       `db:"deleted_at"`
}

type CreateBlogParams struct {
	OwnerID           int64
	Slug              string
	Title             string
	Description       *string
	Visibility        Visibility
	RegistrationMode  RegistrationMode
	RegistrationLimit *int64
}

type UpdateBlogParams struct {
	BlogID      int64
	OwnerID     int64
	Slug        string
	Title       string
	Description *string
}

type UpdateBlogRegistrationParams struct {
	BlogID            int64
	OwnerID           int64
	RegistrationMode  RegistrationMode
	RegistrationLimit *int64
}

type Post struct {
	ID            int64      `db:"id"`
	BlogID        int64      `db:"blog_id"`
	AuthorID      int64      `db:"author_id"`
	PublicID      string     `db:"public_id"`
	Slug          *string    `db:"slug"`
	Title         string     `db:"title"`
	Description   *string    `db:"description"`
	S3Key         string     `db:"s3_key"`
	IsEncrypted   bool       `db:"is_encrypted"`
	EncryptionIV  *string    `db:"encryption_iv"`
	RequiresAuth  bool       `db:"requires_auth"`
	IsListed      bool       `db:"is_listed"`
	AllowComments bool       `db:"allow_comments"`
	PublishedAt   *time.Time `db:"published_at"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     *time.Time `db:"updated_at"`
	DeletedAt     *time.Time `db:"deleted_at"`
}

type CreatePostParams struct {
	BlogID        int64
	AuthorID      int64
	Slug          *string
	Title         string
	Description   *string
	IsEncrypted   bool
	EncryptionIV  *string
	RequiresAuth  bool
	IsListed      bool
	AllowComments bool
	PublishedAt   *time.Time
}

func (v Visibility) IsValid() bool {
	switch v {
	case VisibilityPublic, VisibilityPrivate:
		return true
	}
	return false
}

func (r RegistrationMode) IsValid() bool {
	switch r {
	case RegistrationOpen, RegistrationClosed, RegistrationLimited, RegistrationInviteOnly:
		return true
	}
	return false
}
