DROP TABLE IF EXISTS chat_messages;

CREATE TABLE chats (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL CHECK(type IN ('private', 'group')),
    private_key TEXT UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE chat_participants (
    chat_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (chat_id, user_id),

    FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE chat_messages (
    id TEXT PRIMARY KEY,
    chat_id TEXT NOT NULL,
    sender_id TEXT NOT NULL,
    body TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE,
    FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_chat_participants_user
ON chat_participants(user_id);

CREATE INDEX idx_chat_messages_chat
ON chat_messages(chat_id);

CREATE INDEX idx_chat_messages_created
ON chat_messages(created_at);