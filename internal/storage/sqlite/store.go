package sqlite

import (
	"blogengine/internal/storage"
	"context"
	"errors"

	"github.com/jmoiron/sqlx"
)

type Store struct {
	db *sqlx.DB
}

var _ storage.Store = (*Store)(nil)

// NewStore creates a new database store
func NewStore(dbPath string) (*Store, error) {
	db, err := NewDB(dbPath)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

func validateContent(content string) error {
	if content == "" {
		return errors.New("content cannot be empty")
	}
	if len(content) > 10000 {
		return errors.New("content too long")
	}
	return nil
}

func (s *Store) WithTx(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// TODO update modified_at when changing a comment content
// TODO LIMIT 1 for 1 entry operations
// TODO check if idempotentency should be present (updating returns the updated object instead of nil error)
// TODO get user functions to now consider is user has been soft deleted
