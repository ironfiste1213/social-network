package chat

import "time"

type Message struct {
	ID        string    `json:"id"`
	ChatType  string    `json:"chat_type"`
	SenderID  string    `json:"sender_id"`
	Sender    *UserInfo `json:"sender,omitempty"`
	Body      string    `json:"body"`
	TargetID  string    `json:"target_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type UserInfo struct {
	ID         string `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Nickname   string `json:"nickname,omitempty"`
	AvatarPath string `json:"avatar_path,omitempty"`
}

// Conversation is what the frontend sees in the sidebar.
// No chat_id — frontend uses receiver_id or group_id to fetch history.
type Conversation struct {
	ChatType    string    `json:"chat_type"`             // "private" | "group"
	Participant *UserInfo `json:"participant,omitempty"` // private: the other user
	GroupID     string    `json:"group_id,omitempty"`    // group: the group id
	GroupTitle  string    `json:"group_title,omitempty"` // group: the group title
	LastMessage string    `json:"last_message"`
	LastAt      time.Time `json:"last_at"`
}

// InboundEvent : client -> server
// Frontend never sends chat_id — only who to send to.
type InboundEvent struct {
	Type string `json:"type"` // "send_private" | "send_group" | "ping"
	To   string `json:"to"`   // receiverID (private) | groupID (group)
	Body string `json:"body"`
}

// OutboundEvent : server -> client
type OutboundEvent struct {
	Type    string   `json:"type"` // "message" | "error" | "pong"
	Payload *Message `json:"payload,omitempty"`
	Error   string   `json:"error,omitempty"`
}