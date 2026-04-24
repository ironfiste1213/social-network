package groups

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"
)

///////////////// bel role ----> member_role!!!!!!!!!!!!!!!!!!

var ErrNotFound = errors.New("group not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateGroup(ctx context.Context, creatorID string, input CreateGroupInput) (Group, error) {
	id := uuid.NewString()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Group{}, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO groups (id, title, group_description, creator_id)
		VALUES (?, ?, ?, ?);
	`, id, strings.TrimSpace(input.Title), strings.TrimSpace(input.Description), creatorID)
	if err != nil {
		return Group{}, err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO group_members (group_id, user_id, role)
		VALUES (?, ?, 'creator');
	`, id, creatorID)
	if err != nil {
		return Group{}, err
	}

	if err := tx.Commit(); err != nil {
		return Group{}, err
	}

	return r.GetGroupByID(ctx, id, creatorID)
}


func (r *Repository) GetGroupByID(ctx context.Context, groupID, viewerID string) (Group, error) {
	var g Group
	var creator UserSummary
	var status GroupViewerStatus

	err := r.db.QueryRowContext(ctx, `
		SELECT g.id,
		       g.title,
		       COALESCE(g.group_description, ''),
		       g.creator_id,
		       g.created_at,
		       u.first_name,
		       u.last_name,
		       COALESCE(u.nickname, ''),
		       COALESCE(u.avatar_path, ''),
		       (SELECT COUNT(1) FROM group_members gm WHERE gm.group_id = g.id) AS member_count,
		       EXISTS(SELECT 1 FROM group_members gm WHERE gm.group_id = g.id AND gm.user_id = ?),
		       COALESCE((SELECT gm.role FROM group_members gm WHERE gm.group_id = g.id AND gm.user_id = ? LIMIT 1), ''),
		       EXISTS(SELECT 1 FROM group_join_requests gjr WHERE gjr.group_id = g.id AND gjr.user_id = ? AND gjr.status = 'pending'),
		       COALESCE((SELECT gjr.id FROM group_join_requests gjr WHERE gjr.group_id = g.id AND gjr.user_id = ? AND gjr.status = 'pending' LIMIT 1), ''),
		       EXISTS(SELECT 1 FROM group_invitations gi WHERE gi.group_id = g.id AND gi.invitee_id = ? AND gi.status = 'pending'),
		       COALESCE((SELECT gi.id FROM group_invitations gi WHERE gi.group_id = g.id AND gi.invitee_id = ? AND gi.status = 'pending' LIMIT 1), '')
		FROM groups g
		JOIN users u ON u.id = g.creator_id
		WHERE g.id = ?;
	`, viewerID, viewerID, viewerID, viewerID, viewerID, viewerID, groupID).Scan(
		&g.ID,
		&g.Title,
		&g.Description,
		&g.CreatorID,
		&g.CreatedAt,
		&creator.FirstName,
		&creator.LastName,
		&creator.Nickname,
		&creator.AvatarPath,
		&g.MemberCount,
		&status.IsMember,
		&status.Role,
		&status.HasPendingJoin,
		&status.PendingJoinRequestID,
		&status.HasPendingInvite,
		&status.PendingInvitationID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Group{}, ErrNotFound
	}
	if err != nil {
		return Group{}, err
	}

	creator.ID = g.CreatorID
	g.Creator = &creator
	g.ViewerStatus = status

	return g, nil
}

func (r *Repository) ListGroups(ctx context.Context, viewerID string, limit, offset int) ([]Group, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT g.id,
		       g.title,
		       COALESCE(g.group_description, ''),
		       g.creator_id,
		       g.created_at,
		       u.first_name,
		       u.last_name,
		       COALESCE(u.nickname, ''),
		       COALESCE(u.avatar_path, ''),
		       (SELECT COUNT(1) FROM group_members gm WHERE gm.group_id = g.id) AS member_count,
		       EXISTS(SELECT 1 FROM group_members gm WHERE gm.group_id = g.id AND gm.user_id = ?),
		       COALESCE((SELECT gm.role FROM group_members gm WHERE gm.group_id = g.id AND gm.user_id = ? LIMIT 1), ''),
		       EXISTS(SELECT 1 FROM group_join_requests gjr WHERE gjr.group_id = g.id AND gjr.user_id = ? AND gjr.status = 'pending'),
		       COALESCE((SELECT gjr.id FROM group_join_requests gjr WHERE gjr.group_id = g.id AND gjr.user_id = ? AND gjr.status = 'pending' LIMIT 1), ''),
		       EXISTS(SELECT 1 FROM group_invitations gi WHERE gi.group_id = g.id AND gi.invitee_id = ? AND gi.status = 'pending'),
		       COALESCE((SELECT gi.id FROM group_invitations gi WHERE gi.group_id = g.id AND gi.invitee_id = ? AND gi.status = 'pending' LIMIT 1), '')
		FROM groups g
		JOIN users u ON u.id = g.creator_id
		ORDER BY g.created_at DESC
		LIMIT ? OFFSET ?;
	`, viewerID, viewerID, viewerID, viewerID, viewerID, viewerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var g Group
		var creator UserSummary
		var status GroupViewerStatus

		if err := rows.Scan(
			&g.ID,
			&g.Title,
			&g.Description,
			&g.CreatorID,
			&g.CreatedAt,
			&creator.FirstName,
			&creator.LastName,
			&creator.Nickname,
			&creator.AvatarPath,
			&g.MemberCount,
			&status.IsMember,
			&status.Role,
			&status.HasPendingJoin,
			&status.PendingJoinRequestID,
			&status.HasPendingInvite,
			&status.PendingInvitationID,
		); err != nil {
			return nil, err
		}

		creator.ID = g.CreatorID
		g.Creator = &creator
		g.ViewerStatus = status
		groups = append(groups, g)
	}

	return groups, rows.Err()
}

func (r *Repository) UserExists(ctx context.Context, userID string) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = ?);`, userID).Scan(&exists)
	return exists == 1, err
}


func (r *Repository) GetMembershipRole(ctx context.Context, groupID, userID string) (string, bool, error) {
	var role string
	err := r.db.QueryRowContext(ctx, `
		SELECT role
		FROM group_members
		WHERE group_id = ? AND user_id = ?;
	`, groupID, userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return role, err == nil, err
}


func (r *Repository) GetPendingJoinRequest(ctx context.Context, groupID, userID string) (string, bool, error) {
	var requestID string
	err := r.db.QueryRowContext(ctx, `
		SELECT id
		FROM group_join_requests
		WHERE group_id = ? AND user_id = ? AND status = 'pending'
		LIMIT 1;
	`, groupID, userID).Scan(&requestID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return requestID, err == nil, err
}



func (r *Repository) GetPendingInvitation(ctx context.Context, groupID, userID string) (string, bool, error) {
	var invitationID string
	err := r.db.QueryRowContext(ctx, `
		SELECT id
		FROM group_invitations
		WHERE group_id = ? AND invitee_id = ? AND status = 'pending'
		LIMIT 1;
	`, groupID, userID).Scan(&invitationID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return invitationID, err == nil, err
}