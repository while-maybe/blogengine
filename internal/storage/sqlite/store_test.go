package sqlite

import (
	"blogengine/internal/storage"
	"errors"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func TestStoreImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ storage.Store = (*Store)(nil)
}

func TestNewStore(t *testing.T) {
	t.Parallel()
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Fatal("Store is nil")
	}
}

func setupTestStore(t *testing.T) *Store {
	t.Helper()

	tempDir := t.TempDir()
	// dbPath := filepath.Join(tempDir, "test_blog.db")
	dbPath, _ := os.CreateTemp(tempDir, "test_blog.*.db")

	store, err := NewStore(dbPath.Name())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	migrationsPath := "../../../migrations"
	m, err := migrate.New("file://"+migrationsPath, "sqlite3://"+dbPath.Name())
	if err != nil {
		t.Fatalf("creating migrations failed: %v", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("migration failed: %v", err)
	}

	t.Cleanup(func() {
		m.Close()
		store.Close()
	})

	return store
}
