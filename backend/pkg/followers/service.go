package followers

import (
	"context"
	"errors"
)

var ErrCannotFollowSelf = errors.New("cannot follow yourself")
 var ErrInvalidCredentials = errors.New("invalid credentials")
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Follow initiates a follow. For public profiles it immediately creates the follower
// relationship. For private profiles it creates a pending follow request.
func (s *Service) Follow(ctx context.Context, followerID, followingID string) error {
	if followerID == followingID {
		return ErrCannotFollowSelf
	}

	// Already following?
	already, err := s.repo.IsFollowing(ctx, followerID, followingID)
	if err != nil {
		return err
	}
	if already {
		return ErrAlreadyFollowing
	}

	// Check target profile visibility
	visibility, err := s.repo.GetProfileVisibility(ctx, followingID)
	if err != nil {
		return err
	}

	if visibility == "public" {
		// Direct follow — no request needed
		return s.repo.CreateFollower(ctx, followerID, followingID)
	}

	// Private profile — check if there's already a pending request
	status, err := s.repo.GetFollowRequestStatus(ctx, followerID, followingID)
	if err != nil {
		return err
	}
	if status == "pending" {
		return ErrRequestAlreadyExists
	}

	return s.repo.CreateFollowRequest(ctx, followerID, followingID)
}

// Unfollow removes a follower relationship or cancels a pending request
func (s *Service) Unfollow(ctx context.Context, followerID, followingID string) error {
	// Remove actual follow if it exists
	if err := s.repo.DeleteFollower(ctx, followerID, followingID); err != nil {
		return err
	}
	// Also clean up any pending request
	return s.repo.DeleteFollowRequest(ctx, followerID, followingID)
}

// AcceptRequest accepts a pending follow request
func (s *Service) AcceptRequest(ctx context.Context, requestID, receiverID string) error {
	// We need the sender ID — get it from pending requests
	reqs, err := s.repo.GetPendingRequests(ctx, receiverID)
	if err != nil {
		return err
	}
	for _, req := range reqs {
		if req.ID == requestID {
			return s.repo.AcceptFollowRequest(ctx, requestID, req.Sender.ID, receiverID)
		}
	}
	return ErrNotFound
}

// DeclineRequest declines a pending follow request
func (s *Service) DeclineRequest(ctx context.Context, requestID, receiverID string) error {
	return s.repo.DeclineFollowRequest(ctx, requestID, receiverID)
}

// GetFollowers returns the follower list for a user
func (s *Service) GetFollowers(ctx context.Context, targetID string) ([]UserSummary, error) {
	return s.repo.GetFollowers(ctx, targetID)
}

// GetFollowing returns the following list for a user
func (s *Service) GetFollowing(ctx context.Context, targetID string) ([]UserSummary, error) {
	return s.repo.GetFollowing(ctx, targetID)
}

// GetPendingRequests returns incoming follow requests for a user
func (s *Service) GetPendingRequests(ctx context.Context, receiverID string) ([]FollowRequest, error) {
	return s.repo.GetPendingRequests(ctx, receiverID)
}

// GetFollowStatus returns the follow relationship from viewerID → targetID
func (s *Service) GetFollowStatus(ctx context.Context, viewerID, targetID string) (FollowStatus, error) {
	var fs FollowStatus

	following, err := s.repo.IsFollowing(ctx, viewerID, targetID)
	if err != nil {
		return fs, err
	}
	fs.IsFollowing = following

	if !following {
		status, err := s.repo.GetFollowRequestStatus(ctx, viewerID, targetID)
		if err != nil {
			return fs, err
		}
		if status == "pending" {
			fs.HasPendingRequest = true
			// Get the request ID
			reqs, err := s.repo.GetPendingRequests(ctx, targetID)
			if err == nil {
				for _, req := range reqs {
					if req.Sender.ID == viewerID {
						fs.RequestID = req.ID
						break
					}
				}
			}
		}
	}

	return fs, nil
}



func (s *Service) currentUserID(ctx context.Context, sessionID string) (string, error) {
    if sessionID == "" {
		return "", ErrInvalidCredentials
	}
   
   
    return s.repo.GetuserID(ctx, sessionID)
}