package chat

import "time"

// ---- DB model ----

type Message struct {
	ID        string    `json:"id"`
	ChatID    string    `json:"chat_id"`
	ChatType  string    `json:"chat_type"` // "private" | "group"
	SenderID  string    `json:"sender_id"`
	Sender    *UserInfo `json:"sender,omitempty"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type UserInfo struct {
	ID         string `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Nickname   string `json:"nickname,omitempty"`
	AvatarPath string `json:"avatar_path,omitempty"`
}

// Conversation shown in the sidebar list
type Conversation struct {
	ChatID      string    `json:"chat_id"`
	ChatType    string    `json:"chat_type"`
	Participant *UserInfo `json:"participant,omitempty"` // private only
	GroupID     string    `json:"group_id,omitempty"`    // group only
	GroupTitle  string    `json:"group_title,omitempty"` // group only
	LastMessage string    `json:"last_message"`
	LastAt      time.Time `json:"last_at"`
	UnreadCount int       `json:"unread_count"`
}

// ---- WebSocket wire format ----

// InboundEvent: client → server
type InboundEvent struct {
	Type string `json:"type"` // "send_private" | "send_group" | "ping"
	To   string `json:"to"`   // userID (private) or groupID (group)
	Body string `json:"body"`
}

// OutboundEvent: server → client
type OutboundEvent struct {
	Type    string   `json:"type"`    // "message" | "error" | "pong"
	Payload *Message `json:"payload,omitempty"`
	Error   string   `json:"error,omitempty"`
}