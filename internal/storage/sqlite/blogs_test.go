package sqlite

import (
	"blogengine/internal/storage"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestCreateBlog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		ownerID           int64
		slug              string
		title             string
		description       *string
		visibility        storage.Visibility
		registrationMode  storage.RegistrationMode
		registrationLimit *int64
		CreatedAt         time.Time
		UpdatedAt         *time.Time
		DeletedAt         *time.Time
		wantErr           error
		duplicate         bool
	}{
		{
			name: "nominal",
			slug: "technology", title: "a tech blog", description: new("A blog to blog about tech things"),
			visibility: storage.VisibilityPublic, registrationMode: storage.RegistrationOpen, registrationLimit: nil,
			wantErr: nil,
		},
		{
			name: "non existent owner",
			slug: "technology", title: "a tech blog", description: nil,
			visibility: storage.VisibilityPublic, registrationMode: storage.RegistrationOpen,
			ownerID: 99999,
			wantErr: ErrCreateBlog,
		},
		{
			name: "existing slug",
			slug: "technology", title: "a tech blog", description: nil,
			visibility: storage.VisibilityPublic, registrationMode: storage.RegistrationOpen,
			ownerID: 99999,
			wantErr: ErrCreateBlog,
		},
	}

	// cheat a little as created_at from sqlite registers up to second precision
	now := time.Now().Add(-1 * time.Second)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := setupTestStore(t)
			ctx := context.Background()

			// foreign key constraint needs a user first
			u, err := store.CreateUser(ctx, "admin", gen60CharString())
			if err != nil {
				t.Fatalf("could not create user: %s", err)
			}

			ownerID := u.ID
			if tt.ownerID != 0 {
				ownerID = tt.ownerID
			}

			createBlogParams := storage.CreateBlogParams{
				OwnerID:           ownerID,
				Slug:              tt.slug,
				Title:             tt.title,
				Description:       tt.description,
				Visibility:        tt.visibility,
				RegistrationMode:  tt.registrationMode,
				RegistrationLimit: tt.registrationLimit,
			}

			blog, err := store.CreateBlog(ctx, createBlogParams)

			if tt.duplicate {
				createBlogParams := storage.CreateBlogParams{
					OwnerID:           ownerID,
					Slug:              tt.slug,
					Title:             tt.title,
					Description:       tt.description,
					Visibility:        tt.visibility,
					RegistrationMode:  tt.registrationMode,
					RegistrationLimit: tt.registrationLimit,
				}

				_, err = store.CreateBlog(ctx, createBlogParams)
			}

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error differs: want %s, got %s", tt.wantErr, err)
			}

			if tt.wantErr != nil {
				return
			}

			if blog.CreatedAt.Before(now) {
				fmt.Printf("\n*****\n%s\n%s\n", blog.CreatedAt, time.Now())
				t.Fatalf("nonsense blog creation time: %s", blog.CreatedAt)
			}
		})
	}
}

