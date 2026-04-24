package comments

import "time"

type Comment struct {
	ID        string    `json:"id"`
	PostID    string    `json:"post_id"`
	AuthorID  string    `json:"author_id"`
	Author    *Author   `json:"author,omitempty"`
	Body      string    `json:"body"`
	ImagePath string    `json:"image_path,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Author struct {
	ID         string `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Nickname   string `json:"nickname,omitempty"`
	AvatarPath string `json:"avatar_path,omitempty"`
}

type CreateCommentInput struct {
	Body      string `json:"body"`
	ImagePath string `json:"image_path,omitempty"`
}