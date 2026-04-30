package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	hub     *Hub
	service *Service
	conn    *websocket.Conn
	send    chan OutboundEvent // buffered; hub writes here
}

func NewClient(userID string, hub *Hub, service *Service, conn *websocket.Conn) *Client {
	return &Client{
		userID:  userID,
		hub:     hub,
		service: service,
		conn:    conn,
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

// HandleGroupMessage validates and delivers a group chat message.
func (s *Service) HandleGroupMessage(c *Client, event InboundEvent) {
	ctx := context.Background()

	if strings.TrimSpace(event.Body) == "" || event.To == "" {
		c.send <- OutboundEvent{Type: "error", Error: "to and body are required"}
		return
	}

	isMember, err := s.repo.IsGroupMember(ctx, event.To, c.userID)
	if err != nil {
		fmt.Printf("[CHAT][SERVICE] IsGroupMember error: %v\n", err)
		c.send <- OutboundEvent{Type: "error", Error: "internal error"}
		return
	}
	if !isMember {
		c.send <- OutboundEvent{Type: "error", Error: "not a member of this group"}
		return
	}

	msg, err := s.repo.SaveMessage(ctx, event.To, "group", c.userID, strings.TrimSpace(event.Body))
	if err != nil {
		fmt.Printf("[CHAT][SERVICE] SaveMessage error: %v\n", err)
		c.send <- OutboundEvent{Type: "error", Error: "failed to save message"}
		return
	}

	memberIDs, err := s.repo.GetGroupMemberIDs(ctx, event.To)
	if err != nil {
		fmt.Printf("[CHAT][SERVICE] GetGroupMemberIDs error: %v\n", err)
		return
	}

	out := OutboundEvent{Type: "message", Payload: &msg}
	s.hub.Send(memberIDs, out)
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
