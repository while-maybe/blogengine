package seeder

import (
	"blogengine/internal/storage"
	"blogengine/internal/utils"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

var (
	ErrSeedOpenSource       = errors.New("could not open source directory")
	ErrSeedReadSource       = errors.New("could not read source directory")
	ErrSeedBlog             = errors.New("could not seed blog")
	ErrSeedPost             = errors.New("could not seed post")
	ErrCreateBlogDir        = errors.New("could not create blog directory")
	ErrCreatePostDir        = errors.New("could not create post directory")
	ErrMovePostFile         = errors.New("could not move post file")
	ErrMoveBlogFile         = errors.New("could not move blog file")
	ErrResolveOwner         = errors.New("could not resolve blog owner")
	ErrUploadPost           = errors.New("could not upload post to S3")
	ErrOwnerNotFound        = errors.New("owner not found, create user first")
	ErrInvalidPublishedTime = errors.New("invalid published_at format, use RFC3339")
)

var (
	validPublicID = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
)

type Seeder struct {
	DB        storage.Store
	S3        storage.Provider
	SourceDir string
	Logger    *slog.Logger
}

func NewSeeder(db storage.Store, s3 storage.Provider, sourceDir string, logger *slog.Logger) *Seeder {
	return &Seeder{
		DB:        db,
		S3:        s3,
		SourceDir: sourceDir,
		Logger:    logger,
	}
}

func (s *Seeder) Seed(ctx context.Context) error {
	if err := s.SeedBlogs(ctx); err != nil {
		return err
	}

	return s.SeedPosts(ctx)
}

func (s *Seeder) SeedBlogs(ctx context.Context) error {
	root, err := os.OpenRoot(s.SourceDir)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSeedOpenSource, err)
	}
	defer root.Close()

	alreadyProcessed := make(map[string]struct{})

	wantedExts := []string{"*.yml", "*.yaml"}
	for _, ext := range wantedExts {
		glob, err := fs.Glob(root.FS(), ext)
		if err != nil {
			return err
		}
		// round one for loose blog (*.yaml) files in the SourceDir
		for _, blogFile := range glob {
			fullPath := filepath.Join(root.Name(), blogFile)
			manifest, err := ParseBlogManifest(fullPath)
			if err != nil {
				s.Logger.Error("failed to parse manifest, skipping", "file", blogFile, "err", err)
				continue
			}

			slug := utils.Slugify(manifest.Title)

			// remember that seedBlog also derives the new folder name from the slug
			if err := s.seedBlog(ctx, fullPath); err != nil {
				s.Logger.Error("failed to seed blog, skipping", "file", blogFile, "err", err)
				continue
			}
			alreadyProcessed[slug] = struct{}{}
		}
	}

	// process already-structured blog dirs
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSeedReadSource, err)
	}

	blogFilenames := []string{"blog.yml", "blog.yaml"}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, done := alreadyProcessed[entry.Name()]; done {
			continue
		}
		for _, ext := range blogFilenames {
			manifestPath := filepath.Join(root.Name(), entry.Name(), ext)

			if _, err := os.Stat(manifestPath); err != nil {
				continue
			}

			if err := s.seedBlog(ctx, manifestPath); err != nil {
				s.Logger.Error("failed to seed blog, skipping", "file", manifestPath, "err", err)
			}
			break
		}
	}
	return nil
}