func TestGetPublicBlogs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		slug              string
		title             string
		description       *string
		visibility        storage.Visibility
		registrationMode  storage.RegistrationMode
		registrationLimit *int64
		offset            int64
		limit             int64
		wantLen           int
		wantErr           error
	}{
		{
			name:   "nominal",
			offset: 0, limit: 10,
			registrationMode:  storage.RegistrationOpen,
			registrationLimit: nil,
			wantLen:           2,
			wantErr:           nil,
		},
		{
			name:   "invalid limit",
			offset: 0, limit: -5,
			registrationMode:  storage.RegistrationOpen,
			registrationLimit: nil,
			wantLen:           2,
			wantErr:           ErrLimitOffset,
		},
		{
			name:   "invalid offset",
			offset: -1, limit: 10,
			registrationMode:  storage.RegistrationOpen,
			registrationLimit: nil,
			wantLen:           2,
			wantErr:           ErrLimitOffset,
		},
		{
			name:   "invalid limit and offset",
			offset: -1, limit: -5,
			registrationMode:  storage.RegistrationOpen,
			registrationLimit: nil,
			wantLen:           2,
			wantErr:           ErrLimitOffset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := setupTestStore(t)
			ctx := context.Background()

			// foreign key constraint needs a user first
			u, err := store.CreateUser(ctx, "admin", gen60CharString())
			if err != nil {
				t.Fatalf("could not create user: %s", err)
			}

			// create 2 public, 1 private blogs
			createBlogParamsPub1 := storage.CreateBlogParams{
				OwnerID:           u.ID,
				Slug:              "technology",
				Title:             "a tech blog",
				Description:       new("A blog to blog about tech things"),
				Visibility:        storage.VisibilityPublic,
				RegistrationMode:  tt.registrationMode,
				RegistrationLimit: tt.registrationLimit,
			}

			_, _ = store.CreateBlog(ctx, createBlogParamsPub1)
			createBlogParamsPub2 := storage.CreateBlogParams{
				OwnerID:           u.ID,
				Slug:              "technology2",
				Title:             "a tech blog",
				Description:       new("A blog to blog about tech things"),
				Visibility:        storage.VisibilityPublic,
				RegistrationMode:  tt.registrationMode,
				RegistrationLimit: tt.registrationLimit,
			}

			_, _ = store.CreateBlog(ctx, createBlogParamsPub2)
			createBlogParamsPriv3 := storage.CreateBlogParams{
				OwnerID:           u.ID,
				Slug:              "technology3",
				Title:             "a tech blog",
				Description:       new("A blog to blog about tech things"),
				Visibility:        storage.VisibilityPrivate,
				RegistrationMode:  tt.registrationMode,
				RegistrationLimit: tt.registrationLimit,
			}
			_, _ = store.CreateBlog(ctx, createBlogParamsPriv3)

			blogs, err := store.GetPublicBlogs(ctx, tt.offset, tt.limit)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error: want %s, got %s", tt.wantErr, err)
			}
			if tt.wantErr != nil {
				return
			}
			if len(blogs) != tt.wantLen {
				t.Fatalf("number of public blogs: want %d, got %d", tt.wantLen, len(blogs))
			}
		})
	}
}

func TestGetBlogByID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		ownerID           int64
		slug              string
		title             string
		description       *string
		visibility        storage.Visibility
		registrationMode  storage.RegistrationMode
		registrationLimit *int64
		wantErr           error
		deleted           bool
	}{
		{
			name: "nominal",
			slug: "technology", title: "a tech blog", description: new("A blog to blog about tech things"),
			visibility: storage.VisibilityPublic, registrationMode: storage.RegistrationOpen, registrationLimit: nil,
			wantErr: nil,
		},
		{
			name: "private blog",
			slug: "technology", title: "a tech blog", description: new("A blog to blog about tech things"),
			visibility: storage.VisibilityPrivate, registrationMode: storage.RegistrationOpen, registrationLimit: nil,
			wantErr: nil,
		},
		{
			name: "get deleted blog",
			slug: "technology", title: "a tech blog", description: new("A blog to blog about tech things"),
			visibility: storage.VisibilityPrivate, registrationMode: storage.RegistrationOpen, registrationLimit: nil,
			deleted: true,
			wantErr: ErrGetBlogByID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := setupTestStore(t)
			ctx := context.Background()

			// foreign key constraint needs a user first
			u, err := store.CreateUser(ctx, "admin", gen60CharString())
			if err != nil {
				t.Fatalf("could not create user: %s", err)
			}

			createBlogParams := storage.CreateBlogParams{
				OwnerID:           u.ID,
				Slug:              tt.slug,
				Title:             tt.title,
				Description:       tt.description,
				Visibility:        tt.visibility,
				RegistrationMode:  tt.registrationMode,
				RegistrationLimit: tt.registrationLimit,
			}
			newBlog1, _ := store.CreateBlog(
				ctx, createBlogParams)

			// don't test delete in get
			if tt.deleted {
				_ = store.DeleteBlog(ctx, newBlog1.ID, u.ID)
			}

			blog, err := store.GetBlogByID(ctx, newBlog1.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error: want %s, got %s", tt.wantErr, err)
			}
			if tt.wantErr != nil {
				return
			}

			if newBlog1.ID != blog.ID {
				t.Fatalf("blog ID: create %d, get %d", newBlog1.ID, blog.ID)
			}
		})
	}
}

