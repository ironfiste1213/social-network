package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const SessionDuration = 7 * 24 * time.Hour

var (
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrInvalidInput          = errors.New("invalid input")
	ErrEmailAlreadyExists    = errors.New("email already exists")
	ErrNicknameAlreadyExists = errors.New("nickname already exists")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (User, Session, error) {
	fmt.Println("[SERVICE] Register started for email:", input.Email)
	if err := validateRegisterInput(input); err != nil {
		fmt.Println("[SERVICE] Register validation failed")
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

	emailExists, err := s.repo.EmailExists(ctx, user.Email)
	if err != nil {
		return User{}, Session{}, err
	}
	if emailExists {
		return User{}, Session{}, ErrEmailAlreadyExists
	}

	if user.Nickname != "" {
		nicknameExists, err := s.repo.NicknameExists(ctx, user.Nickname)
		if err != nil {
			return User{}, Session{}, err
		}
		if nicknameExists {
			return User{}, Session{}, ErrNicknameAlreadyExists
		}
	}

	fmt.Println("[SERVICE] Register calling CreateUser")
	if err := s.repo.CreateUser(ctx, user); err != nil {
		fmt.Println("[SERVICE] Register CreateUser failed")
		return User{}, Session{}, err
	}
	fmt.Println("[SERVICE] Register user created successfully")

	fmt.Println("[SERVICE] Register creating session")
	session := newSession(user.ID)
	if err := s.repo.CreateSession(ctx, session); err != nil {
		fmt.Println("[SERVICE] Register session creation failed")
		return User{}, Session{}, err
	}
	fmt.Println("[SERVICE] Register session created successfully")

	fmt.Println("[SERVICE] Register fetching created user by email")
	createdUser, err := s.repo.GetUserByEmail(ctx, user.Email)
	if err != nil {
		fmt.Println("[SERVICE] Register GetUserByEmail failed")
		return User{}, Session{}, err
	}
	fmt.Println("[SERVICE] Register user fetched successfully")

	return createdUser, session, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (User, Session, error) {
	fmt.Println("[SERVICE] Login started for email:", input.Email)
	if strings.TrimSpace(input.Email) == "" || input.Password == "" {
		fmt.Println("[SERVICE] Login validation failed")
		return User{}, Session{}, ErrInvalidInput
	}

	user, err := s.repo.GetUserByEmail(ctx, strings.TrimSpace(strings.ToLower(input.Email)))
	if err != nil {
		fmt.Println("[SERVICE] Login user not found or error")
		if errors.Is(err, ErrUserNotFound) {
			return User{}, Session{}, ErrInvalidCredentials
		}
		return User{}, Session{}, err
	}
	fmt.Println("[SERVICE] Login user found")

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		fmt.Println("[SERVICE] Login password mismatch")
		return User{}, Session{}, ErrInvalidCredentials
	}
	fmt.Println("[SERVICE] Login password match")

	fmt.Println("[SERVICE] Login creating session")
	session := newSession(user.ID)
	if err := s.repo.CreateSession(ctx, session); err != nil {
		fmt.Println("[SERVICE] Login session creation failed")
		return User{}, Session{}, err
	}
	fmt.Println("[SERVICE] Login session created successfully")

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
