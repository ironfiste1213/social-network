package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
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

// IsFollowing returns true if followerID follows followingID.
func (r *Repository) IsFollowing(ctx context.Context, followerID, followingID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM followers WHERE follower_id = ? AND following_id = ?`,
		followerID, followingID,
	).Scan(&count)
	return count > 0, err
}

func (r *Repository) UpdateUser(ctx context.Context, id string, input UpdateInput) (User, error) {
	if input.ProfileVisibility != nil {
		newVisibility := *input.ProfileVisibility
		if newVisibility != "public" && newVisibility != "private" {
			newVisibility = "public"
		}

		currentVisibility, err := r.getProfileVisibility(ctx, id)
		if err != nil {
			return User{}, err
		}

		if currentVisibility == "private" && newVisibility == "public" {
			return r.updateUserAndPromotePendingRequests(ctx, id, input, newVisibility)
		}
	}

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
		return User{}, mapUniquenessError(err)
	}

	return r.GetUserByID(ctx, id)
}

func (r *Repository) getProfileVisibility(ctx context.Context, id string) (string, error) {
	var visibility string
	err := r.db.QueryRowContext(ctx,
		`SELECT profile_visibility FROM users WHERE id = ?;`,
		id,
	).Scan(&visibility)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrUserNotFound
	}
	return visibility, err
}

func (r *Repository) updateUserAndPromotePendingRequests(ctx context.Context, id string, input UpdateInput, visibility string) (User, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, err
	}

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

	setClauses = append(setClauses, "profile_visibility = ?")
	args = append(args, visibility)
	args = append(args, id)

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = ?;", strings.Join(setClauses, ", "))
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		_ = tx.Rollback()
		return User{}, mapUniquenessError(err)
	}

	if err := r.promotePendingRequestsToFollowersTx(ctx, tx, id); err != nil {
		_ = tx.Rollback()
		return User{}, err
	}

	if err := tx.Commit(); err != nil {
		return User{}, err
	}

	return r.GetUserByID(ctx, id)
}

func (r *Repository) promotePendingRequestsToFollowersTx(ctx context.Context, tx *sql.Tx, receiverID string) error {
	now := time.Now().UTC()

	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO followers (follower_id, following_id)
		SELECT sender_id, receiver_id
		FROM follow_requests
		WHERE receiver_id = ? AND status = 'pending';
	`, receiverID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE follow_requests
		SET status = 'accepted', responded_at = ?
		WHERE receiver_id = ? AND status = 'pending';
	`, now, receiverID); err != nil {
		return err
	}

	return nil
}

func (r *Repository) NicknameExistsForOtherUsers(ctx context.Context, userID, nickname string) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM users
			WHERE id != ?
			  AND nickname IS NOT NULL
			  AND TRIM(nickname) != ''
			  AND LOWER(nickname) = LOWER(?)
		);
	`, userID, nickname).Scan(&exists)
	return exists == 1, err
}

func (r *Repository) SearchUsersByNickname(ctx context.Context, excludeUserID, query string, limit int) ([]SearchResult, error) {
	normalizedQuery := strings.ToLower(query)

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, first_name, last_name, COALESCE(nickname, ''), COALESCE(avatar_path, ''), profile_visibility
		FROM users
		WHERE id != ?
		  AND nickname IS NOT NULL
		  AND TRIM(nickname) != ''
		  AND LOWER(nickname) LIKE ?
		ORDER BY
		  CASE
		    WHEN LOWER(nickname) = ? THEN 0
		    WHEN LOWER(nickname) LIKE ? THEN 1
		    ELSE 2
		  END,
		  LOWER(nickname) ASC
		LIMIT ?;
	`, excludeUserID, "%"+normalizedQuery+"%", normalizedQuery, normalizedQuery+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var user SearchResult
		if err := rows.Scan(
			&user.ID,
			&user.FirstName,
			&user.LastName,
			&user.Nickname,
			&user.AvatarPath,
			&user.ProfileVisibility,
		); err != nil {
			return nil, err
		}
		results = append(results, user)
	}

	return results, rows.Err()
}

func mapUniquenessError(err error) error {
	if err == nil {
		return nil
	}

	if strings.Contains(err.Error(), "UNIQUE constraint failed: users.nickname") {
		return ErrNicknameAlreadyExists
	}

	return err
}