func TestGetBlogBySlug(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		ownerID           int64
		slug              string
		title             string
		description       *string
		visibility        storage.Visibility
		registrationMode  storage.RegistrationMode
		registrationLimit *int64
		wantErr           error
		deleted           bool
	}{
		{
			name: "nominal",
			slug: "technology", title: "a tech blog", description: new("A blog to blog about tech things"),
			visibility: storage.VisibilityPublic, registrationMode: storage.RegistrationOpen, registrationLimit: nil,
			wantErr: nil,
		},
		{
			name: "private blog",
			slug: "technology", title: "a tech blog", description: new("A blog to blog about tech things"),
			visibility: storage.VisibilityPrivate, registrationMode: storage.RegistrationOpen, registrationLimit: nil,
			wantErr: nil,
		},
		{
			name: "get deleted blog",
			slug: "technology", title: "a tech blog", description: new("A blog to blog about tech things"),
			visibility: storage.VisibilityPrivate, registrationMode: storage.RegistrationOpen, registrationLimit: nil,
			deleted: true,
			wantErr: ErrGetBlogBySlug,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := setupTestStore(t)
			ctx := context.Background()

			// foreign key constraint needs a user first
			u, err := store.CreateUser(ctx, "admin", gen60CharString())
			if err != nil {
				t.Fatalf("could not create user: %s", err)
			}

			createBlogParams := storage.CreateBlogParams{
				OwnerID:           u.ID,
				Slug:              tt.slug,
				Title:             tt.title,
				Description:       tt.description,
				Visibility:        tt.visibility,
				RegistrationMode:  tt.registrationMode,
				RegistrationLimit: tt.registrationLimit,
			}
			newBlog1, _ := store.CreateBlog(ctx, createBlogParams)

			// don't test delete in get
			if tt.deleted {
				_ = store.DeleteBlog(ctx, newBlog1.ID, u.ID)
			}

			blog, err := store.GetBlogBySlug(ctx, newBlog1.Slug)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error: want %s, got %s", tt.wantErr, err)
			}
			if tt.wantErr != nil {
				return
			}

			if newBlog1.Slug != blog.Slug {
				t.Fatalf("blog slug: create %s, get %s", newBlog1.Slug, blog.Slug)
			}
		})
	}
}

func TestUpdateBlog(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		ownerID           int64
		slug              string
		title             string
		description       *string
		visibility        storage.Visibility
		registrationMode  storage.RegistrationMode
		registrationLimit *int64
		wantErr           error
		deleted           bool
	}{
		{
			name: "nominal",
			slug: "updated-slug", title: "Updated Title", description: new("Updated Description! blog to blog about tech things"),
			visibility: storage.VisibilityPublic, registrationMode: storage.RegistrationOpen, registrationLimit: nil,
			wantErr: nil,
		},
		{
			name: "deleted blog",
			slug: "updated-slug", title: "Updated Title", description: new("Updated Description! blog to blog about tech things"),
			visibility: storage.VisibilityPublic, registrationMode: storage.RegistrationOpen, registrationLimit: nil,
			wantErr: ErrUpdateBlog,
			deleted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := setupTestStore(t)
			ctx := context.Background()

			// foreign key constraint needs a user first
			u, err := store.CreateUser(ctx, "admin", gen60CharString())
			if err != nil {
				t.Fatalf("could not create user: %s", err)
			}

			createBlogParams := storage.CreateBlogParams{
				OwnerID:           u.ID,
				Slug:              "technology",
				Title:             "a tech blog",
				Description:       new("A blog to blog about tech things"),
				Visibility:        storage.VisibilityPublic,
				RegistrationMode:  tt.registrationMode,
				RegistrationLimit: tt.registrationLimit,
			}
			b1, _ := store.CreateBlog(ctx, createBlogParams)

			// don't test delete
			if tt.deleted {
				_ = store.DeleteBlog(ctx, b1.ID, u.ID)
			}

			updateBlogParams := storage.UpdateBlogParams{
				BlogID:      b1.ID,
				OwnerID:     b1.OwnerID,
				Slug:        tt.slug,
				Title:       tt.title,
				Description: tt.description,
			}
			blog, err := store.UpdateBlog(ctx, updateBlogParams)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("errors: want %s, got %s", tt.wantErr, err)
			}
			if tt.wantErr != nil {
				return
			}
			if blog.Slug != tt.slug {
				t.Fatalf("slug: want %s, got %s", tt.slug, blog.Slug)
			}
			if blog.Title != tt.title {
				t.Fatalf("title: want %s, got %s", tt.title, blog.Title)
			}
			if blog.Description != nil && tt.description != nil {
				if *blog.Description != *tt.description {
					t.Fatalf("description: want %v, got %v", *tt.description, *blog.Description)
				}
			}
		})
	}
}

