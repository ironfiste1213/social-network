package followers

import (
	"context"
	"errors"
	"fmt"
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
	fmt.Println("[FOLLOWERS][SERVICE] follow started follower:", followerID, "target:", followingID)
	if followerID == followingID {
		fmt.Println("[FOLLOWERS][SERVICE] follow validation failed: cannot follow self")
		return ErrCannotFollowSelf
	}

	// Already following?
	already, err := s.repo.IsFollowing(ctx, followerID, followingID)
	if err != nil {
		fmt.Println("[FOLLOWERS][SERVICE] follow failed during IsFollowing:", err)
		return err
	}
	if already {
		fmt.Println("[FOLLOWERS][SERVICE] follow skipped: already following")
		return ErrAlreadyFollowing
	}

	// Check target profile visibility
	visibility, err := s.repo.GetProfileVisibility(ctx, followingID)
	if err != nil {
		fmt.Println("[FOLLOWERS][SERVICE] follow failed during GetProfileVisibility:", err)
		return err
	}
	fmt.Println("[FOLLOWERS][SERVICE] target visibility:", visibility)

	if visibility == "public" {
		// Direct follow — no request needed
		fmt.Println("[FOLLOWERS][SERVICE] creating direct follower relationship")
		return s.repo.CreateFollower(ctx, followerID, followingID)
	}

	// Private profile — check if there's already a pending request
	status, err := s.repo.GetFollowRequestStatus(ctx, followerID, followingID)
	if err != nil {
		fmt.Println("[FOLLOWERS][SERVICE] follow failed during GetFollowRequestStatus:", err)
		return err
	}
	if status == "pending" {
		fmt.Println("[FOLLOWERS][SERVICE] follow skipped: pending request already exists")
		return ErrRequestAlreadyExists
	}

	fmt.Println("[FOLLOWERS][SERVICE] creating follow request for private profile")
	return s.repo.CreateFollowRequest(ctx, followerID, followingID)
}

// Unfollow removes a follower relationship or cancels a pending follow request
func (s *Service) Unfollow(ctx context.Context, followerID, followingID string) error {
	fmt.Println("[FOLLOWERS][SERVICE] unfollow started follower:", followerID, "target:", followingID)
	// Remove actual follow if it exists
	if err := s.repo.DeleteFollower(ctx, followerID, followingID); err != nil {
		fmt.Println("[FOLLOWERS][SERVICE] unfollow failed during DeleteFollower:", err)
		return err
	}
	// Also clean up any pending request
	fmt.Println("[FOLLOWERS][SERVICE] unfollow removing pending request if present")
	return s.repo.DeleteFollowRequest(ctx, followerID, followingID)
}

// AcceptRequest accepts a pending follow request
func (s *Service) AcceptRequest(ctx context.Context, requestID, receiverID string) error {
    senderID, targetReceiverID, err := s.repo.GetFollowRequestByID(ctx, requestID)
    if err != nil {
        return err
    }
    // Verify the receiver matches — prevents accepting someone else's request
    if targetReceiverID != receiverID {
        return ErrNotFound
    }
    return s.repo.AcceptFollowRequest(ctx, requestID, senderID, receiverID)
}

// DeclineRequest declines a pending follow request
func (s *Service) DeclineRequest(ctx context.Context, requestID, receiverID string) error {
	fmt.Println("[FOLLOWERS][SERVICE] decline request started request:", requestID, "receiver:", receiverID)
	return s.repo.DeclineFollowRequest(ctx, requestID, receiverID)
}

// CancelRequest allows a sender to cancel their own pending follow request
func (s *Service) CancelRequest(ctx context.Context, requestID, senderID string) error {
	fmt.Println("[FOLLOWERS][SERVICE] cancel request started request:", requestID, "sender:", senderID)
	return s.repo.DeleteFollowRequestByID(ctx, requestID, senderID)
}

// GetFollowers returns the follower list for a user
func (s *Service) GetFollowers(ctx context.Context, targetID string) ([]UserSummary, error) {
	fmt.Println("[FOLLOWERS][SERVICE] get followers target:", targetID)
	return s.repo.GetFollowers(ctx, targetID)
}

// GetFollowing returns the following list for a user
func (s *Service) GetFollowing(ctx context.Context, targetID string) ([]UserSummary, error) {
	fmt.Println("[FOLLOWERS][SERVICE] get following target:", targetID)
	return s.repo.GetFollowing(ctx, targetID)
}

// GetPendingRequests returns incoming follow requests for a user
func (s *Service) GetPendingRequests(ctx context.Context, receiverID string) ([]FollowRequest, error) {
	fmt.Println("[FOLLOWERS][SERVICE] get pending requests receiver:", receiverID)
	return s.repo.GetPendingRequests(ctx, receiverID)
}

// GetFollowStatus returns the follow relationship from viewerID → targetID
func (s *Service) GetFollowStatus(ctx context.Context, viewerID, targetID string) (FollowStatus, error) {
	var fs FollowStatus
	fmt.Println("[FOLLOWERS][SERVICE] get follow status viewer:", viewerID, "target:", targetID)

	following, err := s.repo.IsFollowing(ctx, viewerID, targetID)
	if err != nil {
		fmt.Println("[FOLLOWERS][SERVICE] get follow status failed during IsFollowing:", err)
		return fs, err
	}
	fs.IsFollowing = following
	fmt.Println("[FOLLOWERS][SERVICE] follow status is_following:", following)

	if !following {
		status, err := s.repo.GetFollowRequestStatus(ctx, viewerID, targetID)
		if err != nil {
			fmt.Println("[FOLLOWERS][SERVICE] get follow status failed during GetFollowRequestStatus:", err)
			return fs, err
		}
		if status == "pending" {
			fs.HasPendingRequest = true
			fmt.Println("[FOLLOWERS][SERVICE] follow status has pending request")
			// Get the request ID
			reqs, err := s.repo.GetPendingRequests(ctx, targetID)
			if err == nil {
				for _, req := range reqs {
					if req.Sender.ID == viewerID {
						fs.RequestID = req.ID
						fmt.Println("[FOLLOWERS][SERVICE] follow status request id:", req.ID)
						break
					}
				}
			} else {
				fmt.Println("[FOLLOWERS][SERVICE] follow status warning: failed to fetch pending requests:", err)
			}
		}
	}

	fmt.Println("[FOLLOWERS][SERVICE] get follow status completed")
	return fs, nil
}

func (s *Service) currentUserID(ctx context.Context, sessionID string) (string, error) {
	fmt.Println("[FOLLOWERS][SERVICE] current user lookup started")
	if sessionID == "" {
		fmt.Println("[FOLLOWERS][SERVICE] current user lookup failed: empty session id")
		return "", ErrInvalidCredentials
	}

	return s.repo.GetuserID(ctx, sessionID)
}
