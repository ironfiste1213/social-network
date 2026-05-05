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
	"strconv"
	"strings"

	"social-network/backend/pkg/response"
	"social-network/backend/pkg/sessionauth"

	"github.com/google/uuid"
)

const maxAvatarSize = 5 << 20 // 5MB

// FollowRouteHandler is implemented by the followers.Handler
type FollowRouteHandler interface {
	HandleUserFollowRoutes(w http.ResponseWriter, r *http.Request, targetID, sub string) bool
}

type PostRouteHandler interface {
	HandleUserPostRoutes(w http.ResponseWriter, r *http.Request, targetID, sub string) bool
}

type Handler struct {
	service          *Service
	uploadDir        string
	followersHandler FollowRouteHandler
	postsHandler     PostRouteHandler
}

func NewHandler(db *sql.DB, uploadDir string) *Handler {
	repo := NewRepository(db)
	service := NewService(repo)
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		fmt.Println("[USERS] Warning: could not create upload dir:", err)
	}
	return &Handler{service: service, uploadDir: uploadDir}
}

// SetFollowersHandler wires in the followers handler for /users/{id}/followers etc.
func (h *Handler) SetFollowersHandler(fh FollowRouteHandler) {
	h.followersHandler = fh
}

// SetPostsHandler wires in the posts handler for /users/{id}/posts.
func (h *Handler) SetPostsHandler(ph PostRouteHandler) {
	h.postsHandler = ph
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/users/me", h.handleMe)
	mux.HandleFunc("/users/me/avatar", h.handleAvatar)
	mux.HandleFunc("/users/search", h.handleSearchUsers)
	mux.HandleFunc("/users/", h.handleUserByID)
}

// GET /users/me  — own profile
// PATCH /users/me — update own profile
func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	user, err := h.authenticate(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	switch r.Method {
	case http.MethodGet:
		response.JSON(w, http.StatusOK, map[string]any{"user": user})
	case http.MethodPatch:
		h.handleUpdateMe(w, r, user.ID)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleUpdateMe(w http.ResponseWriter, r *http.Request, userID string) {
	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	updated, err := h.service.UpdateProfile(r.Context(), userID, input)
	if err != nil {
		if errors.Is(err, ErrNicknameAlreadyExists) {
			response.Error(w, http.StatusConflict, "nickname already exists")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to update profile")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"user": updated})
}

// POST /users/me/avatar
func (h *Handler) handleAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	user, err := h.authenticate(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarSize)
	if err := r.ParseMultipartForm(maxAvatarSize); err != nil {
		response.Error(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "missing avatar file")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true}
	if !allowed[ext] {
		response.Error(w, http.StatusBadRequest, "only jpg, png, gif allowed")
		return
	}

	filename := uuid.NewString() + ext
	dest := filepath.Join(h.uploadDir, filename)
	out, err := os.Create(dest)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to save file")
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to write file")
		return
	}

	avatarPath := "/uploads/" + filename
	updated, err := h.service.UpdateProfile(r.Context(), user.ID, UpdateInput{
		AvatarPath: &avatarPath,
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to update avatar")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"user": updated, "avatar_path": avatarPath})
}

func (h *Handler) handleSearchUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user, err := h.authenticate(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	limit := 0
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = parsedLimit
	}

	results, err := h.service.SearchUsers(r.Context(), user.ID, r.URL.Query().Get("q"), limit)
	if err != nil {
		if errors.Is(err, ErrInvalidSearchQuery) {
			response.Error(w, http.StatusBadRequest, "search query must be at least 2 characters")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to search users")
		return
	}

	if results == nil {
		results = []SearchResult{}
	}

	response.JSON(w, http.StatusOK, map[string]any{"users": results})
}

// GET /users/{id}               — public profile
// GET /users/{id}/followers     — follower list
// GET /users/{id}/following     — following list
// GET /users/{id}/follow-status — viewer → target relationship
func (h *Handler) handleUserByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	parts := strings.SplitN(path, "/", 2)

	targetID := parts[0]
	if targetID == "" {
		http.NotFound(w, r)
		return
	}

	if targetID == "me" {
		requester, err := h.authenticate(r)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		targetID = requester.ID
	}

	// Sub-routes delegated to followers handler
	if len(parts) == 2 && h.followersHandler != nil {
		sub := parts[1]
		if h.followersHandler.HandleUserFollowRoutes(w, r, targetID, sub) {
			return
		}
	}

	if len(parts) == 2 && h.postsHandler != nil {
		sub := parts[1]
		if h.postsHandler.HandleUserPostRoutes(w, r, targetID, sub) {
			return
		}
	}

	// Main profile route
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	requester, _ := h.authenticate(r)

	profile, err := h.service.GetUserByID(r.Context(), targetID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.Error(w, http.StatusNotFound, "user not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	if profile.ProfileVisibility == "private" {
		isOwner := requester != nil && requester.ID == profile.ID
		canView := isOwner
		if !canView && requester != nil {
			following, err := h.service.IsFollowing(r.Context(), requester.ID, profile.ID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, "failed to get user")
				return
			}
			canView = following
		}

		if !canView {
			response.JSON(w, http.StatusOK, map[string]any{
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

	response.JSON(w, http.StatusOK, map[string]any{"user": profile})
}

func (h *Handler) authenticate(r *http.Request) (*User, error) {
	sessionID, err := sessionauth.SessionIDFromRequest(r)
	if err != nil {
		return nil, err
	}
	user, err := h.service.GetUserBySession(r.Context(), sessionID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ServeUploads returns a handler that serves files from uploadDir under /uploads/
func ServeUploads(uploadDir string) http.Handler {
	return http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir)))
}
