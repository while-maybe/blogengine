package sqlite

import (
	"blogengine/internal/storage"
	"blogengine/internal/utils"
	"context"
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
)

const (
	defaultAdminUsername = "admin"
	defaultAdminPassword = "admin"

	defaultStarterBlogTitle       = "Hello World starter blog"
	defaultStarterBlogDescription = "this is the initial blog created upon database bootstrapping"
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

	defaultBlog, err := s.getOrCreateDefaultBlog(ctx, logger, adminUser.ID, defaultStarterBlogTitle)
	if err != nil {
		return err
	}
	logger.Info("default blog created", "title", defaultBlog.Title, "slug", defaultBlog.Slug)

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

func (s *Store) getOrCreateDefaultBlog(ctx context.Context, logger *slog.Logger, ownerId int64, starterBlogTitle string) (*storage.Blog, error) {
	slug := utils.Slugify(starterBlogTitle)

	blog, err := s.GetBlogBySlug(ctx, slug)
	// return if error is anything except ErrNotFound
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	if err == nil {
		logger.Warn("default blog already exists, skipping", "title", blog.Title, "slug", blog.Slug)
		return blog, nil
	}
	createBlogParams := storage.CreateBlogParams{
		OwnerID:           ownerId,
		Slug:              slug,
		Title:             starterBlogTitle,
		Description:       new(defaultStarterBlogDescription),
		Visibility:        storage.VisibilityPublic,
		RegistrationMode:  storage.RegistrationClosed,
		RegistrationLimit: nil,
	}
	blog, err = s.CreateBlog(
		ctx, createBlogParams)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateDefaultBlog, err)
	}
	return blog, nil
}
