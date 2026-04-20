package users

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const sessionCookieName = "session_id"
const maxAvatarSize = 5 << 20 // 5MB

type Handler struct {
	service *Service
	uploadDir string
}

func NewHandler(db *sql.DB, uploadDir string) *Handler {
	repo := NewRepository(db)
	service := NewService(repo)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		fmt.Println("[USERS] Warning: could not create upload dir:", err)
	}
	return &Handler{service: service, uploadDir: uploadDir}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/users/me", h.handleMe)
	mux.HandleFunc("/users/me/avatar", h.handleAvatar)
	mux.HandleFunc("/users/", h.handleUserByID)
}

// GET /users/me — returns own profile
// PATCH /users/me — updates own profile
func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	user, err := h.authenticate(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"user": user})
	case http.MethodPatch:
		h.handleUpdateMe(w, r, user.ID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleUpdateMe(w http.ResponseWriter, r *http.Request, userID string) {
	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	updated, err := h.service.UpdateProfile(r.Context(), userID, input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"user": updated})
}

// POST /users/me/avatar — upload avatar image
func (h *Handler) handleAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user, err := h.authenticate(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarSize)
	if err := r.ParseMultipartForm(maxAvatarSize); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing avatar file")
		return
	}
	defer file.Close()

	// Validate content type
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true}
	if !allowed[ext] {
		writeError(w, http.StatusBadRequest, "only jpg, png, gif allowed")
		return
	}

	// Save file
	filename := uuid.NewString() + ext
	dest := filepath.Join(h.uploadDir, filename)
	out, err := os.Create(dest)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save file")
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write file")
		return
	}

	avatarPath := "/uploads/" + filename
	updated, err := h.service.UpdateProfile(r.Context(), user.ID, UpdateInput{
		AvatarPath: &avatarPath,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update avatar")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"user": updated, "avatar_path": avatarPath})
}

// GET /users/:id — get any user's profile
func (h *Handler) handleUserByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract ID from path: /users/{id}
	id := strings.TrimPrefix(r.URL.Path, "/users/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	// Try to get the requesting user (optional auth)
	requester, _ := h.authenticate(r)

	profile, err := h.service.GetUserByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	// If profile is private and requester is not the owner, check if they follow
	if profile.ProfileVisibility == "private" {
		isOwner := requester != nil && requester.ID == profile.ID
		if !isOwner {
			// For now return limited info — follower check comes in Day 5
			writeJSON(w, http.StatusOK, map[string]any{
				"user": map[string]any{
					"id":                 profile.ID,
					"first_name":         profile.FirstName,
					"last_name":          profile.LastName,
					"profile_visibility": profile.ProfileVisibility,
					"is_private":         true,
				},
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"user": profile})
}

func (h *Handler) authenticate(r *http.Request) (*User, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, err
	}
	user, err := h.service.GetUserBySession(r.Context(), cookie.Value)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// Serve uploaded files
func ServeUploads(uploadDir string) http.Handler {
	return http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir)))
}

// Types used only in handler
type UpdateInput struct {
	Nickname          *string `json:"nickname"`
	AboutMe           *string `json:"about_me"`
	AvatarPath        *string `json:"avatar_path"`
	ProfileVisibility *string `json:"profile_visibility"`
}

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