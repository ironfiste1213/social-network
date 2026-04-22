package posts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("post not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreatePost(ctx context.Context, authorID string, input CreatePostInput) (Post, error) {
	id := uuid.NewString()
	privacy := input.Privacy
	if privacy != "public" && privacy != "followers" && privacy != "selected_followers" {
		privacy = "public"
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO posts (id, author_id, body, image_path, privacy)
		VALUES (?, ?, ?, ?, ?);
	`, id, authorID, strings.TrimSpace(input.Body), nullIfEmpty(input.ImagePath), privacy)
	if err != nil {
		return Post{}, fmt.Errorf("create post: %w", err)
	}

	if privacy == "selected_followers" && len(input.ViewerIDs) > 0 {
		for _, uid := range input.ViewerIDs {
			_, err := r.db.ExecContext(ctx,
				`INSERT OR IGNORE INTO post_viewers (post_id, user_id) VALUES (?, ?);`,
				id, uid,
			)
			if err != nil {
				return Post{}, fmt.Errorf("add post viewer: %w", err)
			}
		}
	}

	return r.GetPostByID(ctx, id)
}

func (r *Repository) GetPostByID(ctx context.Context, id string) (Post, error) {
	var p Post
	var imagePath sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT p.id, p.author_id, p.body, p.image_path, p.privacy, p.created_at, p.updated_at,
		       u.first_name, u.last_name, COALESCE(u.nickname,''), COALESCE(u.avatar_path,'')
		FROM posts p
		JOIN users u ON u.id = p.author_id
		WHERE p.id = ?;
	`, id).Scan(
		&p.ID, &p.AuthorID, &p.Body, &imagePath, &p.Privacy, &p.CreatedAt, &p.UpdatedAt,
		new(string), new(string), new(string), new(string),
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Post{}, ErrNotFound
	}
	if err != nil {
		return Post{}, err
	}
	if imagePath.Valid {
		p.ImagePath = imagePath.String
	}
	return r.attachAuthorAndViewers(ctx, p)
}

func (r *Repository) GetFeedPosts(ctx context.Context, viewerID string, limit, offset int) ([]Post, error) {
	// Feed = public posts from everyone + followers-only/selected posts from people viewer follows
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT p.id, p.author_id, p.body, p.image_path, p.privacy, p.created_at, p.updated_at,
		       u.first_name, u.last_name, COALESCE(u.nickname,''), COALESCE(u.avatar_path,'')
		FROM posts p
		JOIN users u ON u.id = p.author_id
		WHERE
		    p.author_id = ?
		    OR p.privacy = 'public'
		    OR (p.privacy = 'followers' AND EXISTS (
		        SELECT 1 FROM followers f WHERE f.follower_id = ? AND f.following_id = p.author_id
		    ))
		    OR (p.privacy = 'selected_followers' AND EXISTS (
		        SELECT 1 FROM post_viewers pv WHERE pv.post_id = p.id AND pv.user_id = ?
		    ))
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?;
	`, viewerID, viewerID, viewerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanPostRows(ctx, rows)
}

func (r *Repository) GetUserPosts(ctx context.Context, authorID, viewerID string, limit, offset int) ([]Post, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT p.id, p.author_id, p.body, p.image_path, p.privacy, p.created_at, p.updated_at,
		       u.first_name, u.last_name, COALESCE(u.nickname,''), COALESCE(u.avatar_path,'')
		FROM posts p
		JOIN users u ON u.id = p.author_id
		WHERE p.author_id = ?
		  AND (
		      p.author_id = ?
		      OR p.privacy = 'public'
		      OR (p.privacy = 'followers' AND EXISTS (
		          SELECT 1 FROM followers f WHERE f.follower_id = ? AND f.following_id = p.author_id
		      ))
		      OR (p.privacy = 'selected_followers' AND EXISTS (
		          SELECT 1 FROM post_viewers pv WHERE pv.post_id = p.id AND pv.user_id = ?
		      ))
		  )
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?;
	`, authorID, viewerID, viewerID, viewerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanPostRows(ctx, rows)
}

func (r *Repository) DeletePost(ctx context.Context, id, authorID string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM posts WHERE id = ? AND author_id = ?;`, id, authorID,
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

func (r *Repository) UpdateImagePath(ctx context.Context, postID, authorID, imagePath string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE posts SET image_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND author_id = ?;`,
		imagePath, postID, authorID,
	)
	return err
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

// GetFollowersOfUser returns the list of user IDs that follow the given user (for selected_followers picker)
func (r *Repository) GetFollowersOfUser(ctx context.Context, userID string) ([]FollowerSummary, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.first_name, u.last_name, COALESCE(u.nickname,''), COALESCE(u.avatar_path,'')
		FROM followers f
		JOIN users u ON u.id = f.follower_id
		WHERE f.following_id = ?
		ORDER BY u.first_name, u.last_name;
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []FollowerSummary
	for rows.Next() {
		var fs FollowerSummary
		if err := rows.Scan(&fs.ID, &fs.FirstName, &fs.LastName, &fs.Nickname, &fs.AvatarPath); err != nil {
			return nil, err
		}
		result = append(result, fs)
	}
	return result, rows.Err()
}

// --- helpers ---

func (r *Repository) scanPostRows(ctx context.Context, rows *sql.Rows) ([]Post, error) {
	var posts []Post
	for rows.Next() {
		var p Post
		var a Author
		var imagePath sql.NullString
		if err := rows.Scan(
			&p.ID, &p.AuthorID, &p.Body, &imagePath, &p.Privacy, &p.CreatedAt, &p.UpdatedAt,
			&a.FirstName, &a.LastName, &a.Nickname, &a.AvatarPath,
		); err != nil {
			return nil, err
		}
		a.ID = p.AuthorID
		p.Author = &a
		if imagePath.Valid {
			p.ImagePath = imagePath.String
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// attach viewer IDs for selected_followers posts
	for i, p := range posts {
		if p.Privacy == "selected_followers" {
			vids, err := r.getViewerIDs(ctx, p.ID)
			if err != nil {
				return nil, err
			}
			posts[i].ViewerIDs = vids
		}
	}
	return posts, nil
}

func (r *Repository) attachAuthorAndViewers(ctx context.Context, p Post) (Post, error) {
	var a Author
	err := r.db.QueryRowContext(ctx, `
		SELECT first_name, last_name, COALESCE(nickname,''), COALESCE(avatar_path,'')
		FROM users WHERE id = ?;
	`, p.AuthorID).Scan(&a.FirstName, &a.LastName, &a.Nickname, &a.AvatarPath)
	if err != nil {
		return p, err
	}
	a.ID = p.AuthorID
	p.Author = &a

	if p.Privacy == "selected_followers" {
		vids, err := r.getViewerIDs(ctx, p.ID)
		if err != nil {
			return p, err
		}
		p.ViewerIDs = vids
	}
	return p, nil
}

func (r *Repository) getViewerIDs(ctx context.Context, postID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT user_id FROM post_viewers WHERE post_id = ?;`, postID)
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

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

type FollowerSummary struct {
	ID         string `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Nickname   string `json:"nickname,omitempty"`
	AvatarPath string `json:"avatar_path,omitempty"`
}

// suppress unused import warning
var _ = fmt.Sprintf
