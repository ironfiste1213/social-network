package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CanChatPrivate returns true if at least one user follows the other.
// Per spec: "at least one of the users must be following the other."
func (r *Repository) CanChatPrivate(ctx context.Context, senderID, receiverID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM followers
		WHERE (follower_id = ? AND following_id = ?)
		   OR (follower_id = ? AND following_id = ?);
	`, senderID, receiverID, receiverID, senderID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsGroupMember returns true if userID is a member of groupID.
func (r *Repository) IsGroupMember(ctx context.Context, groupID, userID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM group_members WHERE group_id = ? AND user_id = ?;
	`, groupID, userID).Scan(&count)
	return count > 0, err
}

// GetGroupMemberIDs returns all member user IDs for a group.
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

// ---- message persistence ----

func (r *Repository) SaveMessage(ctx context.Context, chatID, chatType, senderID, body string) (Message, error) {
	id := uuid.NewString()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO chat_messages (id, chat_id, chat_type, sender_id, body)
		VALUES (?, ?, ?, ?, ?);
	`, id, chatID, chatType, senderID, body)
	if err != nil {
		return Message{}, fmt.Errorf("save message: %w", err)
	}
	return r.GetMessageByID(ctx, id)
}

func (r *Repository) GetMessageByID(ctx context.Context, id string) (Message, error) {
	var m Message
	var sender UserInfo
	err := r.db.QueryRowContext(ctx, `
		SELECT m.id, m.chat_id, m.chat_type, m.sender_id, m.body, m.created_at,
		       u.first_name, u.last_name, COALESCE(u.nickname,''), COALESCE(u.avatar_path,'')
		FROM chat_messages m
		JOIN users u ON u.id = m.sender_id
		WHERE m.id = ?;
	`, id).Scan(
		&m.ID, &m.ChatID, &m.ChatType, &m.SenderID, &m.Body, &m.CreatedAt,
		&sender.FirstName, &sender.LastName, &sender.Nickname, &sender.AvatarPath,
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


// GetHistory returns the last `limit` messages for a chat, oldest-first.
func (r *Repository) GetHistory(ctx context.Context, chatID, beforeID  string, limit int) ([]Message, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	var (
		rows *sql.Rows
		err error
	)
	if beforeID == "" {
		rows, err = r.db.QueryContext(ctx, `
		SELECT m.id, m.chat_id, m.chat_type, m.sender_id, m.body, m.created_at,
		       u.first_name, u.last_name, COALESCE(u.nickname,''), COALESCE(u.avatar_path,'')
		FROM chat_messages m
		JOIN users u ON u.id = m.sender_id
		WHERE m.chat_id = ?
		ORDER BY m.created_at DESC
		LIMIT ?;
	`, chatID, limit)
	}else {
		rows, err = r.db.QueryContext(ctx, `
		SELECT m.id, m.chat_id, m.chat_type, m.sender_id, m.body, m.created_at,
		      u.first_name, u.last_name, COALESCE(u.nickname,''), COALESCE(u.avatar_path,'')
		FROM chat_messages m
		JOIN users u ON u.id = m.sender_id
		WHERE m.chat_id = ?
		AND m.created_at < (SELECT created_at FROM chat_messages WHERE id = ?)
		ORDER BY m.created_at DESC
		LIMIT ?;
		`, chatID, beforeID, limit)
	}
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()
 
	var msgs []Message
	for rows.Next() {
		var m Message
		var sender UserInfo
		if err := rows.Scan(
			&m.ID, &m.ChatID, &m.ChatType, &m.SenderID, &m.Body, &m.CreatedAt,
			&sender.FirstName, &sender.LastName, &sender.Nickname, &sender.AvatarPath,
		); err != nil {
			return nil, err
		}
		sender.ID = m.SenderID
		m.Sender = &sender
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
 
	// reverse to get oldest-first
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}