package chat

import (
	"fmt"
	"sync"
)

// Hub maintains the set of active WebSocket clients and broadcasts messages.
// All mutation happens inside Run() — no external locking needed for the maps.
type Hub struct {
	// registered clients, keyed by userID
	clients map[string]*Client

	// channel-based operations fed into the single Run() goroutine
	register   chan *Client
	unregister chan *Client
	broadcast  chan *OutboundMessage

	mu sync.RWMutex // only for IsOnline reads outside Run()
}

// OutboundMessage carries the event and the set of recipient userIDs.
type OutboundMessage struct {
	Recipients []string // userIDs that should receive this message
	Event      OutboundEvent
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		broadcast:  make(chan *OutboundMessage, 256),
	}
}

// Run must be called in its own goroutine; it serialises all hub state changes.
func (h *Hub) Run() {
	fmt.Println("[CHAT][HUB] started")
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// close previous connection for same user if any
			if old, ok := h.clients[client.userID]; ok {
				fmt.Printf("[CHAT][HUB] replacing existing connection for user %s\n", client.userID)
				close(old.send)
			}
			h.clients[client.userID] = client
			h.mu.Unlock()
			fmt.Printf("[CHAT][HUB] registered user %s (total=%d)\n", client.userID, len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if current, ok := h.clients[client.userID]; ok && current == client {
				delete(h.clients, client.userID)
				close(client.send)
				fmt.Printf("[CHAT][HUB] unregistered user %s (total=%d)\n", client.userID, len(h.clients))
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			for _, recipientID := range msg.Recipients {
				if c, ok := h.clients[recipientID]; ok {
					select {
					case c.send <- msg.Event:
					default:
						// slow client — drop message rather than block hub
						fmt.Printf("[CHAT][HUB] dropping message for slow client %s\n", recipientID)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Send queues an event to the given recipients.
func (h *Hub) Send(recipients []string, event OutboundEvent) {
	h.broadcast <- &OutboundMessage{Recipients: recipients, Event: event}
}

// IsOnline returns true if the user currently has an active WS connection.
func (h *Hub) IsOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[userID]
	return ok
}