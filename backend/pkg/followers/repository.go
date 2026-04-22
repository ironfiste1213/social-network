package followers

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)
 
var ErrNotFound = errors.New("not found")
var ErrAlreadyFollowing = errors.New("already following")
var ErrRequestAlreadyExists = errors.New("follow request already exists")
 

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// IsFollowing returns true if followerID follows followingID
func (r *Repository) IsFollowing(ctx context.Context, followingID, followerID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM followers WHERE follower_id = ? AND following_id = ?`,
		followerID, followingID,
	).Scan(&count)
	return count > 0, err
}

// GetFollowRequestStatus returns the status of a pending request, or "" if none
func (r *Repository) GetFollowRequestStatus(ctx context.Context, senderID, receiverID string) (string, error) {
	var status string
	err := r.db.QueryRowContext(ctx,
		`SELECT status FROM follow_requests WHERE sender_id = ? AND receiver_id = ?`,
		senderID, receiverID,
	).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return status, err
}

// GetProfileVisibility returns the profile_visibility field for a user
func (r *Repository) GetProfileVisibility(ctx context.Context, userID string) (string, error) {
	var v string
	err := r.db.QueryRowContext(ctx,
		`SELECT profile_visibility FROM users WHERE id = ?`, userID,
	).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return v, err
}

// CreateFollowRequest inserts a pending follow request
func (r *Repository) CreateFollowRequest(ctx context.Context, senderID, receiverID string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO follow_requests (id, sender_id, receiver_id, status) VALUES (?, ?, ?, 'pending')`,
		uuid.NewString(), senderID, receiverID,
	)
	return err
}

// CreateFollower directly adds a follower relationship (for public profiles)
func (r *Repository) CreateFollower(ctx context.Context, followerID, followingID string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO followers (follower_id, following_id) VALUES (?, ?)`,
		followerID, followingID,
	)
	return err
}

// AcceptFollowRequest accepts a pending follow request and creates the follower row
func (r *Repository) AcceptFollowRequest(ctx context.Context, requestID, senderID, receiverID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx,
		`UPDATE follow_requests SET status = 'accepted', responded_at = ? WHERE id = ? AND receiver_id = ?`,
		now, requestID, receiverID,
	); err != nil {
		_ = tx.Rollback()
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO followers (follower_id, following_id) VALUES (?, ?)`,
		senderID, receiverID,
	); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

 
// DeclineFollowRequest declines a pending follow request
func (r *Repository) DeclineFollowRequest(ctx context.Context, requestID, receiverID string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE follow_requests SET status = 'declined', responded_at = ? WHERE id = ? AND receiver_id = ?`,
		now, requestID, receiverID,
	)
	return err
}
 
 
// DeleteFollower removes a follower relationship (unfollow)
func (r *Repository) DeleteFollower(ctx context.Context, followerID, followingID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM followers WHERE follower_id = ? AND following_id = ?`,
		followerID, followingID,
	)
	return err
}

// DeleteFollowRequest removes a follow request (cancel or cleanup on unfollow)
func (r *Repository) DeleteFollowRequest(ctx context.Context, senderID, receiverID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM follow_requests WHERE sender_id = ? AND receiver_id = ?`,
		senderID, receiverID,
	)
	return err
}
 
// GetFollowers returns users who follow targetID
func (r *Repository) GetFollowers(ctx context.Context, targetID string) ([]UserSummary, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.first_name, u.last_name,
		       COALESCE(u.nickname, ''), COALESCE(u.avatar_path, '')
		FROM followers f
		JOIN users u ON u.id = f.follower_id
		WHERE f.following_id = ?
		ORDER BY u.first_name, u.last_name
	`, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUserSummaries(rows)
}
 
 
func scanUserSummaries(rows *sql.Rows) ([]UserSummary, error) {
	var result []UserSummary
	for rows.Next() {
		var u UserSummary
		if err := rows.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Nickname, &u.AvatarPath); err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

 
// GetFollowing returns users that targetID follows
func (r *Repository) GetFollowing(ctx context.Context, targetID string) ([]UserSummary, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.first_name, u.last_name,
		       COALESCE(u.nickname, ''), COALESCE(u.avatar_path, '')
		FROM followers f
		JOIN users u ON u.id = f.following_id
		WHERE f.follower_id = ?
		ORDER BY u.first_name, u.last_name
	`, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUserSummaries(rows)
}
 
 
// GetPendingRequests returns incoming pending follow requests for receiverID
func (r *Repository) GetPendingRequests(ctx context.Context, receiverID string) ([]FollowRequest, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT fr.id, u.id, u.first_name, u.last_name,
		       COALESCE(u.nickname, ''), COALESCE(u.avatar_path, ''),
		       fr.created_at
		FROM follow_requests fr
		JOIN users u ON u.id = fr.sender_id
		WHERE fr.receiver_id = ? AND fr.status = 'pending'
		ORDER BY fr.created_at DESC
	`, receiverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
 
	var result []FollowRequest
	for rows.Next() {
		var req FollowRequest
		if err := rows.Scan(
			&req.ID,
			&req.Sender.ID, &req.Sender.FirstName, &req.Sender.LastName,
			&req.Sender.Nickname, &req.Sender.AvatarPath,
			&req.CreatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, req)
	}
	return result, rows.Err()
}



func (r * Repository) GetuserID(ctx context.Context, sessionID string) (string, error) {
	var userId string
	 err := r.db.QueryRowContext(ctx,
        `SELECT user_id FROM sessions WHERE id = ? AND expires_at > CURRENT_TIMESTAMP`,
        sessionID,
    ).Scan(&userId)
    if err != nil {
        return "", errors.New("invalid session")
    }
	return userId, err
}