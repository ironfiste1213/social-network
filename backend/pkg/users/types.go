package users

import "time"

// User represents a user in the system
type User struct {
	ID                string    `json:"id"`
	Email             string    `json:"email"`
	FirstName         string    `json:"first_name"`
	LastName          string    `json:"last_name"`
	DateOfBirth       time.Time `json:"date_of_birth"`
	AvatarPath        string    `json:"avatar_path,omitempty"`
	Nickname          string    `json:"nickname,omitempty"`
	AboutMe           string    `json:"about_me,omitempty"`
	ProfileVisibility string    `json:"profile_visibility"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// UpdateInput represents the input for updating a user profile
type UpdateInput struct {
	Nickname          *string `json:"nickname"`
	AboutMe           *string `json:"about_me"`
	AvatarPath        *string `json:"avatar_path"`
	ProfileVisibility *string `json:"profile_visibility"`
}

// SearchResult represents a user search result
type SearchResult struct {
	ID                string `json:"id"`
	FirstName         string `json:"first_name"`
	LastName          string `json:"last_name"`
	Nickname          string `json:"nickname"`
	AvatarPath        string `json:"avatar_path,omitempty"`
	ProfileVisibility string `json:"profile_visibility"`
}
