package http

import (
	"context"
	"database/sql"
	"errors"
)

var ErrNoRows = sql.ErrNoRows

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// GetSessionByToken retrieves a session from the database and checks its validity.
func (repo *Repository) GetSessionByToken(token string) (*Session, error) {
	stmt, err := repo.db.Prepare(`
		SELECT id, user_id, expires_at, created_at
		FROM sessions
		WHERE id = ?
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	session := &Session{}
	err = stmt.QueryRow(token).Scan(&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt)
	if err != nil {
		if err == ErrNoRows {
			return nil, nil // Session not found, not a server error
		}
		return nil, err
	}

	return session, nil
}

// DeleteSession removes a session from the database based on its token.
func (repo *Repository) DeleteSession(token string) error {
	stmt, err := repo.db.Prepare("DELETE FROM sessions WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(token)
	return err
}

func (r *Repository) GetUserBySessionID(ctx context.Context, sessionID string) (*User, error) {
	user :=&User{}

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
		return user, ErrUserNotFound
	}

	return user, err
}
