package followers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type Handler struct {
	service *Service
}

var ErrUserNotFound = errors.New("user not found")

const sessionCookieName = "session_id"

func NewHandler(db *sql.DB) *Handler {
	repo := NewRepository(db)
	service := NewService(repo)
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	fmt.Println("[FOLLOWERS][HANDLER] registering follow routes")
	mux.HandleFunc("/follow/", h.routeFollow)
	mux.HandleFunc("/follow/requests", h.handleListRequests)
}

func (h *Handler) routeFollow(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/follow/")
	fmt.Println("[FOLLOWERS][HANDLER] route follow path:", r.URL.Path, "trimmed:", path, "method:", r.Method)
	if strings.HasPrefix(path, "requests/") {
		fmt.Println("[FOLLOWERS][HANDLER] route follow dispatch: respond request")
		h.handleRespondRequest(w, r)
		return
	}
	fmt.Println("[FOLLOWERS][HANDLER] route follow dispatch: follow target")
	h.handleFollow(w, r, path)
}

// POST /follow/{targetID} → follow
// DELETE /follow/{targetID} → unfollow
func (h *Handler) handleFollow(w http.ResponseWriter, r *http.Request, targetID string) {
	if targetID == "" {
		fmt.Println("[FOLLOWERS][HANDLER] follow rejected: missing target user id")
		writeError(w, http.StatusBadRequest, "missing target user id")
		return
	}
	viewerID := h.getUserID(r, w)
	if viewerID == "" {
		fmt.Println("[FOLLOWERS][HANDLER] follow aborted: missing authenticated viewer for target:", targetID)
		return
	}

	fmt.Println("[FOLLOWERS][HANDLER] follow route:", r.Method, "viewer:", viewerID, "target:", targetID)

	switch r.Method {
	case http.MethodPost:
		err := h.service.Follow(r.Context(), viewerID, targetID)
		switch {
		case errors.Is(err, ErrCannotFollowSelf):
			fmt.Println("[FOLLOWERS][HANDLER] follow rejected: cannot follow self")
			writeError(w, http.StatusBadRequest, "cannot follow yourself")
		case errors.Is(err, ErrAlreadyFollowing):
			fmt.Println("[FOLLOWERS][HANDLER] follow rejected: already following")
			writeError(w, http.StatusConflict, "already following")
		case errors.Is(err, ErrRequestAlreadyExists):
			fmt.Println("[FOLLOWERS][HANDLER] follow rejected: request already exists")
			writeError(w, http.StatusConflict, "follow request already sent")
		case errors.Is(err, ErrNotFound):
			fmt.Println("[FOLLOWERS][HANDLER] follow rejected: target user not found")
			writeError(w, http.StatusNotFound, "user not found")
		case err != nil:
			fmt.Println("[FOLLOWERS][HANDLER] follow failed:", err)
			writeError(w, http.StatusInternalServerError, "failed to follow")
		default:
			fmt.Println("[FOLLOWERS][HANDLER] follow success")
			writeJSON(w, http.StatusOK, map[string]string{"message": "ok"})
		}

	case http.MethodDelete:
		if err := h.service.Unfollow(r.Context(), viewerID, targetID); err != nil {
			fmt.Println("[FOLLOWERS][HANDLER] unfollow failed:", err)
			writeError(w, http.StatusInternalServerError, "failed to unfollow")
			return
		}
		fmt.Println("[FOLLOWERS][HANDLER] unfollow success")
		writeJSON(w, http.StatusOK, map[string]string{"message": "ok"})

	default:
		fmt.Println("[FOLLOWERS][HANDLER] follow route rejected: method not allowed:", r.Method)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// GET /follow/requests
func (h *Handler) handleListRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fmt.Println("[FOLLOWERS][HANDLER] list requests rejected: method not allowed:", r.Method)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	viewerID := h.getUserID(r, w)
	if viewerID == "" {
		fmt.Println("[FOLLOWERS][HANDLER] list requests aborted: missing authenticated viewer")
		return
	}
	fmt.Println("[FOLLOWERS][HANDLER] list requests for viewer:", viewerID)

	reqs, err := h.service.GetPendingRequests(r.Context(), viewerID)
	if err != nil {
		fmt.Println("[FOLLOWERS][HANDLER] list requests failed:", err)
		writeError(w, http.StatusInternalServerError, "failed to get requests")
		return
	}
	if reqs == nil {
		reqs = []FollowRequest{}
	}
	fmt.Println("[FOLLOWERS][HANDLER] list requests success count:", len(reqs))
	writeJSON(w, http.StatusOK, map[string]any{"requests": reqs})
}

// POST /follow/requests/{requestID}/accept|decline
func (h *Handler) handleRespondRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fmt.Println("[FOLLOWERS][HANDLER] respond request rejected: method not allowed:", r.Method)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	viewerID := h.getUserID(r, w)
	if viewerID == "" {
		fmt.Println("[FOLLOWERS][HANDLER] respond request aborted: missing authenticated viewer")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/follow/requests/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		fmt.Println("[FOLLOWERS][HANDLER] respond request rejected: invalid path:", path)
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	requestID, action := parts[0], parts[1]
	fmt.Println("[FOLLOWERS][HANDLER] respond request action:", action, "request:", requestID, "viewer:", viewerID)

	switch action {
	case "accept":
		if err := h.service.AcceptRequest(r.Context(), requestID, viewerID); err != nil {
			if errors.Is(err, ErrNotFound) {
				fmt.Println("[FOLLOWERS][HANDLER] accept request failed: not found")
				writeError(w, http.StatusNotFound, "request not found")
				return
			}
			fmt.Println("[FOLLOWERS][HANDLER] accept request failed:", err)
			writeError(w, http.StatusInternalServerError, "failed to accept")
			return
		}
	case "decline":
		if err := h.service.DeclineRequest(r.Context(), requestID, viewerID); err != nil {
			fmt.Println("[FOLLOWERS][HANDLER] decline request failed:", err)
			writeError(w, http.StatusInternalServerError, "failed to decline")
			return
		}
	default:
		fmt.Println("[FOLLOWERS][HANDLER] invalid follow request action:", action)
		writeError(w, http.StatusBadRequest, "action must be accept or decline")
		return
	}
	fmt.Println("[FOLLOWERS][HANDLER] respond request success")
	writeJSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

// HandleUserFollowRoutes is called by the users handler for sub-routes:
//
//	GET /users/{id}/followers
//	GET /users/{id}/following
//	GET /users/{id}/follow-status
func (h *Handler) HandleUserFollowRoutes(w http.ResponseWriter, r *http.Request, targetID, sub string) bool {
	if r.Method != http.MethodGet {
		fmt.Println("[FOLLOWERS][HANDLER] user follow sub-route skipped: method not allowed for sub:", sub, "method:", r.Method)
		return false
	}
	fmt.Println("[FOLLOWERS][HANDLER] user follow sub-route:", sub, "target:", targetID)
	switch sub {
	case "followers":
		fmt.Println("[FOLLOWERS][HANDLER] get followers for target:", targetID)
		list, err := h.service.GetFollowers(r.Context(), targetID)
		if err != nil {
			fmt.Println("[FOLLOWERS][HANDLER] get followers failed:", err)
			writeError(w, http.StatusInternalServerError, "failed")
			return true
		}
		if list == nil {
			list = []UserSummary{}
		}
		fmt.Println("[FOLLOWERS][HANDLER] get followers success count:", len(list))
		writeJSON(w, http.StatusOK, map[string]any{"followers": list})
		return true

	case "following":
		fmt.Println("[FOLLOWERS][HANDLER] get following for target:", targetID)
		list, err := h.service.GetFollowing(r.Context(), targetID)
		if err != nil {
			fmt.Println("[FOLLOWERS][HANDLER] get following failed:", err)
			writeError(w, http.StatusInternalServerError, "failed")
			return true
		}
		if list == nil {
			list = []UserSummary{}
		}
		fmt.Println("[FOLLOWERS][HANDLER] get following success count:", len(list))
		writeJSON(w, http.StatusOK, map[string]any{"following": list})
		return true

	case "follow-status":
		viewerID := h.getUserID(r, w)
		if viewerID == "" {
			fmt.Println("[FOLLOWERS][HANDLER] follow status aborted: missing authenticated viewer")
			return true
		}
		fmt.Println("[FOLLOWERS][HANDLER] get follow status viewer:", viewerID, "target:", targetID)
		status, err := h.service.GetFollowStatus(r.Context(), viewerID, targetID)
		if err != nil {
			fmt.Println("[FOLLOWERS][HANDLER] get follow status failed:", err)
			writeError(w, http.StatusInternalServerError, "failed")
			return true
		}
		fmt.Println("[FOLLOWERS][HANDLER] get follow status success")
		writeJSON(w, http.StatusOK, status)
		return true
	}
	fmt.Println("[FOLLOWERS][HANDLER] user follow sub-route not handled:", sub)
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

func (h *Handler) getUserID(r *http.Request, w http.ResponseWriter) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		fmt.Println("[FOLLOWERS][HANDLER] session cookie missing")
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return ""
	}
	fmt.Println("[FOLLOWERS][HANDLER] resolving user from session:", cookie.Value)
	viewerID, err := h.service.currentUserID(r.Context(), cookie.Value)
	if err != nil {
		fmt.Println("[FOLLOWERS][HANDLER] current user lookup failed:", err)
		if errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return ""
		}
		writeError(w, http.StatusInternalServerError, "failed to get current user")
		return ""
	}
	fmt.Println("[FOLLOWERS][HANDLER] authenticated viewer:", viewerID)
	return viewerID
}
