package posts

import (
	"context"
	"errors"
	"strings"
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

func (s *Service) CreatePost(ctx context.Context, authorID string, input CreatePostInput) (Post, error) {
	if strings.TrimSpace(input.Body) == "" {
		return Post{}, ErrInvalidInput
	}
	if input.Privacy == "selected_followers" && len(input.ViewerIDs) == 0 {
		return Post{}, ErrInvalidInput
	}
	return s.repo.CreatePost(ctx, authorID, input)
}

func (s *Service) GetFeed(ctx context.Context, viewerID string, limit, offset int) ([]Post, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.repo.GetFeedPosts(ctx, viewerID, limit, offset)
}

func (s *Service) GetUserPosts(ctx context.Context, authorID, viewerID string, limit, offset int) ([]Post, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.repo.GetUserPosts(ctx, authorID, viewerID, limit, offset)
}

func (s *Service) GetGroupPosts(ctx context.Context, groupID, viewerID string, limit, offset int) ([]Post, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.repo.GetGroupPosts(ctx, groupID, viewerID, limit, offset)
}

func (s *Service) DeletePost(ctx context.Context, postID, requesterID string) error {
	return s.repo.DeletePost(ctx, postID, requesterID)
}

func (s *Service) UpdateImagePath(ctx context.Context, postID, authorID, imagePath string) error {
	return s.repo.UpdateImagePath(ctx, postID, authorID, imagePath)
}

func (s *Service) GetFollowersOfUser(ctx context.Context, userID string) ([]FollowerSummary, error) {
	return s.repo.GetFollowersOfUser(ctx, userID)
}

func (s *Service) CurrentUserID(ctx context.Context, sessionID string) (string, error) {
	if sessionID == "" {
		return "", ErrInvalidCredentials
	}
	return s.repo.GetUserBySessionID(ctx, sessionID)
}
