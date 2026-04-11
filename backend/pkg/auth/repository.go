package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrUserNotFound = errors.New("user not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(ctx context.Context, user User) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (
			id, email, password_hash, first_name, last_name, date_of_birth,
			avatar_path, nickname, about_me, profile_visibility
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.FirstName,
		user.LastName,
		user.DateOfBirth,
		nullIfEmpty(user.AvatarPath),
		nullIfEmpty(user.Nickname),
		nullIfEmpty(user.AboutMe),
		user.ProfileVisibility,
	)
	return err
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var user User

	err := r.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, first_name, last_name, date_of_birth,
		       COALESCE(avatar_path, ''), COALESCE(nickname, ''), COALESCE(about_me, ''),
		       profile_visibility, created_at, updated_at
		FROM users
		WHERE email = ?;
	`, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.DateOfBirth,
		&user.AvatarPath,
		&user.Nickname,
		&user.AboutMe,
		&user.ProfileVisibility,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}

	return user, err
}

func (r *Repository) GetUserBySessionID(ctx context.Context, sessionID string) (User, error) {
	var user User

	err := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.password_hash, u.first_name, u.last_name, u.date_of_birth,
		       COALESCE(u.avatar_path, ''), COALESCE(u.nickname, ''), COALESCE(u.about_me, ''),
		       u.profile_visibility, u.created_at, u.updated_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = ? AND s.expires_at > CURRENT_TIMESTAMP;
	`, sessionID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.DateOfBirth,
		&user.AvatarPath,
		&user.Nickname,
		&user.AboutMe,
		&user.ProfileVisibility,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}

	return user, err
}

func (r *Repository) CreateSession(ctx context.Context, session Session) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, expires_at)
		VALUES (?, ?, ?);
	`, session.ID, session.UserID, session.ExpiresAt)
	return err
}

func (r *Repository) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?;`, sessionID)
	return err
}

func (r *Repository) DeleteExpiredSessions(ctx context.Context, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?;`, now)
	return err
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}

	return value
}
