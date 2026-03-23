package sqlite

import (
	"blogengine/internal/storage"
	"context"
	"errors"
	"log/slog"
	"math"
	"strings"
	"testing"
	"time"
)

func setupTestBlog(t *testing.T) (*Store, *storage.User, *storage.Blog) {
	t.Helper()
	store := setupTestStore(t)

	ctx := context.Background()

	logger := slog.New(slog.DiscardHandler)
	if err := store.Bootstrap(ctx, logger); err != nil {
		t.Fatalf("could not bootstrap store: %s", err)
	}
	user, err := store.CreateUser(ctx, "test_user", gen60CharString())
	if err != nil {
		t.Fatalf("could not create user: %s", err)
	}

	createBlogParams := storage.CreateBlogParams{
		OwnerID:           user.ID,
		Slug:              "a-blog-slug",
		Title:             "blog title goes here",
		Description:       new("A blog to blog about tech things"),
		Visibility:        storage.VisibilityPublic,
		RegistrationMode:  storage.RegistrationOpen,
		RegistrationLimit: nil,
	}
	blog, err := store.CreateBlog(ctx, createBlogParams)
	if err != nil {
		t.Fatalf("could not create blog: %s", err)
	}
	return store, user, blog
}

func TestCreatePost(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		authorID          int64
		blogID            int64
		slug              *string
		wantDuplicateSlug bool
		isEncrypted       bool
		EncIV             *string
		wantErr           error
		publishedAt       *time.Time
	}{
		{
			name:    "nominal",
			slug:    new("a-post-slug"),
			wantErr: nil,
		},
		{
			name:     "invalid authorID",
			authorID: -5,
			wantErr:  ErrCreatingPost,
		},
		{
			name:    "invalid blogID",
			blogID:  -5,
			wantErr: ErrCreatingPost,
		},
		{
			name:    "missing blogID - not in DB",
			blogID:  math.MaxInt64,
			wantErr: ErrCreatingPost,
		},
		{
			name:              "existing (duplicate) slug",
			wantDuplicateSlug: true,
			slug:              new("a-duplicate-slug"),
			wantErr:           ErrCreatingPost,
		},
		{
			name:    "nil slug - creates uses publicID",
			slug:    nil,
			wantErr: nil,
		},
		{
			name:        "not enc with encIV",
			isEncrypted: false,
			EncIV:       new("something"),
			wantErr:     ErrCreatingPost,
		},
		{
			name:        "enc without encIV",
			isEncrypted: true,
			EncIV:       nil,
			wantErr:     ErrCreatingPost,
		},
		{
			name:        "valid nil publishedAt - means draft",
			publishedAt: nil,
			wantErr:     nil,
		},
		{
			name:        "valid publishedAt with datetime",
			publishedAt: new(time.Now().Add(1 * time.Hour)),
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			store, user, blog := setupTestBlog(t)

			blogID := blog.ID
			if tt.blogID != 0 {
				blogID = tt.blogID
			}

			authorID := user.ID
			if tt.authorID != 0 {
				authorID = tt.authorID
			}

			var err error
			if tt.wantDuplicateSlug {
				p := storage.CreatePostParams{
					BlogID:       blogID,
					AuthorID:     authorID,
					Slug:         tt.slug,
					Title:        "Title of the post",
					Description:  new("Description of the blog"),
					IsEncrypted:  tt.isEncrypted,
					EncryptionIV: tt.EncIV,
					IsListed:     true,
					PublishedAt:  new(time.Now().Add(1 * time.Hour)),
				}
				_, err = store.CreatePost(ctx, p)
				if err != nil {
					t.Fatalf("could not create first post for duplicate test: %s", err)
				}
			}

			p := storage.CreatePostParams{
				BlogID:       blogID,
				AuthorID:     authorID,
				Slug:         tt.slug,
				Title:        "Title of the post",
				Description:  new("Description of the post"),
				IsEncrypted:  tt.isEncrypted,
				EncryptionIV: tt.EncIV,
				IsListed:     true,
				PublishedAt:  new(time.Now().Add(1 * time.Hour)),
			}

			_, err = store.CreatePost(ctx, p)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("errors: want %s, got %s", tt.wantErr, err)
			}
		})
	}
}

