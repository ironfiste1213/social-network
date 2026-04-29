package posts

import "time"

type Post struct {
	ID        string    `json:"id"`
	AuthorID  string    `json:"author_id"`
	GroupID   string    `json:"group_id,omitempty"`
	Author    *Author   `json:"author,omitempty"`
	Body      string    `json:"body"`
	ImagePath string    `json:"image_path,omitempty"`
	Privacy   string    `json:"privacy"`
	ViewerIDs []string  `json:"viewer_ids,omitempty"`
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

type CreatePostInput struct {
	Body      string   `json:"body"`
	Privacy   string   `json:"privacy"`
	GroupID   string   `json:"group_id,omitempty"`
	ImagePath string   `json:"image_path,omitempty"`
	ViewerIDs []string `json:"viewer_ids,omitempty"`
}
