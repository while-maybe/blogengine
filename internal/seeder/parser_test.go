package seeder

import (
	"blogengine/internal/storage"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

var blogManifests = map[string][]byte{
	"valid": []byte(`
title: "My Blog"
description: "A test blog"
owner: "testuser"
visibility: "public"
registration_mode: "open"
`),
	"missing_title": []byte(`
description: "A test blog"
owner: "testuser"
`),
	"missing_owner": []byte(`
title: "My Blog"
description: "A test blog"
`),
	"missing_visibility": []byte(`
title: "My Blog"
description: "A test blog"
owner: "testuser"
registration_mode: "open"
`),
	"missing_registration_mode": []byte(`
title: "My Blog"
description: "A test blog"
owner: "testuser"
`),
	"will_be_deleted": nil,
	"invalid_yaml": []byte(`
title: [invalid
`),
}

var postMarkdown = map[string][]byte{
	"valid": []byte(`---
title: "My Post"
description: "A test post"
slug: "my-post"
is_listed: true
published_at: "2026-01-20T00:00:00Z"
requires_auth: false
allow_comments: true
---

# My Post

This is the body of the post.
`),
	"missing_title": []byte(`---
description: "A test post"
slug: "my-post"
---

# My Post
`),
	"no_frontmatter": []byte(`# My Post

This is a post with no frontmatter at all.
`),
	"draft": []byte(`---
title: "My Draft Post"
description: "A draft post"
slug: "my-draft"
---

# My Draft Post

This is a draft.
`),
	"will_be_deleted": nil,
}

const defaultFileMode = 0644

func TestParseBlogManifest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                     string
		content                  []byte
		wantPublicVisibility     bool
		wantOpenRegistrationMode bool
		wantDeletedFile          bool
		wantErr                  error
	}{
		{
			name:    "nominal",
			content: blogManifests["valid"],
			wantErr: nil,
		},
		{
			name:    "missing title",
			content: blogManifests["missing_title"],
			wantErr: ErrNoBlogTitle,
		},
		{
			name:    "missing owner",
			content: blogManifests["missing_owner"],
			wantErr: ErrNoBlogOwner,
		},
		{
			name:                 "missing visibility",
			content:              blogManifests["missing_visibility"],
			wantPublicVisibility: true,
			wantErr:              nil,
		},
		{
			name:                     "missing registration default",
			content:                  blogManifests["missing_registration_mode"],
			wantOpenRegistrationMode: true,
			wantErr:                  nil,
		},
		{
			name:            "file does not exist",
			content:         blogManifests["will_be_deleted"],
			wantDeletedFile: true,
			wantErr:         ErrReadingFile,
		},
		{
			name:    "invalid yaml",
			content: blogManifests["invalid_yaml"],
			wantErr: ErrUnmarshallToManifest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "blog.yaml")
			if err := os.WriteFile(path, tt.content, defaultFileMode); err != nil {
				t.Fatalf("could not write files with data to import")
			}

			if tt.wantDeletedFile {
				os.Remove(path)
			}

			bm, err := ParseBlogManifest(path)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("errors: want %s, got %s", tt.wantErr, err)
			}
			if tt.wantErr != nil {
				return
			}

			if tt.wantPublicVisibility && bm.Visibility != storage.VisibilityPublic {
				t.Fatalf("visibility is not public: %s", bm.Visibility)
			}
			if tt.wantOpenRegistrationMode && bm.RegistrationMode != storage.RegistrationOpen {
				t.Fatalf("registration is not open: %s", bm.RegistrationMode)
			}
		})
	}
}

func TestParsePostFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		content         []byte
		wantDeletedFile bool
		wantDraft       bool
		wantErr         error
	}{
		{
			name:    "nominal",
			content: postMarkdown["valid"],
			wantErr: nil,
		},
		{
			name:    "missing title",
			content: postMarkdown["missing_title"],
			wantErr: ErrNoPostTitle,
		},
		{
			name:    "no_frontmatter",
			content: postMarkdown["no_frontmatter"],
			//  a post with no frontmatter has no title, so ErrNoPostTitle is the right error to return
			wantErr: ErrNoPostTitle,
		},
		{
			name:      "draft post",
			content:   postMarkdown["draft"],
			wantDraft: true,
			wantErr:   nil,
		},
		{
			name:            "deleted file",
			content:         postMarkdown["will_be_deleted"],
			wantDeletedFile: true,
			wantErr:         ErrReadingFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "post.md")
			if err := os.WriteFile(path, tt.content, defaultFileMode); err != nil {
				t.Fatalf("could not write files with data to import")
			}

			if tt.wantDeletedFile {
				os.Remove(path)
			}

			fm, body, err := ParsePostFile(path)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("errors: want %s, got %s", tt.wantErr, err)
			}
			if tt.wantErr != nil {
				return
			}

			if fm == nil {
				t.Fatal("expected frontmatter, got nil")
			}
			if len(body) == 0 {
				t.Fatal("expected body content, got empty")
			}
			if tt.wantDraft && fm.PublishedAt != nil {
				t.Fatal("expected nil published_at for draft")
			}

		})
	}
}