func TestGetLatestPublicPosts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		blog2Private  bool
		postsUnlisted bool
		isDraft       bool
		limit         int64
		offset        int64
		wantLen       int
		wantErr       error
	}{
		{
			name:   "nominal",
			offset: 0, limit: 5,
			wantLen: 2,
			wantErr: nil,
		},
		{
			name:   "one listed post, two unlisted",
			offset: 0, limit: 5,
			postsUnlisted: true,
			wantLen:       1,
			wantErr:       nil,
		},
		{
			name:   "one listed post, two draft",
			offset: 0, limit: 5,
			isDraft: true,
			wantLen: 1,
			wantErr: nil,
		},
		{
			name:   "one listed post, other blog private",
			offset: 0, limit: 5,
			blog2Private: true,
			wantLen:      1,
			wantErr:      nil,
		},
		{
			name:   "bad offset",
			offset: -1, limit: 5,
			wantLen: 1,
			wantErr: ErrLatestPublicPosts,
		},
		{
			name:   "bad limit",
			offset: 0, limit: 0,
			wantLen: 1,
			wantErr: ErrLatestPublicPosts,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// create default setup
			ctx := context.Background()
			store, user1, blog1 := setupTestBlog(t)
			p := storage.CreatePostParams{
				BlogID:      blog1.ID,
				AuthorID:    user1.ID,
				Slug:        nil,
				Title:       "Title of the post",
				Description: new("Description of the post"),
				IsListed:    true,
				PublishedAt: new(time.Now().Add(-24 * time.Hour)),
			}

			var err error
			_, err = store.CreatePost(ctx, p)
			if err != nil {
				t.Fatalf("could not create first test post: %s", err)
			}

			// add another user
			user2, err := store.CreateUser(ctx, "test_user_2", gen60CharString())
			if err != nil {
				t.Fatalf("could not create second test user: %s", err)
			}

			blog2Visibility := storage.VisibilityPublic
			if tt.blog2Private {
				blog2Visibility = storage.VisibilityPrivate
			}

			// add one public blog for second user
			createBlogParams := storage.CreateBlogParams{
				OwnerID:          user2.ID,
				Slug:             "another-blog-slug",
				Title:            "blog title goes here",
				Description:      new("A blog to blog about tech things"),
				Visibility:       blog2Visibility,
				RegistrationMode: storage.RegistrationOpen,
			}
			blog2, err := store.CreateBlog(ctx, createBlogParams)
			if err != nil {
				t.Fatalf("could not create second test blog: %s", err)
			}

			published_at := new(time.Now().Add(-1 * time.Hour))
			if tt.isDraft {
				published_at = nil
			}
			postListed := true
			if tt.postsUnlisted {
				postListed = false
			}

			// add one public and one private posts
			p2 := storage.CreatePostParams{
				BlogID:      blog2.ID,
				AuthorID:    user2.ID,
				Slug:        nil,
				Title:       "Title of the post",
				Description: new("Description of the post"),
				IsListed:    postListed,
				PublishedAt: published_at,
			}
			_, err = store.CreatePost(ctx, p2)
			if err != nil {
				t.Fatalf("could not create first test post: %s", err)
			}

			// post 2 is never visible
			p3 := storage.CreatePostParams{
				BlogID:      blog2.ID,
				AuthorID:    user2.ID,
				Slug:        nil,
				Title:       "Title of the second post",
				Description: new("Description of the second post"),
				IsListed:    false,
				PublishedAt: published_at,
			}
			_, err = store.CreatePost(ctx, p3)
			if err != nil {
				t.Fatalf("could not create second test post: %s", err)
			}

			results, err := store.GetLatestPublicPosts(ctx, tt.offset, tt.limit)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("errors: want %s, got %s", tt.wantErr, err)
			}
			if tt.wantErr != nil {
				return
			}

			if len(results) != tt.wantLen {
				t.Fatalf("results mismatch: want %d, got %d", tt.wantLen, len(results))
			}
		})
	}
}

