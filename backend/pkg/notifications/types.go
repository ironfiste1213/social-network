package notifications

import "time"

const (
	TypeFollowRequest   = "follow_request"    // accept/decline via /follow/requests/{ref_id}/accept|decline
	TypeGroupInvite     = "group_invitation"  // accept/decline via /groups/invitations/{ref_id}/accept|decline
	TypeGroupJoinReq    = "group_join_request" // accept/decline via /groups/requests/{ref_id}/accept|decline
	TypeGroupEvent      = "group_event"        // info only — no action needed
)

type Notification struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Type      string     `json:"type"`
	RefID     string     `json:"ref_id,omitempty"` // follow_request_id | invitation_id | join_request_id
	Actor     *UserInfo  `json:"actor,omitempty"`
	Group     *GroupInfo `json:"group,omitempty"`
	Event     *EventInfo `json:"event,omitempty"`
	Read      bool       `json:"read"`
	CreatedAt time.Time  `json:"created_at"`
}

type UserInfo struct {
	ID         string `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Nickname   string `json:"nickname,omitempty"`
	AvatarPath string `json:"avatar_path,omitempty"`
}

type GroupInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type EventInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}