func TestUpdateBlogVisibility(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		ownerID           int64
		blogID            int64
		slug              string
		title             string
		description       *string
		visibility        storage.Visibility
		registrationMode  storage.RegistrationMode
		registrationLimit *int64
		wantErr           error
		deleted           bool
	}{
		{
			name:       "nominal - to private",
			visibility: storage.VisibilityPrivate,
			wantErr:    nil,
		},
		{
			name:       "nominal - to public",
			visibility: storage.VisibilityPublic,
			wantErr:    nil,
		},
		{
			name:       "deleted blog",
			visibility: storage.VisibilityPublic,
			wantErr:    storage.ErrNotFound,
			deleted:    true,
		},
		{
			name:       "negative owner ID",
			visibility: storage.VisibilityPublic,
			ownerID:    -5,
			wantErr:    ErrNegativeIDs,
		},
		{
			name:       "negative blog ID",
			visibility: storage.VisibilityPublic,
			blogID:     -5,
			wantErr:    ErrNegativeIDs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := setupTestStore(t)
			ctx := context.Background()

			// foreign key constraint needs a user first
			u, err := store.CreateUser(ctx, "admin", gen60CharString())
			if err != nil {
				t.Fatalf("could not create user: %s", err)
			}

			createBlogParams := storage.CreateBlogParams{
				OwnerID:           u.ID,
				Slug:              "technology",
				Title:             "a tech blog",
				Description:       new("A blog to blog about tech things"),
				Visibility:        storage.VisibilityPublic,
				RegistrationMode:  storage.RegistrationOpen,
				RegistrationLimit: tt.registrationLimit,
			}
			blog, err := store.CreateBlog(ctx, createBlogParams)
			if err != nil {
				t.Fatalf("could not create blog: %s", err)
			}

			// don't test delete
			if tt.deleted {
				_ = store.DeleteBlog(ctx, blog.ID, u.ID)
			}

			// check if manual owner ID is part of test case
			ownerID := blog.OwnerID
			if tt.ownerID != 0 {
				ownerID = tt.ownerID
			}

			// check if manual blogID is part of test case
			blogID := blog.ID
			if tt.blogID != 0 {
				blogID = tt.blogID
			}

			err = store.UpdateBlogVisibility(ctx, blogID, ownerID, tt.visibility)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("errors: want %s, got %s", tt.wantErr, err)
			}
			if tt.wantErr != nil {
				return
			}

			// don't test get
			updated, _ := store.GetBlogByID(ctx, blog.ID)

			if updated.Visibility != tt.visibility {
				t.Fatalf("visibility: want %s, got %s", tt.visibility, blog.Visibility)
			}
		})
	}
}

func TestUpdateBlogRegistration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		ownerID           int64
		blogID            int64
		slug              string
		title             string
		description       *string
		visibility        storage.Visibility
		registrationMode  storage.RegistrationMode
		registrationLimit *int64
		wantErr           error
		deleted           bool
	}{
		{
			name:             "nominal - open",
			registrationMode: storage.RegistrationOpen,
			wantErr:          nil,
		},
		{
			name:             "nominal - closed",
			registrationMode: storage.RegistrationClosed,
			wantErr:          nil,
		},
		{
			name:              "nominal - limited",
			registrationMode:  storage.RegistrationLimited,
			registrationLimit: new(int64(3)),
			wantErr:           nil,
		},
		{
			name:             "nominal - invite only",
			registrationMode: storage.RegistrationInviteOnly,
			wantErr:          nil,
		},
		{
			name:             "negative owner ID",
			registrationMode: storage.RegistrationOpen,
			ownerID:          -5,
			wantErr:          ErrNegativeIDs,
		},
		{
			name:             "negative blog ID",
			registrationMode: storage.RegistrationOpen,
			blogID:           -5,
			wantErr:          ErrNegativeIDs,
		},
		{
			name:             "deleted blog",
			registrationMode: storage.RegistrationOpen,
			wantErr:          storage.ErrNotFound,
			deleted:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := setupTestStore(t)
			ctx := context.Background()

			// foreign key constraint needs a user first
			u, err := store.CreateUser(ctx, "admin", gen60CharString())
			if err != nil {
				t.Fatalf("could not create user: %s", err)
			}

			createBlogParams := storage.CreateBlogParams{
				OwnerID:           u.ID,
				Slug:              "technology",
				Title:             "a tech blog",
				Description:       new("A blog to blog about tech things"),
				Visibility:        storage.VisibilityPublic,
				RegistrationMode:  storage.RegistrationOpen,
				RegistrationLimit: nil,
			}

			blog, err := store.CreateBlog(ctx, createBlogParams)
			if err != nil {
				t.Fatalf("could not create blog: %s", err)
			}

			// don't test delete
			if tt.deleted {
				_ = store.DeleteBlog(ctx, blog.ID, u.ID)
			}

			// check if manual owner ID is part of test case
			ownerID := blog.OwnerID
			if tt.ownerID != 0 {
				ownerID = tt.ownerID
			}

			// check if manual blogID is part of test case
			blogID := blog.ID
			if tt.blogID != 0 {
				blogID = tt.blogID
			}

			updateBlogRegParams := storage.UpdateBlogRegistrationParams{
				BlogID:            blogID,
				OwnerID:           ownerID,
				RegistrationMode:  tt.registrationMode,
				RegistrationLimit: tt.registrationLimit,
			}
			err = store.UpdateBlogRegistration(ctx, updateBlogRegParams)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("errors: want %s, got %s", tt.wantErr, err)
			}
			if tt.wantErr != nil {
				return
			}

			// don't test get
			updated, _ := store.GetBlogByID(ctx, blog.ID)

			if updated.RegistrationMode != tt.registrationMode {
				t.Fatalf("registration mode: want %s, got %s", tt.registrationMode, blog.RegistrationMode)
			}
		})
	}
}

