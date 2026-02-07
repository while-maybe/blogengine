CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE COLLATE NOCASE,
    password_hash TEXT NOT NULL,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME DEFAULT NULL,

    CHECK(LENGTH(username) >= 3 AND LENGTH(username) <= 50),
    CHECK(LENGTH(password_hash) = 60), -- bcrypt should return a len of 60
    CHECK(id > 0)
);

CREATE TABLE IF NOT EXISTS comments (
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

CREATE INDEX IF NOT EXISTS idx_comments_post_id ON comments(post_id); -- comments on a post
CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments(user_id); -- comments for a user
CREATE INDEX IF NOT EXISTS idx_comments_created_at ON comments(created_at DESC); -- sorted by date