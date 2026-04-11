package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const SessionDuration = 7 * 24 * time.Hour

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidInput       = errors.New("invalid input")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (User, Session, error) {
	if err := validateRegisterInput(input); err != nil {
		return User{}, Session{}, err
	}

	dateOfBirth, err := time.Parse("2006-01-02", input.DateOfBirth)
	if err != nil {
		return User{}, Session{}, ErrInvalidInput
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, Session{}, err
	}

	user := User{
		ID:                uuid.NewString(),
		Email:             strings.TrimSpace(strings.ToLower(input.Email)),
		PasswordHash:      string(passwordHash),
		FirstName:         strings.TrimSpace(input.FirstName),
		LastName:          strings.TrimSpace(input.LastName),
		DateOfBirth:       dateOfBirth,
		AvatarPath:        strings.TrimSpace(input.AvatarPath),
		Nickname:          strings.TrimSpace(input.Nickname),
		AboutMe:           strings.TrimSpace(input.AboutMe),
		ProfileVisibility: "public",
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return User{}, Session{}, err
	}

	session := newSession(user.ID)
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return User{}, Session{}, err
	}

	createdUser, err := s.repo.GetUserByEmail(ctx, user.Email)
	if err != nil {
		return User{}, Session{}, err
	}

	return createdUser, session, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (User, Session, error) {
	if strings.TrimSpace(input.Email) == "" || input.Password == "" {
		return User{}, Session{}, ErrInvalidInput
	}

	user, err := s.repo.GetUserByEmail(ctx, strings.TrimSpace(strings.ToLower(input.Email)))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return User{}, Session{}, ErrInvalidCredentials
		}
		return User{}, Session{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return User{}, Session{}, ErrInvalidCredentials
	}

	session := newSession(user.ID)
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return User{}, Session{}, err
	}

	return user, session, nil
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}

	return s.repo.DeleteSession(ctx, sessionID)
}

func (s *Service) CurrentUser(ctx context.Context, sessionID string) (User, error) {
	if sessionID == "" {
		return User{}, ErrInvalidCredentials
	}

	return s.repo.GetUserBySessionID(ctx, sessionID)
}

func validateRegisterInput(input RegisterInput) error {
	switch {
	case strings.TrimSpace(input.Email) == "":
		return ErrInvalidInput
	case input.Password == "":
		return ErrInvalidInput
	case strings.TrimSpace(input.FirstName) == "":
		return ErrInvalidInput
	case strings.TrimSpace(input.LastName) == "":
		return ErrInvalidInput
	case strings.TrimSpace(input.DateOfBirth) == "":
		return ErrInvalidInput
	default:
		return nil
	}
}

func newSession(userID string) Session {
	now := time.Now().UTC()

	return Session{
		ID:        uuid.NewString(),
		UserID:    userID,
		ExpiresAt: now.Add(SessionDuration),
		CreatedAt: now,
	}
}
