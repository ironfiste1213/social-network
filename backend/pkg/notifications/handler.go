package notifications

import (
	"database/sql"
	"net/http"
	"strings"

	"social-network/backend/pkg/response"
	"social-network/backend/pkg/sessionauth"
)

type Handler struct {
	service *Service
}

func NewHandler(db *sql.DB) *Handler {
	repo := NewRepository(db)
	service := NewService(repo)
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/notifications", h.handleList)
	mux.HandleFunc("/notifications/", h.handleActions)
}

// GET /notifications — returns all notifications + unread count
func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, ok := sessionauth.RequireUserID(w, r, h.service.CurrentUserID)
	if !ok {
		return
	}

	notifs, err := h.service.GetAll(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to get notifications")
		return
	}
	if notifs == nil {
		notifs = []Notification{}
	}

	count, _ := h.service.UnreadCount(r.Context(), userID)
	response.JSON(w, http.StatusOK, map[string]any{
		"notifications": notifs,
		"unread_count":  count,
	})
}

// POST /notifications/read-all         — mark all as read
// POST /notifications/{id}/read        — mark one as read
func (h *Handler) handleActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, ok := sessionauth.RequireUserID(w, r, h.service.CurrentUserID)
	if !ok {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/notifications/")

	if path == "read-all" {
		if err := h.service.MarkAllRead(r.Context(), userID); err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to mark all read")
			return
		}
		response.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
		return
	}

	// /notifications/{id}/read
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[1] != "read" {
		http.NotFound(w, r)
		return
	}

	if err := h.service.MarkRead(r.Context(), parts[0], userID); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to mark read")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

func NewServiceFromDB(db *sql.DB) *Service {
	return NewService(NewRepository(db))
}

func NewHandlerWithService(db *sql.DB, svc *Service) *Handler {
	return &Handler{service: svc}
}