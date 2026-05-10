package groups

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrForbidden          = errors.New("forbidden")
	ErrAlreadyMember      = errors.New("already member")
	ErrPendingExists      = errors.New("pending item already exists")
	ErrUserNotFound       = errors.New("user not found")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateGroup(ctx context.Context, creatorID string, input CreateGroupInput) (Group, error) {
	if strings.TrimSpace(input.Title) == "" {
		return Group{}, ErrInvalidInput
	}
	return s.repo.CreateGroup(ctx, creatorID, input)
}

func (s *Service) ListGroups(ctx context.Context, viewerID string, limit int, beforeID string) ([]Group, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.repo.ListGroups(ctx, viewerID, limit, beforeID)
}

func (s *Service) GetGroup(ctx context.Context, groupID, viewerID string) (Group, error) {
	return s.repo.GetGroupByID(ctx, groupID, viewerID)
}

func (s *Service) RequestJoin(ctx context.Context, groupID, userID string) (GroupJoinRequest, error) {
	if _, err := s.repo.GetGroupByID(ctx, groupID, userID); err != nil {
		return GroupJoinRequest{}, err
	}

	if _, isMember, err := s.repo.GetMembershipRole(ctx, groupID, userID); err != nil {
		return GroupJoinRequest{}, err
	} else if isMember {
		return GroupJoinRequest{}, ErrAlreadyMember
	}

	if _, exists, err := s.repo.GetPendingJoinRequest(ctx, groupID, userID); err != nil {
		return GroupJoinRequest{}, err
	} else if exists {
		return GroupJoinRequest{}, ErrPendingExists
	}

	if _, exists, err := s.repo.GetPendingInvitation(ctx, groupID, userID); err != nil {
		return GroupJoinRequest{}, err
	} else if exists {
		return GroupJoinRequest{}, ErrPendingExists
	}

	return s.repo.CreateJoinRequest(ctx, groupID, userID)
}

func (s *Service) ListJoinRequests(ctx context.Context, groupID, viewerID string) ([]GroupJoinRequest, error) {
	if err := s.requireCreator(ctx, groupID, viewerID); err != nil {
		return nil, err
	}
	return s.repo.ListPendingJoinRequests(ctx, groupID)
}

func (s *Service) AcceptJoinRequest(ctx context.Context, requestID, viewerID string) error {
	groupID, _, err := s.repo.GetJoinRequestTarget(ctx, requestID)
	if err != nil {
		return err
	}

	if err := s.requireCreator(ctx, groupID, viewerID); err != nil {
		return err
	}

	return s.repo.AcceptJoinRequest(ctx, requestID)
}

func (s *Service) DeclineJoinRequest(ctx context.Context, requestID, viewerID string) error {
	groupID, _, err := s.repo.GetJoinRequestTarget(ctx, requestID)
	if err != nil {
		return err
	}

	if err := s.requireCreator(ctx, groupID, viewerID); err != nil {
		return err
	}

	return s.repo.DeclineJoinRequest(ctx, requestID)
}

func (s *Service) Invite(ctx context.Context, groupID, inviterID string, input InviteToGroupInput) (GroupInvitation, error) {
	inviteeID := strings.TrimSpace(input.InviteeID)
	if inviteeID == "" || inviteeID == inviterID {
		return GroupInvitation{}, ErrInvalidInput
	}
    if err := s.requireMember(ctx, groupID, inviterID); err != nil {
		return GroupInvitation{}, err
	}

	exists, err := s.repo.UserExists(ctx, inviteeID)
	if err != nil {
		return GroupInvitation{}, err
	}
	if !exists {
		return GroupInvitation{}, ErrUserNotFound
	}

	if _, isMember, err := s.repo.GetMembershipRole(ctx, groupID, inviteeID); err != nil {
		return GroupInvitation{}, err
	} else if isMember {
		return GroupInvitation{}, ErrAlreadyMember
	}

	if _, exists, err := s.repo.GetPendingInvitation(ctx, groupID, inviteeID); err != nil {
		return GroupInvitation{}, err
	} else if exists {
		return GroupInvitation{}, ErrPendingExists
	}

	return s.repo.CreateInvitation(ctx, groupID, inviterID, inviteeID)
}

func (s *Service) ListInvitations(ctx context.Context, inviteeID string) ([]GroupInvitation, error) {
	return s.repo.ListPendingInvitations(ctx, inviteeID)
}

func (s *Service) AcceptInvitation(ctx context.Context, invitationID, viewerID string) error {
	_, inviteeID, err := s.repo.GetInvitationTarget(ctx, invitationID)
	if err != nil {
		return err
	}
	if inviteeID != viewerID {
		return ErrForbidden
	}
	return s.repo.AcceptInvitation(ctx, invitationID)
}

func (s *Service) requireMember(ctx context.Context, groupID, userID string) error {
	group, err := s.repo.GetGroupByID(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if !group.ViewerStatus.IsMember {
		return ErrForbidden
	}
	return nil
}
func (s *Service) DeclineInvitation(ctx context.Context, invitationID, viewerID string) error {
	_, inviteeID, err := s.repo.GetInvitationTarget(ctx, invitationID)
	if err != nil {
		return err
	}
	if inviteeID != viewerID {
		return ErrForbidden
	}
	return s.repo.DeclineInvitation(ctx, invitationID)
}

func (s *Service) ListMembers(ctx context.Context, groupID string) ([]GroupMember, error) {
	if _, err := s.repo.GetGroupByID(ctx, groupID, ""); err != nil {
		return nil, err
	}
	return s.repo.ListMembers(ctx, groupID)
}

func (s *Service) CurrentUserID(ctx context.Context, sessionID string) (string, error) {
	if sessionID == "" {
		return "", ErrInvalidCredentials
	}
	return s.repo.GetUserBySessionID(ctx, sessionID)
}

func (s *Service) requireCreator(ctx context.Context, groupID, userID string) error {
	group, err := s.repo.GetGroupByID(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if group.CreatorID != userID {
		return ErrForbidden
	}
	return nil
}
