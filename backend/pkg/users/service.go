package users

import (
	"context"
	"errors"
	"strings"
)

var ErrInvalidSearchQuery = errors.New("invalid search query")
var ErrNicknameAlreadyExists = errors.New("nickname already exists")

const (
	defaultSearchLimit = 8
	maxSearchLimit     = 20
	minSearchLength    = 2
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetUserByID(ctx context.Context, id string) (User, error) {
	return s.repo.GetUserByID(ctx, id)
}

func (s *Service) GetUserBySession(ctx context.Context, sessionID string) (User, error) {
	return s.repo.GetUserBySessionID(ctx, sessionID)
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, input UpdateInput) (User, error) {
	if input.Nickname != nil {
		trimmedNickname := strings.TrimSpace(*input.Nickname)
		input.Nickname = &trimmedNickname

		if trimmedNickname != "" {
			exists, err := s.repo.NicknameExistsForOtherUsers(ctx, userID, trimmedNickname)
			if err != nil {
				return User{}, err
			}
			if exists {
				return User{}, ErrNicknameAlreadyExists
			}
		}
	}

	return s.repo.UpdateUser(ctx, userID, input)
}

func (s *Service) SearchUsers(ctx context.Context, viewerID, query string, limit int) ([]SearchResult, error) {
	trimmedQuery := strings.TrimSpace(query)
	if len(trimmedQuery) < minSearchLength {
		return nil, ErrInvalidSearchQuery
	}

	switch {
	case limit <= 0:
		limit = defaultSearchLimit
	case limit > maxSearchLimit:
		limit = maxSearchLimit
	}

	return s.repo.SearchUsersByNickname(ctx, viewerID, trimmedQuery, limit)
}

func (s *Service) IsFollowing(ctx context.Context, followerID, followingID string) (bool, error) {
	return s.repo.IsFollowing(ctx, followerID, followingID)
}
