package notifications
 
import (
	"context"
	"errors"
)
 
var ErrInvalidCredentials = errors.New("invalid credentials")
 
type Service struct {
	repo *Repository
}
 
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}
 
func (s *Service) CurrentUserID(ctx context.Context, sessionID string) (string, error) {
	if sessionID == "" {
		return "", ErrInvalidCredentials
	}
	return s.repo.getUserBySessionID(ctx, sessionID)
}


// NotifyFollowRequest — private profile received a follow request.
// refID = follow_requests.id — frontend uses it to call POST /follow/requests/{refID}/accept|decline
func (s *Service) NotifyFollowRequest(ctx context.Context, recipientID, actorID, followRequestID string) error {
	return s.repo.Create(ctx, recipientID, TypeFollowRequest, followRequestID, actorID, "", "")
}

// NotifyGroupInvitation — user was invited to a group.
// refID = group_invitations.id — frontend uses it to call POST /groups/invitations/{refID}/accept|decline
func (s *Service) NotifyGroupInvitation(ctx context.Context, inviteeID, actorID, groupID, invitationID string) error {
	return s.repo.Create(ctx, inviteeID, TypeGroupInvite, invitationID, actorID, groupID, "")
}
 

// NotifyGroupJoinRequest — someone requested to join the creator's group.
// refID = group_join_requests.id — frontend uses it to call POST /groups/requests/{refID}/accept|decline
func (s *Service) NotifyGroupJoinRequest(ctx context.Context, creatorID, actorID, groupID, joinRequestID string) error {
	return s.repo.Create(ctx, creatorID, TypeGroupJoinReq, joinRequestID, actorID, groupID, "")
}
// NotifyGroupEvent — a new event was created in a group.
// info only — no accept/decline needed, ref_id = event_id for navigation.
func (s *Service) NotifyGroupEvent(ctx context.Context, memberIDs []string, actorID, groupID, eventID string) error {
	for _, memberID := range memberIDs {
		if memberID == actorID {
			continue
		}
		if err := s.repo.Create(ctx, memberID, TypeGroupEvent, eventID, actorID, groupID, eventID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) GetAll(ctx context.Context, userID string) ([]Notification, error) {
	return s.repo.GetAll(ctx, userID)
}
 
func (s *Service) GetUnread(ctx context.Context, userID string) ([]Notification, error) {
	return s.repo.GetUnread(ctx, userID)
}
 
func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}
 
func (s *Service) MarkRead(ctx context.Context, notifID, userID string) error {
	return s.repo.MarkRead(ctx, notifID, userID)
}
 
func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}