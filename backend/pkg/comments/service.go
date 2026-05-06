package comments

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

func (s *Service) CreateComment(ctx context.Context, postID, authorID string, input CreateCommentInput) (Comment, error) {
	if strings.TrimSpace(input.Body) == "" {
		return Comment{}, ErrInvalidInput
	}

	canView, err := s.repo.CanViewPost(ctx, postID, authorID)
	if err != nil {
		return Comment{}, err
	}
	if !canView {
		return Comment{}, ErrForbidden
	}

	return s.repo.CreateComment(ctx, postID, authorID, input)
}

func (s *Service) GetComments(ctx context.Context, postID, viewerID string) ([]Comment, error) {
	canView, err := s.repo.CanViewPost(ctx, postID, viewerID)
	if err != nil {
		return nil, err
	}
	if !canView {
		return nil, ErrForbidden
	}

	return s.repo.GetCommentsByPostID(ctx, postID)
}

func (s *Service) DeleteComment(ctx context.Context, commentID, requesterID string) error {
	return s.repo.DeleteComment(ctx, commentID, requesterID)
}

func (s *Service) UpdateImagePath(ctx context.Context, commentID, authorID, imagePath string) error {
	return s.repo.UpdateImagePath(ctx, commentID, authorID, imagePath)
}

func (s *Service) VerifyCommentPost(ctx context.Context, commentID, postID string) error {
	comment, err := s.repo.GetCommentByID(ctx, commentID)
	if err != nil {
		return err
	}
	if comment.PostID != postID {
		return ErrNotFound
	}
	return nil
}

func (s *Service) CurrentUserID(ctx context.Context, sessionID string) (string, error) {
	if sessionID == "" {
		return "", ErrInvalidCredentials
	}
	return s.repo.GetUserBySessionID(ctx, sessionID)
}