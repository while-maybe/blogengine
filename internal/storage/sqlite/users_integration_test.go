//go:build integration

package sqlite

import (
	"blogengine/internal/storage"
	"context"
	"errors"
	"testing"
)

func TestUserCRUD(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t)

	defer store.Close()

	ctx := context.Background()

	t.Run("create and get user", func(t *testing.T) {
		username := "testuser"

		hash := gen60CharString()

		// CreateUser
		user, err := store.CreateUser(ctx, username, hash)
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		if user.Username != username {
			t.Errorf("want %s, got %s", username, user.Username)
		}

		// GetUserByUsername
		foundByUsername, err := store.GetUserByUsername(ctx, username)
		if err != nil {
			t.Fatalf("failed to get user by username: %v", err)
		}

		if foundByUsername.ID != user.ID {
			t.Errorf("ID mismatch: want %d, got %d", user.ID, foundByUsername.ID)
		}

		// GetUserByID
		foundByID, err := store.GetUserByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("failed to get user by id: %v", err)
		}

		if foundByUsername.ID != user.ID {
			t.Errorf("ID mismatch: want %d, got %d", user.ID, foundByID.ID)
		}

		// ChangeUserPassword
		oldHash := user.PasswordHash
		newHash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lnew"
		if err := store.ChangeUserPassword(ctx, user.ID, newHash); err != nil {
			t.Fatalf("could not change password: %v", err)
		}
		updatedUser, err := store.GetUserByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("could not get updated user with ID %d: %v", user.ID, err)
		}
		if updatedUser.PasswordHash == oldHash {
			t.Errorf("password was not updated")
		}

		// DeleteUser
		if err := store.DeleteUser(ctx, user.ID); err != nil {
			t.Fatalf("could not delete user: %v", err)
		}
		_, err = store.GetUserByID(ctx, user.ID)
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("expected user to be soft deleted, got: %v", err)
		}

	})
}
