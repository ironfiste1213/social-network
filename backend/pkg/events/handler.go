package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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
	NotifyGroupEvent(
		ctx context.Context,
		memberIDs []string,
		actorID string,
		groupID string,
		eventID string,
	) error
}

func NewHandler(db *sql.DB, svc NotifService) *Handler {
	repo := NewRepository(db)
	service := NewService(repo)
	return &Handler{service: service, notifService: svc}
}

// HandleGroupEventRoutes is called by the groups handler (or server) for sub-routes:
//
//	POST   /groups/{groupID}/events
//	GET    /groups/{groupID}/events
//	POST   /groups/{groupID}/events/{eventID}/respond
func (h *Handler) HandleGroupEventRoutes(w http.ResponseWriter, r *http.Request, groupID, subpath string) bool {
	// subpath examples: "events", "events/{id}/respond"
	if subpath != "events" && !strings.HasPrefix(subpath, "events/") {
		return false
	}

	parts := strings.SplitN(subpath, "/", 3)
	// parts[0] = "events"
	// parts[1] = eventID (optional)
	// parts[2] = "respond" (optional)

	if len(parts) == 1 {
		// /groups/{groupID}/events
		switch r.Method {
		case http.MethodGet:
			h.listEvents(w, r, groupID)
		case http.MethodPost:
			h.createEvent(w, r, groupID)
		default:
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return true
	}

	if len(parts) == 3 && parts[2] == "respond" {
		// /groups/{groupID}/events/{eventID}/respond
		if r.Method != http.MethodPost {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return true
		}
		h.respond(w, r, groupID, parts[1])
		return true
	}

	return false
}

func (h *Handler) createEvent(w http.ResponseWriter, r *http.Request, groupID string) {
	userID, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	var input CreateEventInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}

	event, err := h.service.CreateEvent(r.Context(), groupID, userID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "title and event_time (RFC3339) are required")
		case errors.Is(err, ErrForbidden):
			response.Error(w, http.StatusForbidden, "you must be a group member to create events")
		default:
			response.Error(w, http.StatusInternalServerError, "failed to create event")
		}
		return
	}
	if h.notifService != nil {
		memberIDs, err := h.service.GetGroupMemberIDs(r.Context(), groupID)
		if err == nil {
			_ = h.notifService.NotifyGroupEvent(
				r.Context(), memberIDs, userID, groupID, event.ID,
			)
		}
	}

	response.JSON(w, http.StatusCreated, map[string]any{"event": event})
}

func (h *Handler) listEvents(w http.ResponseWriter, r *http.Request, groupID string) {
	userID, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	evts, err := h.service.ListEvents(r.Context(), groupID, userID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			response.Error(w, http.StatusForbidden, "you must be a group member to view events")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to list events")
		return
	}

	if evts == nil {
		evts = []GroupEvent{}
	}
	response.JSON(w, http.StatusOK, map[string]any{"events": evts})
}

func (h *Handler) respond(w http.ResponseWriter, r *http.Request, groupID, eventID string) {
	userID, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	var input RespondEventInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}

	event, err := h.service.Respond(r.Context(), groupID, eventID, userID, input.Response)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, `response must be "going" or "not_going"`)
		case errors.Is(err, ErrForbidden):
			response.Error(w, http.StatusForbidden, "you must be a group member to respond to events")
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "event not found")
		default:
			response.Error(w, http.StatusInternalServerError, "failed to respond to event")
		}
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"event": event})
}

func (h *Handler) authenticate(w http.ResponseWriter, r *http.Request) (string, bool) {
	return sessionauth.RequireUserID(w, r, h.service.CurrentUserID)
}
