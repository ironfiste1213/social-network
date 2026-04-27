package events

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("event not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// GetMembershipRole returns the role of userID in groupID, and whether they are a member.
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

// CreateEvent inserts a new group event.
func (r *Repository) CreateEvent(ctx context.Context, groupID, creatorID string, input CreateEventInput, eventTime interface{}) (GroupEvent, error) {
	id := uuid.NewString()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO group_events (id, group_id, creator_id, title, description, event_time)
		VALUES (?, ?, ?, ?, ?, ?);
	`, id, groupID, creatorID, input.Title, input.Description, eventTime)
	if err != nil {
		return GroupEvent{}, err
	}

	return r.GetEventByID(ctx, id, creatorID)
}

// GetEventByID fetches a single event with counts and viewer response.
func (r *Repository) GetEventByID(ctx context.Context, eventID, viewerID string) (GroupEvent, error) {
	var e GroupEvent
	var creator UserSummary
	var viewerResponse sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT
			ge.id,
			ge.group_id,
			ge.title,
			ge.description,
			ge.event_time,
			ge.created_at,
			u.id,
			u.first_name,
			u.last_name,
			COALESCE(u.nickname, ''),
			COALESCE(u.avatar_path, ''),
			(SELECT COUNT(1) FROM event_responses er WHERE er.event_id = ge.id AND er.response = 'going') AS going_count,
			(SELECT COUNT(1) FROM event_responses er WHERE er.event_id = ge.id AND er.response = 'not_going') AS not_going_count,
			(SELECT er2.response FROM event_responses er2 WHERE er2.event_id = ge.id AND er2.user_id = ?) AS viewer_response
		FROM group_events ge
		JOIN users u ON u.id = ge.creator_id
		WHERE ge.id = ?;
	`, viewerID, eventID).Scan(
		&e.ID,
		&e.GroupID,
		&e.Title,
		&e.Description,
		&e.EventTime,
		&e.CreatedAt,
		&creator.ID,
		&creator.FirstName,
		&creator.LastName,
		&creator.Nickname,
		&creator.AvatarPath,
		&e.GoingCount,
		&e.NotGoingCount,
		&viewerResponse,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return GroupEvent{}, ErrNotFound
	}
	if err != nil {
		return GroupEvent{}, err
	}

	e.Creator = creator
	if viewerResponse.Valid {
		e.ViewerResponse = viewerResponse.String
	}
	return e, nil
}

// ListEvents returns all events for a group, newest first.
func (r *Repository) ListEvents(ctx context.Context, groupID, viewerID string) ([]GroupEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			ge.id,
			ge.group_id,
			ge.title,
			ge.description,
			ge.event_time,
			ge.created_at,
			u.id,
			u.first_name,
			u.last_name,
			COALESCE(u.nickname, ''),
			COALESCE(u.avatar_path, ''),
			(SELECT COUNT(1) FROM event_responses er WHERE er.event_id = ge.id AND er.response = 'going') AS going_count,
			(SELECT COUNT(1) FROM event_responses er WHERE er.event_id = ge.id AND er.response = 'not_going') AS not_going_count,
			(SELECT er2.response FROM event_responses er2 WHERE er2.event_id = ge.id AND er2.user_id = ?) AS viewer_response
		FROM group_events ge
		JOIN users u ON u.id = ge.creator_id
		WHERE ge.group_id = ?
		ORDER BY ge.event_time ASC;
	`, viewerID, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []GroupEvent
	for rows.Next() {
		var e GroupEvent
		var creator UserSummary
		var viewerResponse sql.NullString

		if err := rows.Scan(
			&e.ID,
			&e.GroupID,
			&e.Title,
			&e.Description,
			&e.EventTime,
			&e.CreatedAt,
			&creator.ID,
			&creator.FirstName,
			&creator.LastName,
			&creator.Nickname,
			&creator.AvatarPath,
			&e.GoingCount,
			&e.NotGoingCount,
			&viewerResponse,
		); err != nil {
			return nil, err
		}
		e.Creator = creator
		if viewerResponse.Valid {
			e.ViewerResponse = viewerResponse.String
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// UpsertResponse inserts or replaces a viewer's RSVP for an event.
func (r *Repository) UpsertResponse(ctx context.Context, eventID, userID, response string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO event_responses (event_id, user_id, response, responded_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(event_id, user_id) DO UPDATE SET
			response = excluded.response,
			responded_at = excluded.responded_at;
	`, eventID, userID, response)
	return err
}

// EventBelongsToGroup verifies an event is in the given group.
func (r *Repository) EventBelongsToGroup(ctx context.Context, eventID, groupID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM group_events WHERE id = ? AND group_id = ?;
	`, eventID, groupID).Scan(&count)
	return count > 0, err
}

// GetGroupMemberIDs returns all user IDs that are members of the group.
func (r *Repository) GetGroupMemberIDs(ctx context.Context, groupID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT user_id FROM group_members WHERE group_id = ?;
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetUserBySessionID resolves a session cookie to a user ID.
func (r *Repository) GetUserBySessionID(ctx context.Context, sessionID string) (string, error) {
	var userID string
	err := r.db.QueryRowContext(ctx, `
		SELECT user_id FROM sessions WHERE id = ? AND expires_at > CURRENT_TIMESTAMP;
	`, sessionID).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("invalid session")
	}
	return userID, err
}