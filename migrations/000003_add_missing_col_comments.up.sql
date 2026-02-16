PRAGMA foreign_keys=OFF;

CREATE TABLE IF NOT EXISTS comments_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id INTEGER NOT NULL,
    user_id INTEGER,
    content TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME DEFAULT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    CHECK(LENGTH(content) >= 1 AND LENGTH(content) <= 10000),
    CHECK(id > 0)
);

INSERT INTO comments_new (id, post_id, user_id, content, created_at, deleted_at)
SELECT id, post_id, user_id, content, created_at, deleted_at FROM comments;

DROP TABLE comments;
ALTER TABLE comments_new RENAME TO comments;

CREATE INDEX IF NOT EXISTS idx_comments_post_id ON comments(post_id);
CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments(user_id);
CREATE INDEX IF NOT EXISTS idx_comments_created_at ON comments(created_at DESC);

CREATE TRIGGER IF NOT EXISTS trg_comments_updated_at
AFTER UPDATE ON comments
FOR EACH ROW
BEGIN
    UPDATE comments SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;

PRAGMA foreign_keys=ON;