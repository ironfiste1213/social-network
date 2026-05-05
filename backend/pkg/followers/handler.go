package followers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"social-network/backend/pkg/response"
	"social-network/backend/pkg/sessionauth"
)

type Handler struct {
	service      *Service
	notifService NotifService 
}
type NotifService interface {
	NotifyFollowRequest(
		ctx             context.Context,
		recipientID     string,
		actorID         string,
		followRequestID string,
	) error
}
var ErrUserNotFound = errors.New("user not found")

func NewHandler(db *sql.DB, svc NotifService) *Handler {
	repo := NewRepository(db)
	service := NewService(repo)
	return &Handler{service: service, notifService: svc}
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

// POST /follow/{targetID} → send follow request or direct follow (if public profile)
// DELETE /follow/{targetID} → unfollow user OR cancel pending outgoing follow request
func (h *Handler) handleFollow(w http.ResponseWriter, r *http.Request, targetID string) {
	if targetID == "" {
		fmt.Println("[FOLLOWERS][HANDLER] follow rejected: missing target user id")
		response.Error(w, http.StatusBadRequest, "missing target user id")
		return
	}
	viewerID := h.getUserID(r, w)
	if viewerID == "" {
		fmt.Println("[FOLLOWERS][HANDLER] follow aborted: missing authenticated viewer for target:", targetID)
		return
	}

	fmt.Println("[FOLLOWERS][HANDLER] follow route:", r.Method, "viewer:", viewerID, "target:", targetID)

	switch r.Method {
	// backend/pkg/followers/handler.go
    // Replace handleFollow POST case only

case http.MethodPost:
	err := h.service.Follow(r.Context(), viewerID, targetID)
	switch {
	case errors.Is(err, ErrCannotFollowSelf):
		response.Error(w, http.StatusBadRequest, "cannot follow yourself")
	case errors.Is(err, ErrAlreadyFollowing):
		response.Error(w, http.StatusConflict, "already following")
	case errors.Is(err, ErrRequestAlreadyExists):
		response.Error(w, http.StatusConflict, "follow request already sent")
	case errors.Is(err, ErrNotFound):
		response.Error(w, http.StatusNotFound, "user not found")
	case err != nil:
		response.Error(w, http.StatusInternalServerError, "failed to follow")
	default:
		// Check if a request was created (private profile)
		// or a direct follow happened (public profile)
		status, err := h.service.GetFollowStatus(r.Context(), viewerID, targetID)
		if err != nil {
			// Still success, just can't get status detail
			response.JSON(w, http.StatusOK, map[string]any{"message": "ok"})
			return
		}
		response.JSON(w, http.StatusOK, map[string]any{
			"message":    "ok",
			"request_id": status.RequestID,         // empty string if public (direct follow)
			"is_following":       status.IsFollowing,
			"has_pending_request": status.HasPendingRequest,
		})
	}

	case http.MethodDelete:
		if err := h.service.Unfollow(r.Context(), viewerID, targetID); err != nil {
			fmt.Println("[FOLLOWERS][HANDLER] unfollow failed:", err)
			response.Error(w, http.StatusInternalServerError, "failed to unfollow")
			return
		}
		fmt.Println("[FOLLOWERS][HANDLER] unfollow success")
		response.JSON(w, http.StatusOK, map[string]string{
			"message": "ok",
			"note":    "removed follower relationship and/or cancelled pending follow request",
		})

	default:
		fmt.Println("[FOLLOWERS][HANDLER] follow route rejected: method not allowed:", r.Method)
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// GET /follow/requests
func (h *Handler) handleListRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fmt.Println("[FOLLOWERS][HANDLER] list requests rejected: method not allowed:", r.Method)
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
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
		response.Error(w, http.StatusInternalServerError, "failed to get requests")
		return
	}
	if reqs == nil {
		reqs = []FollowRequest{}
	}
	fmt.Println("[FOLLOWERS][HANDLER] list requests success count:", len(reqs))
	response.JSON(w, http.StatusOK, map[string]any{"requests": reqs})
}

// POST /follow/requests/{requestID}/accept|decline|cancel
func (h *Handler) handleRespondRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fmt.Println("[FOLLOWERS][HANDLER] respond request rejected: method not allowed:", r.Method)
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
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
		response.Error(w, http.StatusBadRequest, "invalid path")
		return
	}
	requestID, action := parts[0], parts[1]
	fmt.Println("[FOLLOWERS][HANDLER] respond request action:", action, "request:", requestID, "viewer:", viewerID)

	switch action {
	case "accept":
		if err := h.service.AcceptRequest(r.Context(), requestID, viewerID); err != nil {
			if errors.Is(err, ErrNotFound) {
				fmt.Println("[FOLLOWERS][HANDLER] accept request failed: not found")
				response.Error(w, http.StatusNotFound, "request not found")
				return
			}
			fmt.Println("[FOLLOWERS][HANDLER] accept request failed:", err)
			response.Error(w, http.StatusInternalServerError, "failed to accept")
			return
		}
	case "decline":
		if err := h.service.DeclineRequest(r.Context(), requestID, viewerID); err != nil {
			fmt.Println("[FOLLOWERS][HANDLER] decline request failed:", err)
			response.Error(w, http.StatusInternalServerError, "failed to decline")
			return
		}
	case "cancel":
		if err := h.service.CancelRequest(r.Context(), requestID, viewerID); err != nil {
			if errors.Is(err, ErrNotFound) {
				fmt.Println("[FOLLOWERS][HANDLER] cancel request failed: not found")
				response.Error(w, http.StatusNotFound, "request not found")
				return
			}
			fmt.Println("[FOLLOWERS][HANDLER] cancel request failed:", err)
			response.Error(w, http.StatusInternalServerError, "failed to cancel")
			return
		}
	default:
		fmt.Println("[FOLLOWERS][HANDLER] invalid follow request action:", action)
		response.Error(w, http.StatusBadRequest, "action must be accept, decline, or cancel")
		return
	}
	fmt.Println("[FOLLOWERS][HANDLER] respond request success")
	response.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
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
			response.Error(w, http.StatusInternalServerError, "failed")
			return true
		}
		if list == nil {
			list = []UserSummary{}
		}
		fmt.Println("[FOLLOWERS][HANDLER] get followers success count:", len(list))
		response.JSON(w, http.StatusOK, map[string]any{"followers": list})
		return true

	case "following":
		fmt.Println("[FOLLOWERS][HANDLER] get following for target:", targetID)
		list, err := h.service.GetFollowing(r.Context(), targetID)
		if err != nil {
			fmt.Println("[FOLLOWERS][HANDLER] get following failed:", err)
			response.Error(w, http.StatusInternalServerError, "failed")
			return true
		}
		if list == nil {
			list = []UserSummary{}
		}
		fmt.Println("[FOLLOWERS][HANDLER] get following success count:", len(list))
		response.JSON(w, http.StatusOK, map[string]any{"following": list})
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
			response.Error(w, http.StatusInternalServerError, "failed")
			return true
		}
		fmt.Println("[FOLLOWERS][HANDLER] get follow status success")
		response.JSON(w, http.StatusOK, status)
		return true
	}
	fmt.Println("[FOLLOWERS][HANDLER] user follow sub-route not handled:", sub)
	return false
}

func (h *Handler) getUserID(r *http.Request, w http.ResponseWriter) string {
	sessionID, err := sessionauth.SessionIDFromRequest(r)
	if err != nil {
		fmt.Println("[FOLLOWERS][HANDLER] session cookie missing")
		response.Error(w, http.StatusUnauthorized, "not authenticated")
		return ""
	}
	fmt.Println("[FOLLOWERS][HANDLER] resolving user from session:", sessionID)
	viewerID, err := h.service.currentUserID(r.Context(), sessionID)
	if err != nil {
		fmt.Println("[FOLLOWERS][HANDLER] current user lookup failed:", err)
		if errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrInvalidCredentials) {
			response.Error(w, http.StatusUnauthorized, "not authenticated")
			return ""
		}
		response.Error(w, http.StatusInternalServerError, "failed to get current user")
		return ""
	}
	fmt.Println("[FOLLOWERS][HANDLER] authenticated viewer:", viewerID)
	return viewerID
}
