package chat

import (
	"database/sql"
	"net/http"
	"social-network/backend/pkg/sessionauth"

	"github.com/gorilla/websocket"
)

type Handler struct {
	service *Service
}

var upgrader = websocket.Upgrader {
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	CheckOrigin: func(r *http.Request) bool {return true},
}

func NewHandler(db *sql.DB, hub *Hub) *Handler {
	repo := NewRepository(db)
	service := NewService(repo, hub)
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux)  {
	mux.HandleFunc("/ws", h.ServWS)
}

func (h *Handler) ServWS(w http.ResponseWriter, r *http.Request) {
	userId, ok := sessionauth.RequireUserID(w, r, h.service.repo.GetUserBySessionID) 
	if !ok {
		// i should add propr error !!
		return 
	}
    conn, err := upgrader.Upgrade(w, r, nil) 
	if err != nil {
		// upgrader already wrote the HTTP error
		return
	}

	client := NewClient(userId, h.service.hub, h.service, conn)
	h.service.hub.register <- client

	go client.ReadPump()
	go client.WritePump()

}
