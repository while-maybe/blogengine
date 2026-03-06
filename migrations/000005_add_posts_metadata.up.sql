CREATE TABLE IF NOT EXISTS posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    blog_id INTEGER NOT NULL,
    author_id INTEGER NOT NULL,

    public_id TEXT NOT NULL UNIQUE,
    slug TEXT,
    title TEXT NOT NULL,
    description TEXT,

    s3_key TEXT NOT NULL,

    is_encrypted BOOLEAN DEFAULT 0,
    encryption_iv TEXT, -- init vector for AES

    -- access control 
    requires_auth BOOLEAN DEFAULT 0, -- public by default
    is_listed BOOLEAN DEFAULT 1, -- shows in homepage
    allow_comments BOOLEAN DEFAULT 1,

    published_at DATETIME,

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT NULL,
    deleted_at DATETIME DEFAULT NULL,

    FOREIGN KEY (blog_id) REFERENCES blogs(id) ON DELETE CASCADE,
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE
);

-- faster lookups by either id type
-- public_id does not need a manually added index as unique constraint will create it

CREATE UNIQUE INDEX IF NOT EXISTS idx_posts_slug_active
ON posts(blog_id, slug) WHERE deleted_at IS NULL;

-- faster lookup for homepage query (so date + listed status)
CREATE INDEX IF NOT EXISTS idx_posts_feed ON posts(published_at, is_listed);

CREATE TRIGGER IF NOT EXISTS trg_posts_updated_at
AFTER UPDATE ON posts
FOR EACH ROW
BEGIN
    UPDATE posts SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;
