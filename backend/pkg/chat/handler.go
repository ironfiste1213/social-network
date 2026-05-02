package chat

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"social-network/backend/pkg/response"
	"social-network/backend/pkg/sessionauth"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Handler struct {
	service *Service
}

func NewHandler(db *sql.DB, hub *Hub) *Handler {
	repo := NewRepository(db)
	service := NewService(repo, hub)
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/ws", h.ServeWS)
	mux.HandleFunc("/chat/", h.HandleREST)
}

// GET /ws
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	userID, ok := sessionauth.RequireUserID(w, r, h.service.CurrentUserID)
	if !ok {
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := NewClient(userID, h.service.hub, h.service, conn)
	h.service.hub.register <- client
	go client.WritePump()
	go client.ReadPump()
}

func (h *Handler) HandleREST(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/chat/")
	switch {
	case path == "conversations" && r.Method == http.MethodGet:
		h.listConversations(w, r)
	case path == "messages" && r.Method == http.MethodGet:
		h.getHistory(w, r)
	default:
		http.NotFound(w, r)
	}
}

// GET /chat/conversations
func (h *Handler) listConversations(w http.ResponseWriter, r *http.Request) {
	userID, ok := sessionauth.RequireUserID(w, r, h.service.CurrentUserID)
	if !ok {
		return
	}
	convos, err := h.service.GetConversations(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to get conversations")
		return
	}
	if convos == nil {
		convos = []Conversation{}
	}
	response.JSON(w, http.StatusOK, map[string]any{"conversations": convos})
}

// GET /chat/messages?receiver_id=<userID>&before_id=<msgID>&limit=50   (private)
// GET /chat/messages?group_id=<groupID>&before_id=<msgID>&limit=50      (group)
//
// Frontend never sends or stores chat_id.
// Backend derives it from receiver_id (private) or uses group_id directly (group).
// before_id is the ID of the oldest message the client already has — omit for first load.
func (h *Handler) getHistory(w http.ResponseWriter, r *http.Request) {
	senderID, ok := sessionauth.RequireUserID(w, r, h.service.CurrentUserID)
	if !ok {
		return
	}

	receiverID := r.URL.Query().Get("receiver_id")
	groupID := r.URL.Query().Get("group_id")
	beforeID := r.URL.Query().Get("before_id")

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	var (
		msgs []Message
		err  error
	)

	switch {
	case receiverID != "":
		msgs, err = h.service.GetPrivateHistory(r.Context(), senderID, receiverID, beforeID, limit)
	case groupID != "":
		msgs, err = h.service.GetGroupHistory(r.Context(), groupID, senderID, beforeID, limit)
	default:
		response.Error(w, http.StatusBadRequest, "receiver_id or group_id is required")
		return
	}

	if err != nil {
		if err == ErrForbidden {
			response.Error(w, http.StatusForbidden, "not allowed")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to get messages")
		return
	}
	if msgs == nil {
		msgs = []Message{}
	}
	response.JSON(w, http.StatusOK, map[string]any{"messages": msgs})
}