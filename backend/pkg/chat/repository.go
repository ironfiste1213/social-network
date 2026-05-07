package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

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

func (r *Repository) IsGroupMember(ctx context.Context, groupID, userID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM group_members WHERE group_id = ? AND user_id = ?;
	`, groupID, userID).Scan(&count)
	return count > 0, err
}

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

func (r *Repository) SavePrivateMessage(ctx context.Context, senderID, receiverID, body string) (Message, error) {
	chatID := r.privateChatID(senderID, receiverID)
	msg, err:= r.saveMessage(ctx, chatID, "private", senderID, body)
	if err != nil {
		return Message{}, err
	}
	msg.TargetID = receiverID
	return msg, nil
}

func (r *Repository) SaveGroupMessage(ctx context.Context, groupID, senderID, body string) (Message, error) {
	 msg, err :=r.saveMessage(ctx, groupID, "group", senderID, body)
	 if err != nil {
		 return Message{}, err
	 }
	 msg.TargetID = groupID
	 return msg, nil
}
func (r *Repository) saveMessage(ctx context.Context, chatID string,  senderID string, body string) (Message, error) {
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

func (r *Repository) GetHistory( ctx context.Context, chatID string, beforeMessageID string, limit int) ([]Message, error) {

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

func (r *Repository) GetConversations(ctx context.Context, userID string) ([]Conversation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT m.chat_id, m.chat_type, m.body, m.created_at
		FROM chat_messages m
		INNER JOIN (
			SELECT chat_id, MAX(created_at) AS latest
			FROM chat_messages
			WHERE sender_id = ?
			   OR (chat_type = 'private' AND chat_id LIKE '%' || ? || '%')
			GROUP BY chat_id
		) l ON m.chat_id = l.chat_id AND m.created_at = l.latest
		ORDER BY m.created_at DESC;
	`, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convos []Conversation
	for rows.Next() {
		var chatID, chatType, lastMsg string
		var lastAt interface{}
		if err := rows.Scan(&chatID, &chatType, &lastMsg, &lastAt); err != nil {
			return nil, err
		}

		c := Conversation{
			ChatType:    chatType,
			LastMessage: lastMsg,
		}
		if t, ok := lastAt.(string); ok {
			// parse time — SQLite returns strings
			_ = t
		}

		if chatType == "group" {
			var title string
			_ = r.db.QueryRowContext(ctx,
				`SELECT title FROM groups WHERE id = ?;`, chatID,
			).Scan(&title)
			// frontend uses group_id to fetch history — safe to expose
			c.GroupID = chatID
			c.GroupTitle = title
		} else {
			// derive the other user — chat_id stays internal
			otherID := otherUserFromChatID(chatID, userID)
			if otherID != "" {
				var p UserInfo
				_ = r.db.QueryRowContext(ctx, `
					SELECT id, first_name, last_name,
					       COALESCE(nickname,''), COALESCE(avatar_path,'')
					FROM users WHERE id = ?;
				`, otherID).Scan(&p.ID, &p.FirstName, &p.LastName, &p.Nickname, &p.AvatarPath)
				c.Participant = &p
			}
		}
		convos = append(convos, c)
	}
	return convos, rows.Err()
}




func (r *Repository) GetOrCreatePrivateChat(ctx context.Context, userA string, userB string) (string, error) {

	// sort users to make stable private_key
	privateKey := userA + ":" + userB
	if userB < userA {
		privateKey = userB + ":" + userA
	}

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
