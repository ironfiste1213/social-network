package chat

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// otherUserFromChatID extracts the other participant's ID from a private chat_id.

var (
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type Service struct {
	repo *Repository
	hub  *Hub
}

func NewService(repo *Repository, hub *Hub) *Service {
	return &Service{repo: repo, hub: hub}
}

// HandlePrivateMessage validates and delivers a private chat message.
func (s *Service) HandlePrivateMessage(c *Client, event InboundEvent) {
	ctx := context.Background()

	if strings.TrimSpace(event.Body) == "" || event.To == "" {
		c.send <- OutboundEvent{Type: "error", Error: "to and body are required"}
		return
	}
	if event.To == c.userID {
		c.send <- OutboundEvent{Type: "error", Error: "cannot message yourself"}
		return
	}

	allowed, err := s.repo.CanChatPrivate(ctx, c.userID, event.To)
	if err != nil {
		fmt.Printf("[CHAT][SERVICE] CanChatPrivate error: %v\n", err)
		c.send <- OutboundEvent{Type: "error", Error: "internal error"}
		return
	}
	if !allowed {
		c.send <- OutboundEvent{Type: "error", Error: "not allowed to message this user"}
		return
	}

	chatID := PrivateChatID(c.userID, event.To)
	msg, err := s.repo.SaveMessage(ctx, chatID, "private", c.userID, strings.TrimSpace(event.Body))
	if err != nil {
		fmt.Printf("[CHAT][SERVICE] SaveMessage error: %v\n", err)
		c.send <- OutboundEvent{Type: "error", Error: "failed to save message"}
		return
	}

	out := OutboundEvent{Type: "message", Payload: &msg}
	// deliver to both sender and recipient (sender sees their own message confirmed)
	s.hub.Send([]string{c.userID, event.To}, out)
}

func otherUserFromChatID(chatID, myID string) string {
	trimmed := strings.TrimPrefix(chatID, "private:")
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	if parts[0] == myID {
		return parts[1]
	}
	return parts[0]
}

// PrivateChatID creates a deterministic chat ID for two users.
// Format: "private:<lower_id>:<higher_id>"
func PrivateChatID(a, b string) string {
	ids := []string{a, b}
	sort.Strings(ids)
	return "private:" + ids[0] + ":" + ids[1]
}


// HandleGroupMessage validates and delivers a group chat message.
func (s *Service) HandleGroupMessage(c *Client, event InboundEvent) {
	ctx := context.Background()

	if strings.TrimSpace(event.Body) == "" || event.To == "" {
		c.send <- OutboundEvent{Type: "error", Error: "to and body are required"}
		return
	}

	isMember, err := s.repo.IsGroupMember(ctx, event.To, c.userID)
	if err != nil {
		fmt.Printf("[CHAT][SERVICE] IsGroupMember error: %v\n", err)
		c.send <- OutboundEvent{Type: "error", Error: "internal error"}
		return
	}
	if !isMember {
		c.send <- OutboundEvent{Type: "error", Error: "not a member of this group"}
		return
	}

	msg, err := s.repo.SaveMessage(ctx, event.To, "group", c.userID, strings.TrimSpace(event.Body))
	if err != nil {
		fmt.Printf("[CHAT][SERVICE] SaveMessage error: %v\n", err)
		c.send <- OutboundEvent{Type: "error", Error: "failed to save message"}
		return
	}

	memberIDs, err := s.repo.GetGroupMemberIDs(ctx, event.To)
	if err != nil {
		fmt.Printf("[CHAT][SERVICE] GetGroupMemberIDs error: %v\n", err)
		return
	}

	out := OutboundEvent{Type: "message", Payload: &msg}
	s.hub.Send(memberIDs, out)
}