func (s *Seeder) seedBlog(ctx context.Context, manifestPath string) error {
	manifest, err := ParseBlogManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSeedBlog, err)
	}

	owner, err := s.DB.GetUserByUsername(ctx, manifest.Owner)
	if err != nil {
		return fmt.Errorf("%w: %q: %w", ErrOwnerNotFound, manifest.Owner, err)
	}

	slug := utils.Slugify(manifest.Title)
	blogDir := filepath.Join(s.SourceDir, slug)

	// check if blog dir already exists, create otherwise
	existing, err := s.DB.GetBlogBySlug(ctx, slug)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrNotFound):
			// here for different handling if needed
		default:
			s.Logger.Warn("could not get slug", "slug", slug, "err", err)
			return fmt.Errorf("%w: %s: %w", ErrSeedBlog, slug, err)
		}
	}

	if existing != nil {
		s.Logger.Info("blog exists, updating", "slug", slug)

		updateParams := storage.UpdateBlogParams{
			BlogID:      existing.ID,
			OwnerID:     owner.ID,
			Slug:        slug,
			Title:       manifest.Title,
			Description: &manifest.Description,
		}
		if _, err := s.DB.UpdateBlog(ctx, updateParams); err != nil {
			return fmt.Errorf("%w: %s: %w", ErrSeedBlog, slug, err)
		}
		// move manifest into the folder if not already present
		if filepath.Dir(manifestPath) != blogDir {
			if err := os.MkdirAll(blogDir, os.ModePerm); err != nil {
				return fmt.Errorf("%w: %w", ErrCreateBlogDir, err)
			}
			if err := os.Rename(manifestPath, filepath.Join(blogDir, "blog.yml")); err != nil {
				return fmt.Errorf("%w: %w", ErrMoveBlogFile, err)
			}
		}
		return nil
	}

	// create blog directory
	if err := os.MkdirAll(blogDir, os.ModePerm); err != nil {
		return fmt.Errorf("%w: %w", ErrCreateBlogDir, err)
	}
	// move manifest into the blog folder
	if err := os.Rename(manifestPath, filepath.Join(blogDir, "blog.yml")); err != nil {
		return fmt.Errorf("%w: %w", ErrMoveBlogFile, err)
	}

	params := storage.CreateBlogParams{
		OwnerID:           owner.ID,
		Slug:              slug,
		Title:             manifest.Title,
		Description:       &manifest.Description,
		Visibility:        manifest.Visibility,
		RegistrationMode:  manifest.RegistrationMode,
		RegistrationLimit: manifest.RegistrationLimit,
	}
	if _, err := s.DB.CreateBlog(ctx, params); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrSeedBlog, slug, err)
	}

	s.Logger.Info("blog created", "slug", slug, "owner", manifest.Owner)
	return nil
}

func (s *Seeder) SeedPosts(ctx context.Context) error {
	root, err := os.OpenRoot(s.SourceDir)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSeedOpenSource, err)
	}
	blogDirs, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSeedReadSource, err)
	}

	// grab existing post IDs from DB
	existingIDs, err := s.loadExistingPostIDs(ctx)
	if err != nil {
		return err
	}

	for _, blogDir := range blogDirs {
		if !blogDir.IsDir() {
			continue
		}
		if err := s.seedPostsForBlog(ctx, blogDir.Name(), existingIDs); err != nil {
			s.Logger.Error("failed to seed posts for blog", "blog", blogDir.Name(), "err", err)
		}
	}
	return nil
}

func (s *Seeder) seedPost(ctx context.Context, blogSlug, postPath, publicID string) error {
	fm, body, err := ParsePostFile(postPath)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSeedPost, err)
	}
	blog, err := s.DB.GetBlogBySlug(ctx, blogSlug)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrNotFound):
			s.Logger.Warn("blog not found, skipping post", "blog", blogSlug, "err", err)
			return nil
		default:
			return fmt.Errorf("%w: %w", ErrSeedPost, err)
		}
	}

	// get a slug
	var postSlug *string
	if fm.Slug != "" {
		postSlug = &fm.Slug
	} else {
		s := utils.Slugify(fm.Title)
		postSlug = &s
	}

	// get publishedAt
	var publishedAt *time.Time
	if fm.PublishedAt != nil {

		var parsed time.Time
		formats := []string{time.RFC3339, "2006-01-02"}
		for _, format := range formats {
			if parsed, err = time.Parse(format, *fm.PublishedAt); err == nil {
				break
			}
		}
		if err != nil {
			return fmt.Errorf("%w: %w: %w", ErrSeedPost, ErrInvalidPublishedTime, err)
		}
		publishedAt = &parsed
	}

	var desc *string
	if fm.Description != "" {
		desc = &fm.Description
	}

	params := storage.CreatePostParams{
		PublicID:      publicID,
		BlogID:        blog.ID,
		AuthorID:      blog.OwnerID,
		Slug:          postSlug,
		Title:         fm.Title,
		Description:   desc,
		IsEncrypted:   false,
		EncryptionIV:  nil,
		RequiresAuth:  fm.RequiresAuth,
		IsListed:      fm.IsListed,
		AllowComments: fm.AllowComments,
		PublishedAt:   publishedAt,
	}

	post, err := s.DB.CreatePost(ctx, params)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSeedPost, err)
	}

	// only create folder and move file for new posts (loose .md files) in the blog root folder
	if publicID == "" {
		// create subfolder
		postDir := filepath.Join(s.SourceDir, blogSlug, post.PublicID)
		if err := os.MkdirAll(postDir, os.ModePerm); err != nil {
			return fmt.Errorf("%w: %w: %w", ErrSeedPost, ErrCreatePostDir, err)
		}
		// move file to folder
		destPath := filepath.Join(postDir, filepath.Base(postPath))
		if err := os.Rename(postPath, destPath); err != nil {
			return fmt.Errorf("%w: %w: %w", ErrSeedPost, ErrMovePostFile, err)
		}
	}

	// upload to object storage
	if err := s.S3.Save(ctx, post.S3Key, bytes.NewReader(body)); err != nil {
		return fmt.Errorf("%w: %w: %w", ErrSeedPost, ErrUploadPost, err)
	}

	s.Logger.Info("post seeded", "blog", blogSlug, "title", fm.Title, "public_id", post.PublicID)
	return nil
}

