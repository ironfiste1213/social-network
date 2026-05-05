package followers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
func (r *Repository) IsFollowing(ctx context.Context, followerID, followingID string) (bool, error) {
	fmt.Println("[FOLLOWERS][REPO] IsFollowing follower:", followerID, "target:", followingID)
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM followers WHERE follower_id = ? AND following_id = ?`,
		followerID, followingID,
	).Scan(&count)
	return count > 0, err
}

// GetFollowRequestStatus returns the status of a pending request, or "" if none
func (r *Repository) GetFollowRequestStatus(ctx context.Context, senderID, receiverID string) (string, error) {
	fmt.Println("[FOLLOWERS][REPO] GetFollowRequestStatus sender:", senderID, "receiver:", receiverID)
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
	fmt.Println("[FOLLOWERS][REPO] GetProfileVisibility user:", userID)
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
	fmt.Println("[FOLLOWERS][REPO] CreateFollowRequest sender:", senderID, "receiver:", receiverID)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO follow_requests (id, sender_id, receiver_id, status) VALUES (?, ?, ?, 'pending')`,
		uuid.NewString(), senderID, receiverID,
	)
	return err
}

// CreateFollower directly adds a follower relationship (for public profiles)
func (r *Repository) CreateFollower(ctx context.Context, followerID, followingID string) error {
	fmt.Println("[FOLLOWERS][REPO] CreateFollower follower:", followerID, "target:", followingID)
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO followers (follower_id, following_id) VALUES (?, ?)`,
		followerID, followingID,
	)
	return err
}

// AcceptFollowRequest accepts a pending follow request and creates the follower row
func (r *Repository) AcceptFollowRequest(ctx context.Context, requestID, senderID, receiverID string) error {
	fmt.Println("[FOLLOWERS][REPO] AcceptFollowRequest request:", requestID, "sender:", senderID, "receiver:", receiverID)
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
	fmt.Println("[FOLLOWERS][REPO] DeclineFollowRequest request:", requestID, "receiver:", receiverID)
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE follow_requests SET status = 'declined', responded_at = ? WHERE id = ? AND receiver_id = ?`,
		now, requestID, receiverID,
	)
	return err
}

// DeleteFollower removes a follower relationship (unfollow)
func (r *Repository) DeleteFollower(ctx context.Context, followerID, followingID string) error {
	fmt.Println("[FOLLOWERS][REPO] DeleteFollower follower:", followerID, "target:", followingID)
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM followers WHERE follower_id = ? AND following_id = ?`,
		followerID, followingID,
	)
	return err
}

// DeleteFollowRequest removes a follow request (cancel or cleanup on unfollow)
func (r *Repository) DeleteFollowRequest(ctx context.Context, senderID, receiverID string) error {
	fmt.Println("[FOLLOWERS][REPO] DeleteFollowRequest sender:", senderID, "receiver:", receiverID)
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM follow_requests WHERE sender_id = ? AND receiver_id = ?`,
		senderID, receiverID,
	)
	return err
}

// DeleteFollowRequestByID removes a follow request by request ID (used for sender cancellation)
func (r *Repository) DeleteFollowRequestByID(ctx context.Context, requestID, senderID string) error {
	fmt.Println("[FOLLOWERS][REPO] DeleteFollowRequestByID request:", requestID, "sender:", senderID)
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM follow_requests WHERE id = ? AND sender_id = ?`,
		requestID, senderID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// GetFollowers returns users who follow targetID
func (r *Repository) GetFollowers(ctx context.Context, targetID string) ([]UserSummary, error) {
	fmt.Println("[FOLLOWERS][REPO] GetFollowers target:", targetID)
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
	fmt.Println("[FOLLOWERS][REPO] GetFollowing target:", targetID)
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
	fmt.Println("[FOLLOWERS][REPO] GetPendingRequests receiver:", receiverID)
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

func (r *Repository) GetuserID(ctx context.Context, sessionID string) (string, error) {
	fmt.Println("[FOLLOWERS][REPO] GetuserID session:", sessionID)
	var userId string
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id FROM sessions WHERE id = ? AND expires_at > CURRENT_TIMESTAMP`,
		sessionID,
	).Scan(&userId)
	if err != nil {
		fmt.Println("[FOLLOWERS][REPO] GetuserID failed:", err)
		return "", errors.New("invalid session")
	}
	fmt.Println("[FOLLOWERS][REPO] GetuserID success user:", userId)
	return userId, err
}