func TestDeleteBlog(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		ownerID           int64
		blogID            int64
		slug              string
		title             string
		description       *string
		visibility        storage.Visibility
		registrationMode  storage.RegistrationMode
		registrationLimit *int64
		wantErr           error
		deleted           bool
	}{
		{
			name:    "nominal",
			wantErr: nil,
		},
		{
			name:    "non existent blog",
			wantErr: storage.ErrNotFound,
			deleted: true,
		},
		{
			name:    "non existent owner",
			ownerID: 9999,
			wantErr: storage.ErrNotFound,
		},
		{
			name:    "negative blog id",
			wantErr: ErrNegativeIDs,
			blogID:  -5,
		},
		{
			name:    "negative owner id",
			wantErr: ErrNegativeIDs,
			ownerID: -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := setupTestStore(t)
			ctx := context.Background()

			// foreign key constraint needs a user first
			u, err := store.CreateUser(ctx, "admin", gen60CharString())
			if err != nil {
				t.Fatalf("could not create user: %s", err)
			}

			createBlogParams := storage.CreateBlogParams{
				OwnerID:           u.ID,
				Slug:              "technology",
				Title:             "a tech blog",
				Description:       new("A blog to blog about tech things"),
				Visibility:        storage.VisibilityPublic,
				RegistrationMode:  storage.RegistrationOpen,
				RegistrationLimit: nil,
			}
			blog, err := store.CreateBlog(ctx, createBlogParams)
			if err != nil {
				t.Fatalf("could not create blog: %s", err)
			}

			// don't test delete
			if tt.deleted {
				_ = store.DeleteBlog(ctx, blog.ID, u.ID)
			}

			// check if manual owner ID is part of test case
			ownerID := blog.OwnerID
			if tt.ownerID != 0 {
				ownerID = tt.ownerID
			}

			// check if manual blogID is part of test case
			blogID := blog.ID
			if tt.blogID != 0 {
				blogID = tt.blogID
			}

			err = store.DeleteBlog(ctx, blogID, ownerID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("errors: want %s, got %s", tt.wantErr, err)
			}
			if tt.wantErr != nil {
				return
			}
			if _, err := store.GetBlogByID(ctx, blogID); err == nil {
				t.Fatalf("can access deleted blog: %d", blogID)
			}
		})
	}
}

func TestValidateBlogSlug(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		slug    string
		wantErr error
	}{
		{
			name:    "nominal",
			slug:    "something-valid",
			wantErr: nil,
		},
		{
			name:    "less than minLen",
			slug:    "abc",
			wantErr: ErrBlogSlug,
		},
		{
			name:    "more than maxLen",
			slug:    "this-is-a-very-very-long-slug-which-should-cause-a-slug-error-better-work-or-i-will-have-to-keep-adding-many-more-characters",
			wantErr: ErrBlogSlug,
		},
		{
			name:    "invalid with Capitals",
			slug:    "A-Slug-Is-A-Slug",
			wantErr: ErrBlogSlug,
		},
		{
			name:    "invalid with punctuation",
			slug:    "slug-with-a-dot.",
			wantErr: ErrBlogSlug,
		},
		{
			name:    "invalid underscore",
			slug:    "slug_that_errors",
			wantErr: ErrBlogSlug,
		},
		{
			name:    "invalid leading hyphen",
			slug:    "-should-fail-slug",
			wantErr: ErrBlogSlug,
		},
		{
			name:    "invalid trailing hyphen",
			slug:    "should-fail-slug-",
			wantErr: ErrBlogSlug,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateBlogSlug(tt.slug)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("errors: got %s, want %s", err, tt.wantErr)
			}
		})
	}
}

