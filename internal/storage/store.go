package storage

import (
	"context"
	"errors"
	"time"
)

type Store interface {
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

	Close() error
}

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
	UserID     int64      `db:"user_id" json:"user_id"`
	Content    string     `db:"content" json:"content"`
	AuthorName string     `db:"author_name" json:"author_name"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt  *time.Time `db:"updated_at" json:"updated_at,omitempty"`
	DeletedAt  *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}
