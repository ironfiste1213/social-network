package users

import "context"

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
	return s.repo.UpdateUser(ctx, userID, input)
}