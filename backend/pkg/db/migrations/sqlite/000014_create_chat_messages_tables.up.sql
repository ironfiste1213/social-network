CREATE TABLE IF NOT EXISTS chat_messages (
    id TEXT PRIMARY KEY,
    chat_id TEXT NOT NULL,              -- "private:<sorted_uid1>:<sorted_uid2>" OR group_id
    chat_type TEXT NOT NULL CHECK (chat_type IN ('private', 'group')),
    sender_id TEXT NOT NULL,
    body TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_chat
ON chat_messages(chat_id, created_at);

CREATE INDEX IF NOT EXISTS idx_chat_messages_sender
ON chat_messages(sender_id);