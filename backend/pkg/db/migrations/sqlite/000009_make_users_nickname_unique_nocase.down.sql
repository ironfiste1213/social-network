DROP INDEX IF EXISTS idx_users_nickname_unique_nocase;

CREATE INDEX IF NOT EXISTS idx_users_nickname_nocase
ON users(nickname COLLATE NOCASE);
