DROP INDEX IF EXISTS idx_users_nickname_nocase;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_nickname_unique_nocase
ON users(nickname COLLATE NOCASE)
WHERE nickname IS NOT NULL AND TRIM(nickname) != '';
