package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type Service struct {
	repo *Repository
	hub  *Hub
}

func NewService(repo *Repository, hub *Hub) *Service {
	return &Service{repo: repo, hub: hub}
}

func (s *Service) CurrentUserID(ctx context.Context, sessionID string) (string, error) {
	if sessionID == "" {
		return "", ErrInvalidCredentials
	}
	return s.repo.getUserBySessionID(ctx, sessionID)
}

// HandlePrivateMessage — senderID from auth, chat_id generated in repo.
func (s *Service) HandlePrivateMessage(c *Client, event InboundEvent) {

	if strings.TrimSpace(event.Body) == "" || strings.TrimSpace(event.To) == "" {
		c.send <- OutboundEvent{Type: "error", Error: "to and body are required"}
		return
	}
	if event.To == c.userID {
		c.send <- OutboundEvent{Type: "error", Error: "cannot message yourself"}
		return
	}

	allowed, err := s.repo.CanChatPrivate(c.ctx, c.userID, event.To)
	if err != nil {
		fmt.Printf("[CHAT][SERVICE] CanChatPrivate error: %v\n", err)
		c.send <- OutboundEvent{Type: "error", Error: "internal error"}
		return
	}
	if !allowed {
		c.send <- OutboundEvent{Type: "error", Error: "not allowed to message this user"}
		return
	}

	msg, err := s.repo.SavePrivateMessage(c.ctx, c.userID, event.To, strings.TrimSpace(event.Body))
	if err != nil {
		fmt.Printf("[CHAT][SERVICE] SavePrivateMessage error: %v\n", err)
		c.send <- OutboundEvent{Type: "error", Error: "failed to save message"}
		return
	}

	s.hub.Send([]string{c.userID, event.To}, OutboundEvent{Type: "message", Payload: &msg})
}

// HandleGroupMessage — senderID from auth, chat_id = groupID assigned in repo.
func (s *Service) HandleGroupMessage(c *Client, event InboundEvent) {

	if strings.TrimSpace(event.Body) == "" || strings.TrimSpace(event.To) == "" {
		c.send <- OutboundEvent{Type: "error", Error: "to and body are required"}
		return
	}

	isMember, err := s.repo.IsGroupMember(c.ctx, event.To, c.userID)
	if err != nil {
		fmt.Printf("[CHAT][SERVICE] IsGroupMember error: %v\n", err)
		c.send <- OutboundEvent{Type: "error", Error: "internal error"}
		return
	}
	if !isMember {
		c.send <- OutboundEvent{Type: "error", Error: "not a member of this group"}
		return
	}

	msg, err := s.repo.SaveGroupMessage(c.ctx, event.To, c.userID, strings.TrimSpace(event.Body))
	if err != nil {
		fmt.Printf("[CHAT][SERVICE] SaveGroupMessage error: %v\n", err)
		c.send <- OutboundEvent{Type: "error", Error: "failed to save message"}
		return
	}

	memberIDs, err := s.repo.GetGroupMemberIDs(c.ctx, event.To)
	if err != nil {
		fmt.Printf("[CHAT][SERVICE] GetGroupMemberIDs error: %v\n", err)
		return
	}

	s.hub.Send(memberIDs, OutboundEvent{Type: "message", Payload: &msg})
}

// GetPrivateHistory — backend derives chat_id from senderID+receiverID.
// Frontend only knows receiver_id.
func (s *Service) GetPrivateHistory(ctx context.Context, senderID, receiverID, beforeID string, limit int) ([]Message, error) {
	allowed, err := s.repo.CanChatPrivate(ctx, senderID, receiverID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	chatID, exists, err := s.repo.GetPrivateChatID(ctx, senderID, receiverID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrForbidden
	}
	return s.repo.GetHistory(ctx, chatID, beforeID, limit)
}

// GetGroupHistory — backend uses groupID as chat_id.
// Frontend only knows group_id.
func (s *Service) GetGroupHistory(ctx context.Context, groupID, requesterID, beforeID string, limit int) ([]Message, error) {
	isMember, err := s.repo.IsGroupMember(ctx, groupID, requesterID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrForbidden
	}
	return s.repo.GetHistory(ctx, groupID, beforeID, limit)
}

// GetConversations returns all conversations the user participated in.
func (s *Service) GetConversations(ctx context.Context, userID string) ([]Conversation, error) {
	return s.repo.GetConversations(ctx, userID)
}