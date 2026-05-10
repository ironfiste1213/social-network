package groups

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

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
		INSERT INTO group_members (group_id, user_id, member_role)
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
		       COALESCE((SELECT gm.member_role FROM group_members gm WHERE gm.group_id = g.id AND gm.user_id = ? LIMIT 1), ''),
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

func (r *Repository) ListGroups(ctx context.Context, viewerID string, limit int, beforeID string) ([]Group, error) {
	query := `
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
		       COALESCE((SELECT gm.member_role FROM group_members gm WHERE gm.group_id = g.id AND gm.user_id = ? LIMIT 1), ''),
		       EXISTS(SELECT 1 FROM group_join_requests gjr WHERE gjr.group_id = g.id AND gjr.user_id = ? AND gjr.status = 'pending'),
		       COALESCE((SELECT gjr.id FROM group_join_requests gjr WHERE gjr.group_id = g.id AND gjr.user_id = ? AND gjr.status = 'pending' LIMIT 1), ''),
		       EXISTS(SELECT 1 FROM group_invitations gi WHERE gi.group_id = g.id AND gi.invitee_id = ? AND gi.status = 'pending'),
		       COALESCE((SELECT gi.id FROM group_invitations gi WHERE gi.group_id = g.id AND gi.invitee_id = ? AND gi.status = 'pending' LIMIT 1), '')
		FROM groups g
		JOIN users u ON u.id = g.creator_id
	`
	args := []any{viewerID, viewerID, viewerID, viewerID, viewerID, viewerID}
	if beforeID != "" {
		beforeCreatedAt, err := r.getGroupCreatedAt(ctx, beforeID)
		if err != nil {
			return nil, err
		}
		query += `
		WHERE (g.created_at < ? OR (g.created_at = ? AND g.id < ?))
		`
		args = append(args, beforeCreatedAt, beforeCreatedAt, beforeID)
	}
	query += `
		ORDER BY g.created_at DESC, g.id DESC
		LIMIT ?;
	`
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
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

func (r *Repository) getGroupCreatedAt(ctx context.Context, groupID string) (time.Time, error) {
	var createdAt time.Time
	err := r.db.QueryRowContext(ctx, `SELECT created_at FROM groups WHERE id = ?;`, groupID).Scan(&createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, ErrNotFound
	}
	return createdAt, err
}

func (r *Repository) UserExists(ctx context.Context, userID string) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = ?);`, userID).Scan(&exists)
	return exists == 1, err
}

func (r *Repository) GetMembershipRole(ctx context.Context, groupID, userID string) (string, bool, error) {
	var role string
	err := r.db.QueryRowContext(ctx, `
		SELECT member_role
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

