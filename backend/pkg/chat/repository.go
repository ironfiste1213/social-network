package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

// done !
type Repository struct {
	db *sql.DB
}

// done !
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// done!
func (r *Repository) getUserBySessionID(ctx context.Context, sessionID string) (string, error) {
	var userID string
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id FROM sessions WHERE id = ? AND expires_at > CURRENT_TIMESTAMP;`,
		sessionID,
	).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("invalid session")
	}
	return userID, err
}

// done !
func (r *Repository) CanChatPrivate(ctx context.Context, senderID, receiverID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM followers
		WHERE (follower_id = ? AND following_id = ?)
		   OR (follower_id = ? AND following_id = ?);
	`, senderID, receiverID, receiverID, senderID).Scan(&count)
	
	return count > 0, err
}

// done !
func (r *Repository) IsGroupMember(ctx context.Context, groupID, userID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM group_members WHERE group_id = ? AND user_id = ?;
	`, groupID, userID).Scan(&count)
	return count > 0, err
}

// done !
func (r *Repository) GetGroupMemberIDs(ctx context.Context, groupID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id FROM group_members WHERE group_id = ?;`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// done!
func (r *Repository) SavePrivateMessage(ctx context.Context, senderID, receiverID, body string) (Message, error) {
	chatID, err := r.GetOrCreatePrivateChat(ctx, senderID, receiverID)
	if err != nil {
		return Message{}, err
	}
	return r.saveMessage(ctx, chatID, senderID, body)
}

// done !!
func (r *Repository) SaveGroupMessage(ctx context.Context, groupID, senderID, body string) (Message, error) {
	return r.saveMessage(ctx, groupID, senderID, body)
}

// done !
func (r *Repository) saveMessage(ctx context.Context, chatID, senderID, body string) (Message, error) {
	id := uuid.NewString()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO chat_messages (
			id,
			chat_id,
			sender_id,
			body
		)
		VALUES (?, ?, ?, ?);
	`, id, chatID, senderID, body)
	if err != nil {
		return Message{}, fmt.Errorf("save message: %w", err)
	}

	return r.getMessageByID(ctx, id)
}

// done !
func (r *Repository) getMessageByID(ctx context.Context, id string) (Message, error) {
	var m Message
	var sender UserInfo

	err := r.db.QueryRowContext(ctx, `
		SELECT
			m.id,
			m.chat_id,
			c.type,
			m.sender_id,
			m.body,
			m.created_at,
			u.id,
			u.first_name,
			u.last_name,
			COALESCE(u.nickname, ''),
			COALESCE(u.avatar_path, '')

		FROM chat_messages m

		JOIN chats c
			ON c.id = m.chat_id

		JOIN users u
			ON u.id = m.sender_id

		WHERE m.id = ?;
	`, id).Scan(
		&m.ID,
		&m.ChatID,
		&m.ChatType,
		&m.SenderID,
		&m.Body,
		&m.CreatedAt,
		&sender.ID,
		&sender.FirstName,
		&sender.LastName,
		&sender.Nickname,
		&sender.AvatarPath,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return Message{}, errors.New("message not found")
	}

	if err != nil {
		return Message{}, err
	}

	sender.ID = m.SenderID
	m.Sender = &sender

	return m, nil
}

// done !
func (r *Repository) GetHistory(ctx context.Context, chatID string, beforeMessageID string, limit int) ([]Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT
			m.id,
			m.chat_id,
			m.sender_id,
			m.body,
			m.created_at,
			u.id,
			u.first_name,
			u.last_name,
			COALESCE(u.nickname,''),
			COALESCE(u.avatar_path,'')
		FROM chat_messages m
		JOIN users u ON u.id = m.sender_id
		WHERE m.chat_id = ?
	`

	args := []any{chatID}

	if beforeMessageID != "" {
		query += `
			AND m.created_at < (
				SELECT created_at FROM chat_messages WHERE id = ?
			)
		`
		args = append(args, beforeMessageID)
	}

	query += `
		ORDER BY m.created_at DESC
		LIMIT ?
	`

	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message

	for rows.Next() {
		var m Message
		var sender UserInfo

		if err := rows.Scan(
			&m.ID,
			&m.ChatID,
			&m.SenderID,
			&m.Body,
			&m.CreatedAt,
			&sender.ID,
			&sender.FirstName,
			&sender.LastName,
			&sender.Nickname,
			&sender.AvatarPath,
		); err != nil {
			return nil, err
		}

		m.Sender = &sender
		msgs = append(msgs, m)
	}

	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	return msgs, rows.Err()
}

// done !
func (r *Repository) GetConversations(ctx context.Context, userID string) ([]Conversation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			c.id,
			c.type,
			m.body,
			m.created_at
		FROM chats c
		JOIN chat_participants p ON p.chat_id = c.id
		JOIN chat_messages m ON m.chat_id = c.id
		WHERE p.user_id = ?
		AND m.created_at = (
			SELECT MAX(created_at)
			FROM chat_messages
			WHERE chat_id = c.id
		)
		ORDER BY m.created_at DESC;
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convos []Conversation

	for rows.Next() {
		var (
			chatID   string
			chatType string
			lastMsg  string
			lastAt   time.Time
		)

		if err := rows.Scan(&chatID, &chatType, &lastMsg, &lastAt); err != nil {
			return nil, err
		}

		convos = append(convos, Conversation{
			ChatID:      chatID,
			ChatType:    ChatType(chatType),
			LastMessage: lastMsg,
			LastAt:      lastAt,
		})
	}

	return convos, rows.Err()
}

// done!
func (r *Repository) GetOrCreatePrivateChat(ctx context.Context, userA string, userB string) (string, error) {
	// sort users to make stable private_key
	ids := []string{userA, userB}
	sort.Strings(ids)
	privateKey := ids[0] + ":" + ids[1]

	// try existing chat
	var chatID string

	err := r.db.QueryRowContext(ctx, `SELECT id FROM chats WHERE type = 'private' AND private_key = ? LIMIT 1;`, privateKey).Scan(&chatID)

	if err == nil {
		return chatID, nil
	}

	if err != sql.ErrNoRows {
		return "", err
	}

	// create new chat
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	chatID = uuid.NewString()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO chats (id, type, private_key)
		VALUES (?, 'private', ?);
	`, chatID, privateKey)
	if err != nil {
		return "", err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO chat_participants (chat_id, user_id)
		VALUES (?, ?), (?, ?);
	`, chatID, userA, chatID, userB)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return chatID, nil
}
