package chat

import "time"

type ChatType string

const (
	ChatTypePrivate ChatType = "private"
	ChatTypeGroup   ChatType = "group"
)

//
// =========================
// DATABASE MODELS
// =========================
//

type Chat struct {
	ID         string     `json:"id"`
	Type       ChatType   `json:"type"`
	PrivateKey *string    `json:"private_key,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type ChatParticipant struct {
	ChatID   string    `json:"chat_id"`
	UserID   string    `json:"user_id"`
	JoinedAt time.Time `json:"joined_at"`
}

type ChatMessage struct {
	ID        string    `json:"id"`
	ChatID    string    `json:"chat_id"`
	SenderID  string    `json:"sender_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

//
// =========================
// SHARED / API MODELS
// =========================
//

type UserInfo struct {
	ID         string `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Nickname   string `json:"nickname,omitempty"`
	AvatarPath string `json:"avatar_path,omitempty"`
}

//
// Message returned to frontend/ws
//

type Message struct {
	ID        string     `json:"id"`
	ChatID    string     `json:"chat_id"`
	ChatType  ChatType   `json:"chat_type"`

	SenderID  string     `json:"sender_id"`
	Sender    *UserInfo  `json:"sender,omitempty"`

	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"created_at"`
}

//
// Conversation shown in sidebar
//

type Conversation struct {
	ChatID      string     `json:"chat_id"`
	ChatType    ChatType   `json:"chat_type"`

	// private chat
	Participant *UserInfo  `json:"participant,omitempty"`

	// group chat
	GroupTitle  string     `json:"group_title,omitempty"`

	LastMessage string     `json:"last_message"`
	LastAt      time.Time  `json:"last_at"`
}

//
// =========================
// WEBSOCKET EVENTS
// =========================
//

// client -> server
type InboundEvent struct {
	Type string `json:"type"` // send_private | send_group | ping

	// private => receiver user id
	// group   => group/chat id
	To   string `json:"to"`

	Body string `json:"body"`
}

// server -> client
type OutboundEvent struct {
	Type    string   `json:"type"` // message | error | pong
	Payload *Message `json:"payload,omitempty"`
	Error   string   `json:"error,omitempty"`
}