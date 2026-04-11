package auth

import "time"

type User struct {
	ID                string    `json:"id"`
	Email             string    `json:"email"`
	PasswordHash      string    `json:"-"`
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

type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type RegisterInput struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DateOfBirth string `json:"date_of_birth"`
	AvatarPath  string `json:"avatar_path"`
	Nickname    string `json:"nickname"`
	AboutMe     string `json:"about_me"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SafeUser struct {
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

func (u User) Safe() SafeUser {
	return SafeUser{
		ID:                u.ID,
		Email:             u.Email,
		FirstName:         u.FirstName,
		LastName:          u.LastName,
		DateOfBirth:       u.DateOfBirth,
		AvatarPath:        u.AvatarPath,
		Nickname:          u.Nickname,
		AboutMe:           u.AboutMe,
		ProfileVisibility: u.ProfileVisibility,
		CreatedAt:         u.CreatedAt,
		UpdatedAt:         u.UpdatedAt,
	}
}
