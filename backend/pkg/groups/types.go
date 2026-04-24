package groups

import "time"

type Group struct {
	ID           string            `json:"id"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	CreatorID    string            `json:"creator_id"`
	Creator      *UserSummary      `json:"creator,omitempty"`
	MemberCount  int               `json:"member_count"`
	ViewerStatus GroupViewerStatus `json:"viewer_status"`
	CreatedAt    time.Time         `json:"created_at"`
}

type GroupViewerStatus struct {
	IsMember             bool   `json:"is_member"`
	Role                 string `json:"role,omitempty"`
	HasPendingJoin       bool   `json:"has_pending_join_request"`
	PendingJoinRequestID string `json:"pending_join_request_id,omitempty"`
	HasPendingInvite     bool   `json:"has_pending_invitation"`
	PendingInvitationID  string `json:"pending_invitation_id,omitempty"`
}

type GroupMember struct {
	User     UserSummary `json:"user"`
	Role     string      `json:"role"`
	JoinedAt time.Time   `json:"joined_at"`
}

type GroupJoinRequest struct {
	ID        string      `json:"id"`
	GroupID   string      `json:"group_id"`
	User      UserSummary `json:"user"`
	CreatedAt time.Time   `json:"created_at"`
}

type GroupInvitation struct {
	ID        string      `json:"id"`
	GroupID   string      `json:"group_id"`
	Group     *Group      `json:"group,omitempty"`
	Inviter   UserSummary `json:"inviter"`
	InviteeID string      `json:"invitee_id"`
	CreatedAt time.Time   `json:"created_at"`
}

type UserSummary struct {
	ID         string `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Nickname   string `json:"nickname,omitempty"`
	AvatarPath string `json:"avatar_path,omitempty"`
}

type CreateGroupInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type InviteToGroupInput struct {
	InviteeID string `json:"invitee_id"`
}
