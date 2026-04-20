package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var ErrUserNotFound = errors.New("user not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetUserByID(ctx context.Context, id string) (User, error) {
	var user User
	err := r.db.QueryRowContext(ctx, `
		SELECT id, email, first_name, last_name, date_of_birth,
		       COALESCE(avatar_path, ''), COALESCE(nickname, ''), COALESCE(about_me, ''),
		       profile_visibility, created_at, updated_at
		FROM users WHERE id = ?;
	`, id).Scan(
		&user.ID, &user.Email, &user.FirstName, &user.LastName, &user.DateOfBirth,
		&user.AvatarPath, &user.Nickname, &user.AboutMe,
		&user.ProfileVisibility, &user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	return user, err
}

func (r *Repository) GetUserBySessionID(ctx context.Context, sessionID string) (User, error) {
	var user User
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.first_name, u.last_name, u.date_of_birth,
		       COALESCE(u.avatar_path, ''), COALESCE(u.nickname, ''), COALESCE(u.about_me, ''),
		       u.profile_visibility, u.created_at, u.updated_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = ? AND s.expires_at > CURRENT_TIMESTAMP;
	`, sessionID).Scan(
		&user.ID, &user.Email, &user.FirstName, &user.LastName, &user.DateOfBirth,
		&user.AvatarPath, &user.Nickname, &user.AboutMe,
		&user.ProfileVisibility, &user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	return user, err
}

func (r *Repository) UpdateUser(ctx context.Context, id string, input UpdateInput) (User, error) {
	setClauses := []string{"updated_at = CURRENT_TIMESTAMP"}
	args := []any{}

	if input.Nickname != nil {
		setClauses = append(setClauses, "nickname = ?")
		if *input.Nickname == "" {
			args = append(args, nil)
		} else {
			args = append(args, *input.Nickname)
		}
	}
	if input.AboutMe != nil {
		setClauses = append(setClauses, "about_me = ?")
		if *input.AboutMe == "" {
			args = append(args, nil)
		} else {
			args = append(args, *input.AboutMe)
		}
	}
	if input.AvatarPath != nil {
		setClauses = append(setClauses, "avatar_path = ?")
		if *input.AvatarPath == "" {
			args = append(args, nil)
		} else {
			args = append(args, *input.AvatarPath)
		}
	}
	if input.ProfileVisibility != nil {
		v := *input.ProfileVisibility
		if v != "public" && v != "private" {
			v = "public"
		}
		setClauses = append(setClauses, "profile_visibility = ?")
		args = append(args, v)
	}

	if len(setClauses) == 1 {
		// Nothing to update
		return r.GetUserByID(ctx, id)
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE users SET %s WHERE id = ?;", strings.Join(setClauses, ", "))
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return User{}, err
	}

	return r.GetUserByID(ctx, id)
}