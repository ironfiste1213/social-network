package chat

import (
	"fmt"
	"sync"
)

// Hub maintains the set of active WebSocket clients and broadcasts messages.
// A user can have multiple simultaneous connections (multiple tabs).
type Hub struct {
	clients    map[string][]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *OutboundMessage
	mu         sync.RWMutex
}

type OutboundMessage struct {
	Recipients []string
	Event      OutboundEvent
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string][]*Client),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		broadcast:  make(chan *OutboundMessage, 256),
	}
}

func (h *Hub) Run() {
	fmt.Println("[CHAT][HUB] started")
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.userID] = append(h.clients[client.userID], client)
			total := len(h.clients[client.userID])
			h.mu.Unlock()
			fmt.Printf("[CHAT][HUB] registered user %s (connections=%d)\n", client.userID, total)

		case client := <-h.unregister:
			h.mu.Lock()
			conns := h.clients[client.userID]
			for i, c := range conns {
				if c == client {
					h.clients[client.userID] = append(conns[:i], conns[i+1:]...)
					close(client.send)
					break
				}
			}
			if len(h.clients[client.userID]) == 0 {
				delete(h.clients, client.userID)
			}
			h.mu.Unlock()
			fmt.Printf("[CHAT][HUB] unregistered connection for user %s\n", client.userID)

		case msg := <-h.broadcast:
			h.mu.RLock()
			for _, recipientID := range msg.Recipients {
				for _, client := range h.clients[recipientID] {
					select {
					case client.send <- msg.Event:
					default:
						fmt.Printf("[CHAT][HUB] dropping message for slow client user=%s\n", recipientID)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Send(recipients []string, event OutboundEvent) {
	h.broadcast <- &OutboundMessage{Recipients: recipients, Event: event}
}

func (h *Hub) IsOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}