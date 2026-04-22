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

// FollowRouteHandler is implemented by the followers.Handler
type FollowRouteHandler interface {
	HandleUserFollowRoutes(w http.ResponseWriter, r *http.Request, targetID, sub string) bool
}

type Handler struct {
	service         *Service
	uploadDir       string
	followersHandler FollowRouteHandler
}

func NewHandler(db *sql.DB, uploadDir string) *Handler {
	repo := NewRepository(db)
	service := NewService(repo)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		fmt.Println("[USERS] Warning: could not create upload dir:", err)
	}
	return &Handler{service: service, uploadDir: uploadDir}
}

// SetFollowersHandler wires in the followers handler for /users/{id}/followers etc.
func (h *Handler) SetFollowersHandler(fh FollowRouteHandler) {
	h.followersHandler = fh
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/users/me", h.handleMe)
	mux.HandleFunc("/users/me/avatar", h.handleAvatar)
	mux.HandleFunc("/users/", h.handleUserByID)
}

// GET /users/me  — own profile
// PATCH /users/me — update own profile
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

// POST /users/me/avatar
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

	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true}
	if !allowed[ext] {
		writeError(w, http.StatusBadRequest, "only jpg, png, gif allowed")
		return
	}

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

// GET /users/{id}               — public profile
// GET /users/{id}/followers     — follower list
// GET /users/{id}/following     — following list
// GET /users/{id}/follow-status — viewer → target relationship
func (h *Handler) handleUserByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	parts := strings.SplitN(path, "/", 2)

	targetID := parts[0]
	if targetID == "" || targetID == "me" {
		http.NotFound(w, r)
		return
	}

	// Sub-routes delegated to followers handler
	if len(parts) == 2 && h.followersHandler != nil {
		sub := parts[1]
		if h.followersHandler.HandleUserFollowRoutes(w, r, targetID, sub) {
			return
		}
	}

	// Main profile route
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	requester, _ := h.authenticate(r)

	profile, err := h.service.GetUserByID(r.Context(), targetID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	if profile.ProfileVisibility == "private" {
		isOwner := requester != nil && requester.ID == profile.ID
		if !isOwner {
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

// ServeUploads returns a handler that serves files from uploadDir under /uploads/
func ServeUploads(uploadDir string) http.Handler {
	return http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir)))
}

// Types used in handler
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