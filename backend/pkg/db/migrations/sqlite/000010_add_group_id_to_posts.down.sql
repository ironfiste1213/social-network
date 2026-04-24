DROP INDEX IF EXISTS idx_posts_group_id;

PRAGMA foreign_keys = OFF;

CREATE TABLE posts_backup (
    id TEXT PRIMARY KEY,
    author_id TEXT NOT NULL,
    body TEXT NOT NULL,
    image_path TEXT,
    privacy TEXT NOT NULL CHECK (privacy IN ('public', 'followers', 'selected_followers')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE
);

INSERT INTO posts_backup (id, author_id, body, image_path, privacy, created_at, updated_at)
SELECT id, author_id, body, image_path, privacy, created_at, updated_at
FROM posts;

DROP TABLE posts;

ALTER TABLE posts_backup RENAME TO posts;

CREATE INDEX IF NOT EXISTS idx_posts_author_id ON posts(author_id);

PRAGMA foreign_keys = ON;
