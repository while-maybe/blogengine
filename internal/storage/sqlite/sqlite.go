package sqlite

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// NewDB initializes the SQLite connection
func NewDB(path string) (*sqlx.DB, error) {
	dsn := fmt.Sprintf("%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", path)

	db, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("cannot open db: %w", err)
	}

	db.SetMaxOpenConns(1) // it's file based...
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	return db, nil
}

// _pragma=foreign_keys(1)        → Enable foreign key constraints
// _pragma=journal_mode(WAL)      → Write-Ahead Logging (better concurrency)
// _pragma=busy_timeout(5000)     → Wait 5s instead of failing on lock contention
