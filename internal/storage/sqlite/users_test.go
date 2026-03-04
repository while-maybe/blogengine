package sqlite

import (
	"blogengine/internal/storage"
	"context"
	"errors"
	"testing"
	"time"
)

func TestCreateUser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		want    storage.User
		wantErr error
	}{
		{
			name: "nominal",
			want: storage.User{Username: "user_" + gen60CharString()[:5], PasswordHash: gen60CharString()},
		},
		{
			name:    "username len < 3",
			want:    storage.User{Username: "xx", PasswordHash: gen60CharString()},
			wantErr: storage.ErrCheckViolation,
		},
		{
			name:    "username len > 50",
			want:    storage.User{Username: gen60CharString(), PasswordHash: gen60CharString()},
			wantErr: storage.ErrCheckViolation,
		},
		{
			name:    "hash len not 60",
			want:    storage.User{Username: "user_" + gen60CharString()[:5], PasswordHash: gen60CharString()[:40]},
			wantErr: storage.ErrCheckViolation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupTestStore(t)
			ctx := context.Background()

			t.Parallel()

			got, gotErr := store.CreateUser(ctx, tt.want.Username, tt.want.PasswordHash)
			if !errors.Is(gotErr, tt.wantErr) {
				t.Fatalf("error creating user: got %v, want %v", gotErr, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got.Username != tt.want.Username {
				t.Errorf("invalid username: got %q, want %q", got.Username, tt.want.Username)
			}
			if got.PasswordHash != tt.want.PasswordHash {
				t.Errorf("invalid pwd: got %q, want %q", got.PasswordHash, tt.want.PasswordHash)
			}
			if got.DeletedAt != nil {
				t.Errorf("invalid deleted time: %s", got.DeletedAt)
			}
			if time.Since(got.CreatedAt) > 1*time.Second {
				t.Errorf("invalid creation time: %s", got.CreatedAt)
			}

		})
	}
}

func TestGetUserByUsername(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		username string
		want     storage.User
		wantErr  error
	}{
		{
			name:     "nominal",
			username: "testuser",
			want:     storage.User{Username: "testuser", PasswordHash: gen60CharString()},
			wantErr:  nil,
		},
		{
			name:     "user does not exist",
			username: "username",
			want:     storage.User{Username: "user_" + gen60CharString()[:5], PasswordHash: gen60CharString()},
			wantErr:  storage.ErrNotFound,
		},
		{
			name:     "case sensitivity - search lowercase for Uppercase",
			username: "AdminUser",
			want:     storage.User{Username: "adminuser", PasswordHash: gen60CharString()},
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupTestStore(t)
			ctx := context.Background()

			t.Parallel()

			_, err := store.CreateUser(ctx, tt.username, tt.want.PasswordHash)
			if err != nil {
				t.Fatalf("error creating user: %v", err)
			}

			got, err := store.GetUserByUsername(ctx, tt.want.Username)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error GetUserByUsername: got %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}

			if tt.username != got.Username {
				t.Errorf("username does not match: want %q, got %q", tt.want.Username, got.Username)
			}
		})
	}
}

func TestGetUserByID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		id      int64
		wantID  int64
		wantErr error
	}{
		{
			name:    "nominal",
			wantID:  1,
			wantErr: nil,
		},
		{
			name:    "user ID does not exist",
			wantID:  99,
			wantErr: storage.ErrNotFound,
		},
		{
			name:    "invalid ID",
			wantID:  -5,
			wantErr: storage.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			store := setupTestStore(t)
			ctx := context.Background()

			t.Parallel()

			u, err := store.CreateUser(ctx, "user_"+gen60CharString()[:5], gen60CharString())
			if err != nil {
				t.Fatalf("error creating user: %v", err)
			}

			searchID := tt.wantID
			if tt.name == "nominal" {
				searchID = u.ID
			}

			got, err := store.GetUserByID(ctx, searchID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error GetUserByUsername: got %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}

			if got.ID != u.ID {
				t.Errorf("user ID does not match: want %d, got %d", tt.wantID, got.ID)
			}
		})
	}
}

func TestChangeUserPassword(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		id      int64
		newHash string
		wantErr error
	}{
		{
			name:    "nominal",
			newHash: gen60CharString(),
			wantErr: nil,
		},
		{
			name:    "inexistent id",
			id:      1_001,
			newHash: gen60CharString(),
			wantErr: storage.ErrNotFound,
		},
		{
			name:    "hash length < 60",
			newHash: gen60CharString()[:30],
			wantErr: storage.ErrCheckViolation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupTestStore(t)
			ctx := context.Background()

			t.Parallel()

			username := "user_" + gen60CharString()[:5]
			hash := gen60CharString()

			u, err := store.CreateUser(ctx, username, hash)
			if err != nil {
				t.Fatalf("could not create new user: %v", err)
			}

			// there wouldn't be a naturally created record with id > 0
			workingID := u.ID
			if tt.id > 0 {
				workingID = tt.id
			}

			err = store.ChangeUserPassword(ctx, workingID, tt.newHash)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("unexpected error: got %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				return
			}

			got, err := store.GetUserByID(ctx, workingID)
			if err != nil {
				t.Fatalf("could not fetch user after password change: %v", err)
			}

			if hash == got.PasswordHash {
				t.Errorf("password was not modified")
			}
		})
	}
}

func TestDeleteUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		id            int64
		wantIsDeleted bool
		wantErr       error
	}{
		{
			name:          "nominal",
			wantIsDeleted: true,
			wantErr:       nil,
		},
		{
			name:          "inexistent id",
			wantIsDeleted: false,
			id:            1_0001,
			wantErr:       storage.ErrNotFound,
		},
		{
			name:          "invalid id",
			wantIsDeleted: false,
			id:            -5,
			wantErr:       storage.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupTestStore(t)
			ctx := context.Background()

			t.Parallel()

			username := "user_" + gen60CharString()[:5]
			hash := gen60CharString()

			u, err := store.CreateUser(ctx, username, hash)
			if err != nil {
				t.Fatalf("could not create new user: %v", err)
			}

			// positive test does inject id
			workingID := u.ID
			if tt.name != "nominal" {
				workingID = tt.id
			}

			if err := store.DeleteUser(ctx, workingID); !errors.Is(err, tt.wantErr) {
				t.Fatalf("could not delete user: %v", err)
			}

			if tt.wantErr != nil {
				return
			}

			// GetUserById is not getting tested
			_, err = store.GetUserByID(ctx, u.ID)

			if tt.wantIsDeleted && !errors.Is(err, storage.ErrNotFound) {
				t.Errorf("expected ErrNotFound for deleted user, got %v", err)
			}
		})
	}
}

func TestDeleteUser_ContextError(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())

	cancel()
	err := store.DeleteUser(ctx, 1)

	if err == nil {
		t.Error("expected execution error from canceled context, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context cancellation error, got: %v", err)
	}
}
