package events

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrForbidden          = errors.New("forbidden")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateEvent creates a new event in a group. Only members may create events.
func (s *Service) CreateEvent(ctx context.Context, groupID, creatorID string, input CreateEventInput) (GroupEvent, error) {
	if strings.TrimSpace(input.Title) == "" {
		return GroupEvent{}, ErrInvalidInput
	}
	if strings.TrimSpace(input.EventTime) == "" {
		return GroupEvent{}, ErrInvalidInput
	}

	_, isMember, err := s.repo.GetMembershipRole(ctx, groupID, creatorID)
	if err != nil {
		return GroupEvent{}, err
	}
	if !isMember {
		return GroupEvent{}, ErrForbidden
	}

	eventTime, err := time.Parse(time.RFC3339, input.EventTime)
	if err != nil {
		return GroupEvent{}, ErrInvalidInput
	}

	return s.repo.CreateEvent(ctx, groupID, creatorID, input, eventTime.UTC())
}

// ListEvents returns all events for a group. Only members may view events.
func (s *Service) ListEvents(ctx context.Context, groupID, viewerID string) ([]GroupEvent, error) {
	_, isMember, err := s.repo.GetMembershipRole(ctx, groupID, viewerID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrForbidden
	}

	return s.repo.ListEvents(ctx, groupID, viewerID)
}

// Respond sets or updates a member's RSVP for an event.
func (s *Service) Respond(ctx context.Context, groupID, eventID, userID, response string) (GroupEvent, error) {
	if response != "going" && response != "not_going" {
		return GroupEvent{}, ErrInvalidInput
	}

	_, isMember, err := s.repo.GetMembershipRole(ctx, groupID, userID)
	if err != nil {
		return GroupEvent{}, err
	}
	if !isMember {
		return GroupEvent{}, ErrForbidden
	}

	belongs, err := s.repo.EventBelongsToGroup(ctx, eventID, groupID)
	if err != nil {
		return GroupEvent{}, err
	}
	if !belongs {
		return GroupEvent{}, ErrNotFound
	}

	if err := s.repo.UpsertResponse(ctx, eventID, userID, response); err != nil {
		return GroupEvent{}, err
	}

	return s.repo.GetEventByID(ctx, eventID, userID)
}

func (s *Service) GetGroupMemberIDs(ctx context.Context, groupID string) ([]string, error) {
	return s.repo.GetGroupMemberIDs(ctx, groupID)
}
// CurrentUserID resolves a session ID to a user ID.
func (s *Service) CurrentUserID(ctx context.Context, sessionID string) (string, error) {
	if sessionID == "" {
		return "", ErrInvalidCredentials
	}
	return s.repo.GetUserBySessionID(ctx, sessionID)
}