func (s *Seeder) loadExistingPostIDs(ctx context.Context) (map[string]struct{}, error) {
	existingIDs, err := s.DB.GetAllPostPublicIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSeedPost, err)
	}
	existing := make(map[string]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		existing[id] = struct{}{}
	}
	return existing, nil
}

func (s *Seeder) seedPostsForBlog(ctx context.Context, blogSlug string, existing map[string]struct{}) error {
	// remember blogslug will also be the blog directory name
	blogPath := filepath.Join(s.SourceDir, blogSlug)
	blogRoot, err := os.OpenRoot(blogPath)
	if err != nil {
		s.Logger.Error("could not open blog directory", "dir", blogSlug, "err", err)
		return err
	}

	if err := s.seedLoosePosts(ctx, blogSlug, blogPath, blogRoot); err != nil {
		return err
	}

	return s.seedStructuredPosts(ctx, blogSlug, blogPath, blogRoot, existing)
}

func (s *Seeder) seedLoosePosts(ctx context.Context, blogSlug, blogPath string, blogRoot *os.Root) error {
	wantedExt := "*.md"
	// there are the files in the root of the blog folder (NOT already in proper post subfolders )
	looseFiles, err := fs.Glob(blogRoot.FS(), wantedExt)
	if err != nil {
		return fmt.Errorf("could not glob for loose posts :%w", err)
	}

	for _, looseFile := range looseFiles {
		postPath := filepath.Join(blogPath, looseFile)
		// note the "" meaning a new public ID will be created
		if err := s.seedPost(ctx, blogSlug, postPath, ""); err != nil {
			s.Logger.Error("failed to seed loose post, skipping", "file", postPath, "err", err)
		}
	}
	return nil
}

func (s *Seeder) seedStructuredPosts(ctx context.Context, blogSlug, blogPath string, blogRoot *os.Root, existing map[string]struct{}) error {
	postDirs, err := fs.ReadDir(blogRoot.FS(), ".")
	if err != nil {
		return fmt.Errorf("could not read blog directory: %w", err)
	}
	for _, postDir := range postDirs {
		if !postDir.IsDir() {
			continue
		}

		// validate it looks like a post folder
		// folder name has a diffent length OR regex pattern doesn't match skip it
		if len(postDir.Name()) != storage.PublicIDLen || !validPublicID.MatchString(postDir.Name()) {
			continue
		}
		// already in DB
		if _, exists := existing[postDir.Name()]; exists {
			s.Logger.Info("post already exists, skipping", "public_id", postDir.Name())
			continue
		}
		postSubRoot, err := os.OpenRoot(filepath.Join(blogPath, postDir.Name()))
		if err != nil {
			continue
		}
		mdFiles, err := fs.Glob(postSubRoot.FS(), "*.md")
		// one post folder should only contain one md file
		if err != nil || len(mdFiles) != 1 {
			continue
		}

		postPath := filepath.Join(postSubRoot.Name(), mdFiles[0])
		if err := s.seedPost(ctx, blogSlug, postPath, postDir.Name()); err != nil {
			s.Logger.Error("failed to seed post, skipping", "file", postPath, "err", err)
		}
	}
	return nil
}
