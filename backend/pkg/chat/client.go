package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

// Client is one WebSocket connection tied to a user.
type Client struct {
	userID  string
	ctx     context.Context
	cancel context.CancelFunc
	hub     *Hub
	service *Service
	conn    *websocket.Conn
	send    chan OutboundEvent // buffered; hub writes here
}

func NewClient(userID string, hub *Hub, service *Service, conn *websocket.Conn, ctx context.Context, cancel context.CancelFunc) *Client {
	return &Client{
		userID:  userID,
		hub:     hub,
		service: service,
		conn:    conn,
		ctx: ctx,
		cancel: cancel,
		send:    make(chan OutboundEvent, 64),
	}
}

// ReadPump pumps messages from the WebSocket to the service layer.
// Must run in its own goroutine.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		fmt.Printf("[CHAT][CLIENT] readPump done for user %s\n", c.userID)
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				fmt.Printf("[CHAT][CLIENT] unexpected close for user %s: %v\n", c.userID, err)
			}
			break
		}

		var event InboundEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			c.send <- OutboundEvent{Type: "error", Error: "invalid json"}
			continue
		}

		switch event.Type {
		case "ping":
			c.send <- OutboundEvent{Type: "pong"}
		case "send_private":
			c.service.HandlePrivateMessage(c, event)
		case "send_group":
			c.service.HandleGroupMessage(c, event)
		default:
			c.send <- OutboundEvent{Type: "error", Error: "unknown event type"}
		}
	}
}

// WritePump pumps messages from the hub's send channel to the WebSocket.
// Must run in its own goroutine.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case event, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// hub closed the channel
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(event); err != nil {
				fmt.Printf("[CHAT][CLIENT] write error for user %s: %v\n", c.userID, err)
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