func TestValidateBlog(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		slug        string
		title       string
		description *string
		wantErr     error
	}{
		{
			name:        "nominal",
			slug:        strings.Repeat("s", maxSlugLen),
			title:       strings.Repeat("t", maxTitleLen),
			description: new(strings.Repeat("d", maxDescriptionLen)),
		},
		// validate slug has been tested in detail before, just to increase cover
		{
			name:        "invalid slug",
			slug:        strings.Repeat("s", maxSlugLen+1),
			title:       strings.Repeat("t", maxTitleLen),
			description: new(strings.Repeat("d", maxDescriptionLen)),
			wantErr:     ErrBlogSlug,
		},
		{
			name:        "title less than min len",
			slug:        strings.Repeat("s", maxSlugLen),
			title:       strings.Repeat("t", minTitleLen-1),
			description: new(strings.Repeat("d", maxDescriptionLen)),
			wantErr:     ErrBlogTitle,
		},
		{
			name:        "title more than max len",
			slug:        strings.Repeat("s", maxSlugLen),
			title:       strings.Repeat("t", maxTitleLen+1),
			description: new(strings.Repeat("d", maxDescriptionLen)),
			wantErr:     ErrBlogTitle,
		},
		{
			name:        "description more than max len",
			slug:        strings.Repeat("s", maxSlugLen),
			title:       strings.Repeat("t", maxTitleLen),
			description: new(strings.Repeat("d", maxDescriptionLen+1)),
			wantErr:     ErrBlogDescription,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateBlog(tt.slug, tt.title, tt.description)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("errors: got %s, want %s", err, tt.wantErr)
			}
		})
	}
}

func TestValidateBlogVisibility(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		visibility storage.Visibility
		wantErr    error
	}{
		{
			name:       "nominal - public",
			visibility: storage.VisibilityPublic,
		},
		{
			name:       "nominal - private",
			visibility: storage.VisibilityPrivate,
		},
		{
			name:       "empty visibility",
			visibility: "",
			wantErr:    ErrBlogVisibility,
		},
		{
			name:       "invalid visibility",
			visibility: "invalid",
			wantErr:    ErrBlogVisibility,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateBlogVisibility(tt.visibility)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("errors: got %s, want %s", err, tt.wantErr)
			}
		})
	}
}

func TestValidateBlogRegistration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		registrationMode  storage.RegistrationMode
		registrationLimit *int64
		wantErr           error
	}{
		{
			name:             "nominal - open",
			registrationMode: storage.RegistrationOpen,
		},
		{
			name:             "nominal - closed",
			registrationMode: storage.RegistrationClosed,
		},
		{
			name:             "nominal - invite only",
			registrationMode: storage.RegistrationInviteOnly,
		},
		{
			name:              "nominal - with valid limit",
			registrationMode:  storage.RegistrationLimited,
			registrationLimit: new(int64(5)),
		},
		{
			name:              "registration mode not compatible with limit",
			registrationMode:  storage.RegistrationOpen,
			registrationLimit: new(int64(5)),
			wantErr:           ErrRegistrationValuesForMode,
		},
		{
			name:             "limited registration - missing limit",
			registrationMode: storage.RegistrationLimited,
			wantErr:          ErrBlogRegistrationLimit,
		},
		{
			name:              "limited registration - below min value",
			registrationMode:  storage.RegistrationLimited,
			registrationLimit: new(int64(0)),
			wantErr:           ErrBlogRegistrationLimit,
		},
		{
			name:              "limited registration - exceeds max value",
			registrationMode:  storage.RegistrationLimited,
			registrationLimit: new(int64(maxRegistrationQueue + 1)),
			wantErr:           ErrBlogRegistrationLimit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateBlogRegistration(tt.registrationMode, tt.registrationLimit)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("errors: want %s, got %s", tt.wantErr, err)
			}
		})
	}
}