func TestValidatePostDetails(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		slug        *string
		title       string
		description *string
		wantErr     error
	}{
		{
			name:        "nominal",
			slug:        new(strings.Repeat("s", maxSlugLen)),
			title:       strings.Repeat("t", maxTitleLen),
			description: new(strings.Repeat("d", maxDescriptionLen)),
			wantErr:     nil,
		},
		{
			name:        "slug under min",
			slug:        new(strings.Repeat("s", minSlugLen-1)),
			title:       strings.Repeat("t", maxTitleLen),
			description: new(strings.Repeat("d", maxDescriptionLen)),
			wantErr:     ErrPostSlug,
		},
		{
			name:        "slug over max",
			slug:        new(strings.Repeat("s", maxSlugLen+1)),
			title:       strings.Repeat("t", maxTitleLen),
			description: new(strings.Repeat("d", maxDescriptionLen)),
			wantErr:     ErrPostSlug,
		},
		{
			name:        "title under min",
			slug:        new(strings.Repeat("s", maxSlugLen)),
			title:       strings.Repeat("t", minTitleLen-1),
			description: new(strings.Repeat("d", maxDescriptionLen)),
			wantErr:     ErrPostTitle,
		},
		{
			name:        "title over max",
			slug:        new(strings.Repeat("s", maxSlugLen)),
			title:       strings.Repeat("t", maxTitleLen+1),
			description: new(strings.Repeat("d", maxDescriptionLen)),
			wantErr:     ErrPostTitle,
		},
		{
			name:        "description over max",
			slug:        new(strings.Repeat("s", maxSlugLen)),
			title:       strings.Repeat("t", maxTitleLen),
			description: new(strings.Repeat("d", maxDescriptionLen+1)),
			wantErr:     ErrPostDescription,
		},
		{
			name:        "nil slug is valid",
			slug:        nil,
			title:       strings.Repeat("t", maxTitleLen),
			description: new(strings.Repeat("d", maxDescriptionLen)),
			wantErr:     nil,
		},
		{
			name:        "nil description is valid",
			slug:        new(strings.Repeat("s", maxSlugLen)),
			title:       strings.Repeat("t", maxTitleLen),
			description: nil,
			wantErr:     nil,
		},
		{
			name:        "nil slug and description is valid",
			slug:        nil,
			title:       strings.Repeat("t", maxTitleLen),
			description: nil,
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validatePostDetails(tt.slug, tt.title, tt.description)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("errors: want %s, got %s", tt.wantErr, err)
			}
		})
	}
}

func TestValidatePostSlug(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		slug    *string
		wantErr error
	}{
		{
			name:    "nominal",
			slug:    new(strings.Repeat("s", maxSlugLen)),
			wantErr: nil,
		},
		{
			name:    "less than minLen",
			slug:    new(strings.Repeat("s", minSlugLen-1)),
			wantErr: ErrPostSlug,
		},
		{
			name:    "more than maxLen",
			slug:    new(strings.Repeat("s", maxSlugLen+1)),
			wantErr: ErrPostSlug,
		},
		{
			name:    "invalid with Capitals",
			slug:    new("A-Slug-Is-A-Slug"),
			wantErr: ErrPostSlug,
		},
		{
			name:    "invalid with punctuation",
			slug:    new("slug-with-a-dot."),
			wantErr: ErrPostSlug,
		},
		{
			name:    "invalid underscore",
			slug:    new("slug_that_errors"),
			wantErr: ErrPostSlug,
		},
		{
			name:    "invalid leading hyphen",
			slug:    new("-should-fail-slug"),
			wantErr: ErrPostSlug,
		},
		{
			name:    "invalid trailing hyphen",
			slug:    new("should-fail-slug-"),
			wantErr: ErrPostSlug,
		},
		{
			name:    "nil slug is valid",
			slug:    nil,
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validatePostSlug(tt.slug)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("errors: got %s, want %s", err, tt.wantErr)
			}
		})
	}
}

