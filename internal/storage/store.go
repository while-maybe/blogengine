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
	CreateBlog(ctx context.Context, ownerID int64, slug, title string, description *string, visibility Visibility, registrationMode RegistrationMode, registrationLimit *int64) (*Blog, error)
	GetPublicBlogs(ctx context.Context, offset, limit int64) ([]*Blog, error)
	GetBlogByID(ctx context.Context, blogID int64) (*Blog, error)
	GetBlogBySlug(ctx context.Context, slug string) (*Blog, error)
	UpdateBlog(ctx context.Context, blogID, ownerID int64, slug, title string, description *string) (*Blog, error)
	UpdateBlogVisibility(ctx context.Context, blogID, ownerID int64, visibility Visibility) error
	UpdateBlogRegistration(ctx context.Context, blogID, ownerID int64, registrationMode RegistrationMode, registrationLimit *int64) error
	DeleteBlog(ctx context.Context, blogID, ownerID int64) error
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
	ID           int64      `db:"id" json:"id"`
	Username     string     `db:"username" json:"username"`
	PasswordHash string     `db:"password_hash" json:"-"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	DeletedAt    *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

type Comment struct {
	ID         int64      `db:"id" json:"id"`
	PostID     int64      `db:"post_id" json:"post_id"`
	UserID     *int64     `db:"user_id" json:"user_id"`
	Content    string     `db:"content" json:"content"`
	AuthorName string     `db:"author_name" json:"author_name"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt  *time.Time `db:"updated_at" json:"updated_at,omitempty"`
	DeletedAt  *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

type Blog struct {
	ID                int64            `db:"id" json:"id"`
	OwnerID           int64            `db:"owner_id" json:"owner_id"`
	Slug              string           `db:"slug" json:"slug"`
	Title             string           `db:"title" json:"title"`
	Description       *string          `db:"description" json:"description,omitempty"`
	Visibility        Visibility       `db:"visibility" json:"visibility"`
	RegistrationMode  RegistrationMode `db:"registration_mode" json:"registration_mode"`
	RegistrationLimit *int64           `db:"registration_limit" json:"registration_limit,omitempty"`
	CreatedAt         time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt         *time.Time       `db:"updated_at" json:"updated_at,omitempty"`
	DeletedAt         *time.Time       `db:"deleted_at" json:"deleted_at,omitempty"`
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
