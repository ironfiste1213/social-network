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