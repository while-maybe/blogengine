package sqlite

import (
	"blogengine/internal/storage"
	"context"
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
)

const (
	defaultAdminUsername = "admin"
	defaultAdminPassword = "adminadmin"
)

var (
	ErrCountUsers        = errors.New("failed to count users")
	ErrPasswordHash      = errors.New("could not generate password hash")
	ErrCreateAdminUser   = errors.New("could not create admin user")
	ErrCreateDefaultBlog = errors.New("could not create default blog")
)

// Bootstrap ensure the system has at least one admin and one default blog
func (s *Store) Bootstrap(ctx context.Context, logger *slog.Logger) error {
	logger.Info("bootstrapping database...")

	adminUser, err := s.getOrCreateAdminUser(ctx, logger)
	if err != nil {
		return err
	}
	logger.Info("admin user created", "user", adminUser.Username)
	return nil
}

func (s *Store) getOrCreateAdminUser(ctx context.Context, logger *slog.Logger) (*storage.User, error) {
	u, err := s.GetUserByUsername(ctx, defaultAdminUsername)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrNotFound):
			logger.Info("no admin user found, creating default 'admin' user")
			hash, err := bcrypt.GenerateFromPassword([]byte(defaultAdminPassword), bcrypt.DefaultCost)
			if err != nil {
				return nil, fmt.Errorf("%w: %w", ErrPasswordHash, err)
			}
			u, err = s.CreateUser(ctx, defaultAdminUsername, string(hash))
			if err != nil {
				return nil, fmt.Errorf("%w: %w", ErrCreateAdminUser, err)
			}
			logger.Warn("SYSTEM BOOTSTRAP: created default admin user. Change the password NOW!!!", "username", defaultAdminUsername, "password", defaultAdminPassword)
		default:
			return nil, err
		}
	}
	return u, nil
}
