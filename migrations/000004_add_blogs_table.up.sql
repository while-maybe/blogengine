CREATE TABLE IF NOT EXISTS blogs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id INTEGER NOT NULL, -- super admin of blog

    slug TEXT NOT NULL, -- say "tech", "travel"
    title TEXT NOT NULL,
    description TEXT,

    visibility TEXT NOT NULL DEFAULT 'public',

    registration_mode TEXT NOT NULL DEFAULT 'open',
    registration_limit INTEGER DEFAULT NULL,

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT NULL,
    deleted_at DATETIME DEFAULT NULL,

    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE,

    CHECK (id > 0),
    CHECK (owner_id > 0),
    CHECK (visibility IN ('public', 'private')),
    CHECK (registration_mode IN ('open', 'closed', 'limited', 'invite_only')),

    CHECK (
        (
            registration_mode = 'limited' 
            AND registration_limit IS NOT NULL 
            AND registration_limit > 0
        ) OR 
        (
            registration_mode != 'limited'
            AND registration_limit IS NULL
        )
    )
);

-- allows creating a blog with slug that was previously soft-deleted
CREATE UNIQUE INDEX IF NOT EXISTS idx_blogs_slug_active
ON blogs(slug) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_blogs_slug ON blogs(slug);
CREATE INDEX IF NOT EXISTS idx_blogs_created_at ON blogs(created_at);

CREATE TRIGGER IF NOT EXISTS trg_blogs_updated_at
AFTER UPDATE ON blogs
FOR EACH ROW
BEGIN
    UPDATE blogs SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;