func TestGenPostS3Key(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		newBlogSlug     string
		publicID        string
		isBadBlogID     bool
		isDeletedBlogID bool
		wantS3Key       string
		wantErr         error
	}{
		{
			name:        "nominal - hyphens in name",
			newBlogSlug: "a-test-blog",
			publicID:    "abc123def456",
			wantS3Key:   "a-test-blog/abc123def456",
			wantErr:     nil,
		},
		{
			name:        "nominal - single word in name",
			newBlogSlug: "something",
			publicID:    "abc123def456",
			wantS3Key:   "something/abc123def456",
			wantErr:     nil,
		},
		{
			name:        "non existent blog id",
			newBlogSlug: "something",
			publicID:    "abc123def456",
			isBadBlogID: true,
			wantS3Key:   "something/abc123def456",
			wantErr:     ErrGenerateS3Key,
		},
		{
			name:            "soft deleted blog id",
			newBlogSlug:     "something",
			publicID:        "abc123def456",
			isDeletedBlogID: true,
			wantS3Key:       "something/abc123def456",
			wantErr:         ErrGenerateS3Key,
		},
		{
			name:        "soft deleted blog id",
			newBlogSlug: "something",
			publicID:    "",
			wantS3Key:   "something/abc123def456",
			wantErr:     ErrInvalidPublicID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := setupTestStore(t)
			ctx := context.Background()

			logger := slog.New(slog.DiscardHandler)
			if err := store.Bootstrap(ctx, logger); err != nil {
				t.Fatalf("could not bootstrap store: %s", err)
			}

			u, err := store.CreateUser(ctx, "test_user", gen60CharString())
			if err != nil {
				t.Fatalf("could not create user: %s", err)
			}

			createBlogParams := storage.CreateBlogParams{
				OwnerID:           u.ID,
				Slug:              tt.newBlogSlug,
				Title:             "blog title goes here",
				Description:       new("A blog to blog about tech things"),
				Visibility:        storage.VisibilityPublic,
				RegistrationMode:  storage.RegistrationOpen,
				RegistrationLimit: nil,
			}
			blog, err := store.CreateBlog(ctx, createBlogParams)
			if err != nil {
				t.Fatalf("could not create blog: %s", err)
			}

			blogID := blog.ID
			if tt.isBadBlogID {
				blogID = math.MaxInt64
			}

			if tt.isDeletedBlogID {
				_ = store.DeleteBlog(ctx, blog.ID, blog.OwnerID)
			}

			gotS3Key, err := store.genPostS3Key(ctx, blogID, tt.publicID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("errors: want %s, got %s", tt.wantErr, err)
			}
			if tt.wantErr != nil {
				return
			}
			if tt.wantS3Key != gotS3Key {
				t.Fatalf("s3 key mismatch: want %s, got %s", tt.wantS3Key, gotS3Key)
			}
		})
	}
}

func TestValidatePostEncryptionSettings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		isEnc   bool
		encIV   *string
		wantErr error
	}{
		{
			name:    "nominal with enc",
			isEnc:   true,
			encIV:   new("something"),
			wantErr: nil,
		},
		{
			name:    "nominal no enc",
			isEnc:   false,
			encIV:   nil,
			wantErr: nil,
		},
		{
			name:    "enc - no IV",
			isEnc:   true,
			encIV:   nil,
			wantErr: ErrEncMissingIV,
		},
		{
			name:    "no enc - with IV",
			isEnc:   false,
			encIV:   new("something here"),
			wantErr: ErrEncPointlessIv,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validatePostEncryptionSettings(tt.isEnc, tt.encIV)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("errors: want %s, got %s", tt.wantErr, err)
			}
		})
	}
}
