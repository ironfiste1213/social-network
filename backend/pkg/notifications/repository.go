package notifications

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) getUserBySessionID(ctx context.Context, sessionID string) (string, error) {
	var userID string
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id FROM sessions WHERE id = ? AND expires_at > CURRENT_TIMESTAMP;`,
		sessionID,
	).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("invalid session")
	}
	return userID, err
}

func (r *Repository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE notifications SET read = 1 WHERE user_id = ?;`,
		userID,
	)
	return err
}
 
func (r *Repository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM notifications WHERE user_id = ? AND read = 0;`,
		userID,
	).Scan(&count)
	return count, err
}
 
func (r *Repository) MarkRead(ctx context.Context, notifID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE notifications SET read = 1 WHERE id = ? AND user_id = ?;`,
		notifID, userID,
	)
	return err
}

// Create inserts a notification. refID ties it to the actionable item.
func (r *Repository) Create(ctx context.Context, userID, notifType, refID, actorID, groupID, eventID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO notifications (id, user_id, type, ref_id, actor_id, group_id, event_id)
		VALUES (?, ?, ?, ?, ?, ?, ?);
	`,
		uuid.NewString(),
		userID,
		notifType,
		nullIfEmpty(refID),
		nullIfEmpty(actorID),
		nullIfEmpty(groupID),
		nullIfEmpty(eventID),
	)
	return err
}
 
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}


func (r *Repository) GetAll(ctx context.Context, userID string) ([]Notification, error) {
	return r.query(ctx, userID, true)
}
 
func (r *Repository) GetUnread(ctx context.Context, userID string) ([]Notification, error) {
	return r.query(ctx, userID, false)
}
 
func (r *Repository) query(ctx context.Context, userID string, includeRead bool) ([]Notification, error) {
	q := `
		SELECT n.id, n.user_id, n.type, COALESCE(n.ref_id,''), n.read, n.created_at,
		       COALESCE(u.id,''), COALESCE(u.first_name,''), COALESCE(u.last_name,''),
		       COALESCE(u.nickname,''), COALESCE(u.avatar_path,''),
		       COALESCE(n.group_id,''), COALESCE(g.title,''),
		       COALESCE(n.event_id,''), COALESCE(ge.title,'')
		FROM notifications n
		LEFT JOIN users u ON u.id = n.actor_id
		LEFT JOIN groups g ON g.id = n.group_id
		LEFT JOIN group_events ge ON ge.id = n.event_id
		WHERE n.user_id = ?`
 
	if !includeRead {
		q += ` AND n.read = 0`
	}
	q += ` ORDER BY n.created_at DESC;`
 
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
 
	var notifs []Notification
	for rows.Next() {
		var n Notification
		var actorID, actorFirst, actorLast, actorNick, actorAvatar string
		var groupID, groupTitle string
		var eventID, eventTitle string
 
		if err := rows.Scan(
			&n.ID, &n.UserID, &n.Type, &n.RefID, &n.Read, &n.CreatedAt,
			&actorID, &actorFirst, &actorLast, &actorNick, &actorAvatar,
			&groupID, &groupTitle,
			&eventID, &eventTitle,
		); err != nil {
			return nil, err
		}
 
		if actorID != "" {
			n.Actor = &UserInfo{
				ID: actorID, FirstName: actorFirst, LastName: actorLast,
				Nickname: actorNick, AvatarPath: actorAvatar,
			}
		}
		if groupID != "" {
			n.Group = &GroupInfo{ID: groupID, Title: groupTitle}
		}
		if eventID != "" {
			n.Event = &EventInfo{ID: eventID, Title: eventTitle}
		}
 
		notifs = append(notifs, n)
	}
	return notifs, rows.Err()
}