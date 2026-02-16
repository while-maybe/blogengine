DROP TRIGGER IF EXISTS trg_comments_updated_at;

PRAGMA foreign_keys=OFF;

CREATE TABLE IF NOT EXISTS comments_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id INTEGER NOT NULL,
    user_id INTEGER, -- can be null if user is deleted later comment can stay
    content TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME DEFAULT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,

    CHECK(LENGTH(content) >= 1 AND LENGTH(content) <= 10000),
    CHECK(id > 0)
);

INSERT INTO comments_new (id, post_id, user_id, content, created_at, deleted_at)
SELECT id, post_id, user_id, content, created_at, deleted_at FROM comments;

DROP TABLE comments;

ALTER TABLE comments_new RENAME TO comments;

CREATE INDEX IF NOT EXISTS idx_comments_post_id ON comments(post_id); -- comments on a post
CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments(user_id); -- comments for a user
CREATE INDEX IF NOT EXISTS idx_comments_created_at ON comments(created_at DESC); -- sorted by date

PRAGMA foreign_keys=ON;