func (r *Repository) CreateJoinRequest(ctx context.Context, groupID, userID string) (GroupJoinRequest, error) {
	id := uuid.NewString()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO group_join_requests (id, group_id, user_id)
		VALUES (?, ?, ?);
	`, id, groupID, userID)
	if err != nil {
		return GroupJoinRequest{}, err
	}

	return r.GetJoinRequestByID(ctx, id)
}

func (r *Repository) GetJoinRequestByID(ctx context.Context, requestID string) (GroupJoinRequest, error) {
	var req GroupJoinRequest
	var user UserSummary

	err := r.db.QueryRowContext(ctx, `
		SELECT gjr.id,
		       gjr.group_id,
		       gjr.created_at,
		       u.id,
		       u.first_name,
		       u.last_name,
		       COALESCE(u.nickname, ''),
		       COALESCE(u.avatar_path, '')
		FROM group_join_requests gjr
		JOIN users u ON u.id = gjr.user_id
		WHERE gjr.id = ?;
	`, requestID).Scan(
		&req.ID,
		&req.GroupID,
		&req.CreatedAt,
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Nickname,
		&user.AvatarPath,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return GroupJoinRequest{}, ErrNotFound
	}
	if err != nil {
		return GroupJoinRequest{}, err
	}

	req.User = user
	return req, nil
}

func (r *Repository) ListPendingJoinRequests(ctx context.Context, groupID string) ([]GroupJoinRequest, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT gjr.id,
		       gjr.group_id,
		       gjr.created_at,
		       u.id,
		       u.first_name,
		       u.last_name,
		       COALESCE(u.nickname, ''),
		       COALESCE(u.avatar_path, '')
		FROM group_join_requests gjr
		JOIN users u ON u.id = gjr.user_id
		WHERE gjr.group_id = ? AND gjr.status = 'pending'
		ORDER BY gjr.created_at ASC;
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []GroupJoinRequest
	for rows.Next() {
		var req GroupJoinRequest
		if err := rows.Scan(
			&req.ID,
			&req.GroupID,
			&req.CreatedAt,
			&req.User.ID,
			&req.User.FirstName,
			&req.User.LastName,
			&req.User.Nickname,
			&req.User.AvatarPath,
		); err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}

	return requests, rows.Err()
}

func (r *Repository) GetJoinRequestTarget(ctx context.Context, requestID string) (groupID, userID string, err error) {
	err = r.db.QueryRowContext(ctx, `
		SELECT group_id, user_id
		FROM group_join_requests
		WHERE id = ? AND status = 'pending';
	`, requestID).Scan(&groupID, &userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return groupID, userID, err
}

func (r *Repository) AcceptJoinRequest(ctx context.Context, requestID string) error {
	groupID, userID, err := r.GetJoinRequestTarget(ctx, requestID)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO group_members (group_id, user_id, member_role)
		VALUES (?, ?, 'member');
	`, groupID, userID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO chat_participants (chat_id, user_id)
		SELECT c.id, ?
		FROM chats c
		WHERE c.id = ? AND c.type = 'group';
	`, userID, groupID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE group_join_requests
		SET status = 'accepted', responded_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'pending';
	`, requestID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE group_invitations
		SET status = 'accepted', responded_at = CURRENT_TIMESTAMP
		WHERE group_id = ? AND invitee_id = ? AND status = 'pending';
	`, groupID, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) DeclineJoinRequest(ctx context.Context, requestID string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE group_join_requests
		SET status = 'declined', responded_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'pending';
	`, requestID)
	if err != nil {
		return err
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) CreateInvitation(ctx context.Context, groupID, inviterID, inviteeID string) (GroupInvitation, error) {
	id := uuid.NewString()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO group_invitations (id, group_id, inviter_id, invitee_id)
		VALUES (?, ?, ?, ?);
	`, id, groupID, inviterID, inviteeID)
	if err != nil {
		return GroupInvitation{}, err
	}

	return r.GetInvitationByID(ctx, id, inviteeID)
}

func (r *Repository) GetInvitationByID(ctx context.Context, invitationID, viewerID string) (GroupInvitation, error) {
	var invitation GroupInvitation
	var group Group
	var inviter UserSummary
	var viewerStatus GroupViewerStatus

	err := r.db.QueryRowContext(ctx, `
		SELECT gi.id,
		       gi.group_id,
		       gi.invitee_id,
		       gi.created_at,
		       u.id,
		       u.first_name,
		       u.last_name,
		       COALESCE(u.nickname, ''),
		       COALESCE(u.avatar_path, ''),
		       g.title,
		       COALESCE(g.group_description, ''),
		       g.creator_id,
		       g.created_at,
		       (SELECT COUNT(1) FROM group_members gm WHERE gm.group_id = g.id) AS member_count,
		       EXISTS(SELECT 1 FROM group_members gm WHERE gm.group_id = g.id AND gm.user_id = ?),
		       COALESCE((SELECT gm.member_role FROM group_members gm WHERE gm.group_id = g.id AND gm.user_id = ? LIMIT 1), ''),
		       EXISTS(SELECT 1 FROM group_join_requests gjr WHERE gjr.group_id = g.id AND gjr.user_id = ? AND gjr.status = 'pending'),
		       COALESCE((SELECT gjr.id FROM group_join_requests gjr WHERE gjr.group_id = g.id AND gjr.user_id = ? AND gjr.status = 'pending' LIMIT 1), ''),
		       EXISTS(SELECT 1 FROM group_invitations gx WHERE gx.group_id = g.id AND gx.invitee_id = ? AND gx.status = 'pending'),
		       COALESCE((SELECT gx.id FROM group_invitations gx WHERE gx.group_id = g.id AND gx.invitee_id = ? AND gx.status = 'pending' LIMIT 1), '')
		FROM group_invitations gi
		JOIN users u ON u.id = gi.inviter_id
		JOIN groups g ON g.id = gi.group_id
		WHERE gi.id = ?;
	`, viewerID, viewerID, viewerID, viewerID, viewerID, viewerID, invitationID).Scan(
		&invitation.ID,
		&invitation.GroupID,
		&invitation.InviteeID,
		&invitation.CreatedAt,
		&inviter.ID,
		&inviter.FirstName,
		&inviter.LastName,
		&inviter.Nickname,
		&inviter.AvatarPath,
		&group.Title,
		&group.Description,
		&group.CreatorID,
		&group.CreatedAt,
		&group.MemberCount,
		&viewerStatus.IsMember,
		&viewerStatus.Role,
		&viewerStatus.HasPendingJoin,
		&viewerStatus.PendingJoinRequestID,
		&viewerStatus.HasPendingInvite,
		&viewerStatus.PendingInvitationID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return GroupInvitation{}, ErrNotFound
	}
	if err != nil {
		return GroupInvitation{}, err
	}

	group.ID = invitation.GroupID
	group.ViewerStatus = viewerStatus
	invitation.Group = &group
	invitation.Inviter = inviter

	return invitation, nil
}

func (r *Repository) ListPendingInvitations(ctx context.Context, inviteeID string) ([]GroupInvitation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT gi.id,
		       gi.group_id,
		       gi.invitee_id,
		       gi.created_at,
		       u.id,
		       u.first_name,
		       u.last_name,
		       COALESCE(u.nickname, ''),
		       COALESCE(u.avatar_path, ''),
		       g.title,
		       COALESCE(g.group_description, ''),
		       g.creator_id,
		       g.created_at,
		       (SELECT COUNT(1) FROM group_members gm WHERE gm.group_id = g.id) AS member_count
		FROM group_invitations gi
		JOIN users u ON u.id = gi.inviter_id
		JOIN groups g ON g.id = gi.group_id
		WHERE gi.invitee_id = ? AND gi.status = 'pending'
		ORDER BY gi.created_at DESC;
	`, inviteeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []GroupInvitation
	for rows.Next() {
		var invitation GroupInvitation
		var group Group
		if err := rows.Scan(
			&invitation.ID,
			&invitation.GroupID,
			&invitation.InviteeID,
			&invitation.CreatedAt,
			&invitation.Inviter.ID,
			&invitation.Inviter.FirstName,
			&invitation.Inviter.LastName,
			&invitation.Inviter.Nickname,
			&invitation.Inviter.AvatarPath,
			&group.Title,
			&group.Description,
			&group.CreatorID,
			&group.CreatedAt,
			&group.MemberCount,
		); err != nil {
			return nil, err
		}
		group.ID = invitation.GroupID
		group.ViewerStatus.HasPendingInvite = true
		group.ViewerStatus.PendingInvitationID = invitation.ID
		invitation.Group = &group
		invitations = append(invitations, invitation)
	}

	return invitations, rows.Err()
}

