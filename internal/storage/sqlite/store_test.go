package sqlite

import (
	"blogengine/internal/storage"
	"errors"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
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
	if err := store.Migrate(migrationsPath); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("migration failed: %v", err)
	}

	t.Cleanup(func() {
		store.Close()
	})

	return store
}
