package chat

import (
	"context"
	"database/sql"
)




type Repository  struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}
// CanChatPrivate returns true if at least one user follows the other.
// Per spec: "at least one of the users must be following the other."
func (r *Repository) CanChatPrivate(ctx context.Context, senderID, receiverID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM followers
		WHERE (follower_id = ? AND following_id = ?)
		   OR (follower_id = ? AND following_id = ?);
	`, senderID, receiverID, receiverID, senderID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
 
// IsGroupMember returns true if userID is a member of groupID.
func (r *Repository) IsGroupMember(ctx context.Context, groupID, userID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM group_members WHERE group_id = ? AND user_id = ?;
	`, groupID, userID).Scan(&count)
	return count > 0, err
}
 