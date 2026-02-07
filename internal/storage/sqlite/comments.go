package sqlite

import (
	"blogengine/internal/storage"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

func (s *Store) GetCommentByID(ctx context.Context, commentID int64) (*storage.Comment, error) {
	query := `SELECT c.id, c.post_id, c.content, c.created_at, c.deleted_at, COALESCE(u.username, 'deleted user') as author_name
		FROM comments AS c
		LEFT JOIN users AS u ON c.user_id = u.id
		WHERE c.id = ? AND c.deleted_at IS NULL
		LIMIT 1`

	var comment storage.Comment
	if err := s.db.GetContext(ctx, &comment, query, commentID); err != nil {
		return nil, fmt.Errorf("cannot find comment with ID %d: %w", commentID, mapSqlError(err))
	}

	return &comment, nil
}

func (s *Store) GetCommentsForPost(ctx context.Context, postID, offset, limit int64) ([]*storage.Comment, error) {
	query := `SELECT c.id, c.post_id, c.content, c.created_at, c.deleted_at, COALESCE(u.username, 'deleted user') as author_name
		FROM comments AS c
		LEFT JOIN users AS u ON c.user_id = u.id
		WHERE c.post_id = ? AND c.deleted_at IS NULL
		ORDER BY c.created_at DESC
		LIMIT ?
		OFFSET ?`

	var comments []*storage.Comment
	if err := s.db.SelectContext(ctx, &comments, query, postID, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", mapSqlError(err))
	}

	return comments, nil
}

func (s *Store) GetCommentsForUserID(ctx context.Context, userID, offset, limit int64) ([]*storage.Comment, error) {
	query := `SELECT c.id, c.post_id, c.content, c.created_at, c.deleted_at, COALESCE(u.username, 'deleted user') as author_name
		FROM comments AS c
		LEFT JOIN users AS u ON c.user_id = u.id
		WHERE c.user_id = ? AND c.deleted_at IS NULL
		ORDER BY c.created_at DESC
		LIMIT ?
		OFFSET ?`

	var comments []*storage.Comment
	if err := s.db.SelectContext(ctx, &comments, query, userID, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", mapSqlError(err))
	}

	return comments, nil
}

func (s *Store) CreateComment(ctx context.Context, postID, userID int64, content string) (*storage.Comment, error) {
	if err := validateContent(content); err != nil {
		return nil, err
	}

	query := `INSERT INTO comments (post_id, user_id, content)
		VALUES (?, ?, ?)
		RETURNING id, post_id, content, created_at, 
			(SELECT username FROM users WHERE id = ?) as author_name`

	var comment storage.Comment
	if err := s.db.GetContext(ctx, &comment, query, postID, userID, content, userID); err != nil {
		return nil, fmt.Errorf("could not create comment: %w", mapSqlError(err))
	}

	return &comment, nil
}

func (s *Store) UpdateComment(ctx context.Context, commentID, userID int64, content string) (*storage.Comment, error) {
	if err := validateContent(content); err != nil {
		return nil, err
	}

	query := `UPDATE comments SET content = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ? AND deleted_at IS NULL
		RETURNING id, post_id, user_id, content, created_at, updated_at,
		(SELECT username FROM users WHERE id = ?) as author_name`

	var comment storage.Comment
	if err := s.db.GetContext(ctx, &comment, query, content, commentID, userID, userID); err != nil {
		return nil, fmt.Errorf("could not update comment: %w", mapSqlError(err))
	}

	return &comment, nil
}

func (s *Store) DeleteComment(ctx context.Context, commentID, userID int64) error {
	query := `UPDATE comments SET deleted_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ? AND deleted_at IS NULL
		LIMIT 1`

	result, err := s.db.ExecContext(ctx, query, commentID, userID)

	if err != nil {
		return fmt.Errorf("could not delete comment: %w", mapSqlError(err))
	}

	if rows, _ := result.RowsAffected(); rows == 0 {
		return storage.ErrNotFound
	}

	return nil
}

func mapSqlError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return storage.ErrNotFound
	}

	// sqlite specific errors
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code() {

		case sqlite3.SQLITE_CONSTRAINT_UNIQUE:
			return storage.ErrUniqueViolation

		case sqlite3.SQLITE_CONSTRAINT_CHECK:
			return storage.ErrCheckViolation
			// other sqlite specific errors
		}
	}
	return err
}
