package sqlite

import (
	"blogengine/internal/storage"
	"context"
	"fmt"
)

func (s *Store) CreateUser(ctx context.Context, username, passwordHash string) (*storage.User, error) {
	query := `INSERT INTO users (username, password_hash)
		VALUES (?, ?)
		RETURNING *`

	var user storage.User
	if err := s.db.GetContext(ctx, &user, query, username, passwordHash); err != nil {
		return nil, fmt.Errorf("cannot create user %q: %w", username, mapSqlError(err))
	}
	return &user, nil
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (*storage.User, error) {
	query := `SELECT * FROM users
		WHERE id = ? AND deleted_at IS NULL
		LIMIT 1`

	var user storage.User
	if err := s.db.GetContext(ctx, &user, query, id); err != nil {
		return nil, fmt.Errorf("cannot find user id %d: %w", id, mapSqlError(err))
	}
	return &user, nil
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (*storage.User, error) {
	query := `SELECT * FROM users
		WHERE username = ? AND deleted_at IS NULL
		LIMIT 1`

	var user storage.User
	if err := s.db.GetContext(ctx, &user, query, username); err != nil {
		return nil, fmt.Errorf("cannot find username %q: %w", username, mapSqlError(err))
	}
	return &user, nil
}

func (s *Store) ChangeUserPassword(ctx context.Context, userID int64, newHash string) error {
	query := `UPDATE users SET password_hash = ?
		WHERE id = ? AND deleted_at IS NULL`

	result, err := s.db.ExecContext(ctx, query, newHash, userID)
	if err != nil {
		return fmt.Errorf("could not update password: %w", mapSqlError(err))
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get rows affected: %w", mapSqlError(err))
	}
	if rows == 0 {
		return storage.ErrNotFound
	}

	return nil
}

func (s *Store) DeleteUser(ctx context.Context, userID int64) error {
	query := `UPDATE users SET deleted_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL`

	result, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("could not delete user: %w", mapSqlError(err))
	}

	if rows, _ := result.RowsAffected(); rows == 0 {
		return storage.ErrNotFound
	}

	return nil
}
