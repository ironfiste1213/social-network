package groups

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"social-network/backend/pkg/response"
	"social-network/backend/pkg/sessionauth"
)

type Handler struct {
	service       *Service
	eventshandler EventsRoutHandler
}

type EventsRoutHandler interface {
	HandleGroupEventRoutes(w http.ResponseWriter, r *http.Request, groupID, subpath string) bool
}

func NewHandler(db *sql.DB) *Handler {
	repo := NewRepository(db)
	service := NewService(repo)
	return &Handler{service: service}
}

func (h *Handler) SetEventsHandler(fn EventsRoutHandler) {
	h.eventshandler = fn
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/groups", h.handleGroups)
	mux.HandleFunc("/groups/", h.handleGroupRoutes)
}

func (h *Handler) handleGroups(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listGroups(w, r)
	case http.MethodPost:
		h.createGroup(w, r)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleGroupRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/groups/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	switch parts[0] {
	case "requests":
		if len(parts) != 3 || r.Method != http.MethodPost {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.respondJoinRequest(w, r, parts[1], parts[2])
		return
	case "invitations":
		if len(parts) == 1 && r.Method == http.MethodGet {
			h.listInvitations(w, r)
			return
		}
		if len(parts) != 3 || r.Method != http.MethodPost {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.respondInvitation(w, r, parts[1], parts[2])
		return
	}

	groupID := parts[0]
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.getGroup(w, r, groupID)
		return
	}

	switch parts[1] {
	case "join":
		if r.Method != http.MethodPost {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.requestJoin(w, r, groupID)
	case "requests":
		if r.Method != http.MethodGet {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.listJoinRequests(w, r, groupID)
	case "invite":
		if r.Method != http.MethodPost {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.inviteToGroup(w, r, groupID)
	case "members":
		if r.Method != http.MethodGet {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.listMembers(w, r, groupID)
	case "events":
		sub := strings.Join(parts[1:], "/")
		h.eventshandler.HandleGroupEventRoutes(w, r, groupID, sub)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) createGroup(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	var input CreateGroupInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}

	group, err := h.service.CreateGroup(r.Context(), userID, input)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			response.Error(w, http.StatusBadRequest, "title is required")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to create group")
		return
	}

	response.JSON(w, http.StatusCreated, map[string]any{"group": group})
}

func (h *Handler) listGroups(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	limit, offset := parsePagination(r)
	groups, err := h.service.ListGroups(r.Context(), userID, limit, offset)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list groups")
		return
	}
	if groups == nil {
		groups = []Group{}
	}

	response.JSON(w, http.StatusOK, map[string]any{"groups": groups})
}

func (h *Handler) getGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	userID, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	group, err := h.service.GetGroup(r.Context(), groupID, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "group not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to get group")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"group": group})
}

func (h *Handler) requestJoin(w http.ResponseWriter, r *http.Request, groupID string) {
	userID, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	request, err := h.service.RequestJoin(r.Context(), groupID, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "group not found")
		case errors.Is(err, ErrAlreadyMember):
			response.Error(w, http.StatusConflict, "already a member")
		case errors.Is(err, ErrPendingExists):
			response.Error(w, http.StatusConflict, "pending join request or invitation already exists")
		default:
			response.Error(w, http.StatusInternalServerError, "failed to create join request")
		}
		return
	}

	response.JSON(w, http.StatusCreated, map[string]any{"request": request})
}

func (h *Handler) listJoinRequests(w http.ResponseWriter, r *http.Request, groupID string) {
	userID, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	requests, err := h.service.ListJoinRequests(r.Context(), groupID, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "group not found")
		case errors.Is(err, ErrForbidden):
			response.Error(w, http.StatusForbidden, "forbidden")
		default:
			response.Error(w, http.StatusInternalServerError, "failed to list join requests")
		}
		return
	}
	if requests == nil {
		requests = []GroupJoinRequest{}
	}

	response.JSON(w, http.StatusOK, map[string]any{"requests": requests})
}

func (h *Handler) respondJoinRequest(w http.ResponseWriter, r *http.Request, requestID, action string) {
	userID, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	var err error
	switch action {
	case "accept":
		err = h.service.AcceptJoinRequest(r.Context(), requestID, userID)
	case "decline":
		err = h.service.DeclineJoinRequest(r.Context(), requestID, userID)
	default:
		response.Error(w, http.StatusBadRequest, "action must be accept or decline")
		return
	}

	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "join request not found")
		case errors.Is(err, ErrForbidden):
			response.Error(w, http.StatusForbidden, "forbidden")
		default:
			response.Error(w, http.StatusInternalServerError, "failed to respond to join request")
		}
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

func (h *Handler) inviteToGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	userID, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	var input InviteToGroupInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}

	invitation, err := h.service.Invite(r.Context(), groupID, userID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "invitee_id is required and must not be yourself")
		case errors.Is(err, ErrForbidden):
			response.Error(w, http.StatusForbidden, "forbidden")
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "group not found")
		case errors.Is(err, ErrUserNotFound):
			response.Error(w, http.StatusNotFound, "user not found")
		case errors.Is(err, ErrAlreadyMember):
			response.Error(w, http.StatusConflict, "user is already a member")
		case errors.Is(err, ErrPendingExists):
			response.Error(w, http.StatusConflict, "pending invitation already exists")
		default:
			response.Error(w, http.StatusInternalServerError, "failed to invite user")
		}
		return
	}

	response.JSON(w, http.StatusCreated, map[string]any{"invitation": invitation})
}

func (h *Handler) listInvitations(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	invitations, err := h.service.ListInvitations(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list invitations")
		return
	}
	if invitations == nil {
		invitations = []GroupInvitation{}
	}

	response.JSON(w, http.StatusOK, map[string]any{"invitations": invitations})
}

func (h *Handler) respondInvitation(w http.ResponseWriter, r *http.Request, invitationID, action string) {
	userID, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	var err error
	switch action {
	case "accept":
		err = h.service.AcceptInvitation(r.Context(), invitationID, userID)
	case "decline":
		err = h.service.DeclineInvitation(r.Context(), invitationID, userID)
	default:
		response.Error(w, http.StatusBadRequest, "action must be accept or decline")
		return
	}

	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.Error(w, http.StatusNotFound, "invitation not found")
		case errors.Is(err, ErrForbidden):
			response.Error(w, http.StatusForbidden, "forbidden")
		default:
			response.Error(w, http.StatusInternalServerError, "failed to respond to invitation")
		}
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request, groupID string) {
	if _, ok := h.authenticate(w, r); !ok {
		return
	}

	members, err := h.service.ListMembers(r.Context(), groupID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "group not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to list members")
		return
	}
	if members == nil {
		members = []GroupMember{}
	}

	response.JSON(w, http.StatusOK, map[string]any{"members": members})
}

func (h *Handler) authenticate(w http.ResponseWriter, r *http.Request) (string, bool) {
	return sessionauth.RequireUserID(w, r, h.service.CurrentUserID)
}

func parsePagination(r *http.Request) (limit, offset int) {
	limit = 20
	offset = 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	return limit, offset
}
