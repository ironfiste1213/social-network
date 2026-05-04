package events

import "time"

type GroupEvent struct {
	ID             string      `json:"id"`
	GroupID        string      `json:"group_id"`
	Creator        UserSummary `json:"creator"`
	Title          string      `json:"title"`
	Description    string      `json:"description"`
	EventTime      time.Time   `json:"event_time"`
	GoingCount     int         `json:"going_count"`
	NotGoingCount  int         `json:"not_going_count"`
	ViewerResponse string      `json:"viewer_response,omitempty"` // "going" | "not_going" | ""
	CreatedAt      time.Time   `json:"created_at"`
}

type UserSummary struct {
	ID         string `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Nickname   string `json:"nickname,omitempty"`
	AvatarPath string `json:"avatar_path,omitempty"`
}

type CreateEventInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	EventTime   string `json:"event_time"` // RFC3339
}

type RespondEventInput struct {
	Response string `json:"response"` // "going" | "not_going"
}

type GroupMember struct {
	User     UserSummary `json:"user"`
	Role     string      `json:"role"`
	JoinedAt time.Time   `json:"joined_at"`
}