func (r *Repository) GetInvitationTarget(ctx context.Context, invitationID string) (groupID, inviteeID string, err error) {
	err = r.db.QueryRowContext(ctx, `
		SELECT group_id, invitee_id
		FROM group_invitations
		WHERE id = ? AND status = 'pending';
	`, invitationID).Scan(&groupID, &inviteeID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return groupID, inviteeID, err
}

func (r *Repository) AcceptInvitation(ctx context.Context, invitationID string) error {
	groupID, inviteeID, err := r.GetInvitationTarget(ctx, invitationID)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO group_members (group_id, user_id, member_role)
		VALUES (?, ?, 'member');
	`, groupID, inviteeID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO chat_participants (chat_id, user_id)
		SELECT c.id, ?
		FROM chats c
		WHERE c.id = ? AND c.type = 'group';
	`, inviteeID, groupID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE group_invitations
		SET status = 'accepted', responded_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'pending';
	`, invitationID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE group_join_requests
		SET status = 'accepted', responded_at = CURRENT_TIMESTAMP
		WHERE group_id = ? AND user_id = ? AND status = 'pending';
	`, groupID, inviteeID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) DeclineInvitation(ctx context.Context, invitationID string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE group_invitations
		SET status = 'declined', responded_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'pending';
	`, invitationID)
	if err != nil {
		return err
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ListMembers(ctx context.Context, groupID string) ([]GroupMember, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id,
		       u.first_name,
		       u.last_name,
		       COALESCE(u.nickname, ''),
		       COALESCE(u.avatar_path, ''),
		       gm.member_role,
		       gm.joined_at
		FROM group_members gm
		JOIN users u ON u.id = gm.user_id
		WHERE gm.group_id = ?
		ORDER BY
		    CASE gm.member_role WHEN 'creator' THEN 0 ELSE 1 END,
		    gm.joined_at ASC;
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []GroupMember
	for rows.Next() {
		var member GroupMember
		if err := rows.Scan(
			&member.User.ID,
			&member.User.FirstName,
			&member.User.LastName,
			&member.User.Nickname,
			&member.User.AvatarPath,
			&member.Role,
			&member.JoinedAt,
		); err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, rows.Err()
}

func (r *Repository) GetUserBySessionID(ctx context.Context, sessionID string) (string, error) {
	var userID string
	err := r.db.QueryRowContext(ctx, `
		SELECT user_id
		FROM sessions
		WHERE id = ? AND expires_at > CURRENT_TIMESTAMP;
	`, sessionID).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("invalid session")
	}
	return userID, err
}
