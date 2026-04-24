package comments

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("comment not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateComment(ctx context.Context, postID, authorID string, input CreateCommentInput) (Comment, error) {
	id := uuid.NewString()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO comments (id, post_id, author_id, body, image_path)
		VALUES (?, ?, ?, ?, ?);
	`, id, postID, authorID, input.Body, nullIfEmpty(input.ImagePath))
	if err != nil {
		return Comment{}, err
	}

	return r.GetCommentByID(ctx, id)
}

func (r *Repository) GetCommentByID(ctx context.Context, id string) (Comment, error) {
	var c Comment
	var a Author
	var imagePath sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT c.id, c.post_id, c.author_id, c.body, c.image_path, c.created_at, c.updated_at,
		       u.first_name, u.last_name, COALESCE(u.nickname,''), COALESCE(u.avatar_path,'')
		FROM comments c
		JOIN users u ON u.id = c.author_id
		WHERE c.id = ?;
	`, id).Scan(
		&c.ID, &c.PostID, &c.AuthorID, &c.Body, &imagePath, &c.CreatedAt, &c.UpdatedAt,
		&a.FirstName, &a.LastName, &a.Nickname, &a.AvatarPath,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Comment{}, ErrNotFound
	}
	if err != nil {
		return Comment{}, err
	}

	a.ID = c.AuthorID
	c.Author = &a
	if imagePath.Valid {
		c.ImagePath = imagePath.String
	}
	return c, nil
}

func (r *Repository) GetCommentsByPostID(ctx context.Context, postID string) ([]Comment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.post_id, c.author_id, c.body, c.image_path, c.created_at, c.updated_at,
		       u.first_name, u.last_name, COALESCE(u.nickname,''), COALESCE(u.avatar_path,'')
		FROM comments c
		JOIN users u ON u.id = c.author_id
		WHERE c.post_id = ?
		ORDER BY c.created_at ASC;
	`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		var a Author
		var imagePath sql.NullString

		if err := rows.Scan(
			&c.ID, &c.PostID, &c.AuthorID, &c.Body, &imagePath, &c.CreatedAt, &c.UpdatedAt,
			&a.FirstName, &a.LastName, &a.Nickname, &a.AvatarPath,
		); err != nil {
			return nil, err
		}

		a.ID = c.AuthorID
		c.Author = &a
		if imagePath.Valid {
			c.ImagePath = imagePath.String
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (r *Repository) DeleteComment(ctx context.Context, commentID, authorID string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM comments WHERE id = ? AND author_id = ?;`,
		commentID, authorID,
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

func (r *Repository) UpdateImagePath(ctx context.Context, commentID, authorID, imagePath string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE comments SET image_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND author_id = ?;`,
		imagePath, commentID, authorID,
	)
	return err
}

// CanViewPost checks whether viewerID is allowed to see a post (respects privacy).
func (r *Repository) CanViewPost(ctx context.Context, postID, viewerID string) (bool, error) {
	var privacy string
	var authorID string
	var groupID sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT author_id, privacy, group_id FROM posts WHERE id = ?;`, postID,
	).Scan(&authorID, &privacy, &groupID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if groupID.Valid && groupID.String != "" {
		var count int
		err := r.db.QueryRowContext(ctx, `
			SELECT COUNT(1)
			FROM group_members
			WHERE group_id = ? AND user_id = ?;
		`, groupID.String, viewerID).Scan(&count)
		return count > 0, err
	}

	// Owner can always view
	if authorID == viewerID {
		return true, nil
	}

	switch privacy {
	case "public":
		return true, nil
	case "followers":
		var count int
		err := r.db.QueryRowContext(ctx,
			`SELECT COUNT(1) FROM followers WHERE follower_id = ? AND following_id = ?`,
			viewerID, authorID,
		).Scan(&count)
		return count > 0, err
	case "selected_followers":
		var count int
		err := r.db.QueryRowContext(ctx,
			`SELECT COUNT(1) FROM post_viewers WHERE post_id = ? AND user_id = ?`,
			postID, viewerID,
		).Scan(&count)
		return count > 0, err
	}
	return false, nil
}

func (r *Repository) GetUserBySessionID(ctx context.Context, sessionID string) (string, error) {
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

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
