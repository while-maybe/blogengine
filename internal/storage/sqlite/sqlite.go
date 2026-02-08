package sqlite

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
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

func (s *Store) Migrate(migrationsPath string) error {
	// s.db is *sqlx.DB so s.db.DB is the underlying *sql.DB
	driver, err := sqlite.WithInstance(s.db.DB, &sqlite.Config{})
	if err != nil {
		return fmt.Errorf("could not create migrations driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://"+migrationsPath, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("migration setup failed: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration exec failed: %w", err)
	}

	return nil
}

// _pragma=foreign_keys(1)        → Enable foreign key constraints
// _pragma=journal_mode(WAL)      → Write-Ahead Logging (better concurrency)
// _pragma=busy_timeout(5000)     → Wait 5s instead of failing on lock contention
