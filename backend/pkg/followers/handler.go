package followers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type Handler struct {
	service   *Service
	
}
var ErrUserNotFound = errors.New("user not found")

const sessionCookieName = "session_id"
func NewHandler(db *sql.DB) *Handler {
	repo := NewRepository(db)
	service := NewService(repo)
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/follow/", h.routeFollow)
	mux.HandleFunc("/follow/requests", h.handleListRequests)
}

func (h *Handler) routeFollow(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/follow/")
	if strings.HasPrefix(path, "requests/") {
		h.handleRespondRequest(w, r)
		return
	}
	h.handleFollow(w, r, path)
}

// POST /follow/{targetID} → follow
// DELETE /follow/{targetID} → unfollow
func (h *Handler) handleFollow(w http.ResponseWriter, r *http.Request, targetID string) {
	if targetID == "" {
		writeError(w, http.StatusBadRequest, "missing target user id")
		return
	}
	viewerID := h.getUserID(r, w)
	
	switch r.Method {
	case http.MethodPost:
		err := h.service.Follow(r.Context(), viewerID, targetID)
		switch {
		case errors.Is(err, ErrCannotFollowSelf):
			writeError(w, http.StatusBadRequest, "cannot follow yourself")
		case errors.Is(err, ErrAlreadyFollowing):
			writeError(w, http.StatusConflict, "already following")
		case errors.Is(err, ErrRequestAlreadyExists):
			writeError(w, http.StatusConflict, "follow request already sent")
		case errors.Is(err, ErrNotFound):
			writeError(w, http.StatusNotFound, "user not found")
		case err != nil:
			writeError(w, http.StatusInternalServerError, "failed to follow")
		default:
			writeJSON(w, http.StatusOK, map[string]string{"message": "ok"})
		}

	case http.MethodDelete:
		if err := h.service.Unfollow(r.Context(), viewerID, targetID); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to unfollow")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "ok"})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// GET /follow/requests
func (h *Handler) handleListRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	viewerID := h.getUserID(r, w)
	
	reqs, err := h.service.GetPendingRequests(r.Context(), viewerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get requests")
		return
	}
	if reqs == nil {
		reqs = []FollowRequest{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": reqs})
}

// POST /follow/requests/{requestID}/accept|decline
func (h *Handler) handleRespondRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	viewerID := h.getUserID(r, w)
	
	path := strings.TrimPrefix(r.URL.Path, "/follow/requests/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	requestID, action := parts[0], parts[1]

	switch action {
	case "accept":
		if err := h.service.AcceptRequest(r.Context(), requestID, viewerID); err != nil {
			if errors.Is(err, ErrNotFound) {
				writeError(w, http.StatusNotFound, "request not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to accept")
			return
		}
	case "decline":
		if err := h.service.DeclineRequest(r.Context(), requestID, viewerID); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to decline")
			return
		}
	default:
		writeError(w, http.StatusBadRequest, "action must be accept or decline")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

// HandleUserFollowRoutes is called by the users handler for sub-routes:
//
//	GET /users/{id}/followers
//	GET /users/{id}/following
//	GET /users/{id}/follow-status
func (h *Handler) HandleUserFollowRoutes(w http.ResponseWriter, r *http.Request, targetID, sub string) bool {
	if r.Method != http.MethodGet {
		return false
	}
	switch sub {
	case "followers":
		list, err := h.service.GetFollowers(r.Context(), targetID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed")
			return true
		}
		if list == nil {
			list = []UserSummary{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"followers": list})
		return true

	case "following":
		list, err := h.service.GetFollowing(r.Context(), targetID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed")
			return true
		}
		if list == nil {
			list = []UserSummary{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"following": list})
		return true

	case "follow-status":
		viewerID := h.getUserID(r, w)
		if viewerID == "" {return true}
		status, err := h.service.GetFollowStatus(r.Context(), viewerID, targetID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed")
			return true
		}
		writeJSON(w, http.StatusOK, status)
		return true
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}


func (h *Handler) getUserID(r *http.Request, w http.ResponseWriter) (string) {
cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return ""
	}
	viewerID, err := h.service.currentUserID(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return ""
		}
		writeError(w, http.StatusInternalServerError, "failed to get current user")
		return ""
	}
	return viewerID
}