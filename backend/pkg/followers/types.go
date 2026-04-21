package followers

import "time"

type FollowRequest struct {
	ID        string      `json:"id"`
	Sender    UserSummary `json:"sender"`
	CreatedAt time.Time   `json:"created_at"`
}

// FollowStatus describes the relationship from viewer → target
type FollowStatus struct {
	IsFollowing       bool   `json:"is_following"`
	HasPendingRequest bool   `json:"has_pending_request"`
	RequestID         string `json:"request_id,omitempty"`
}



type UserSummary struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Nickname  string `json:"nickname,omitempty"`
	AvatarPath string `json:"avatar_path,omitempty"`
}