package websocket

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
	"github.com/v3danth/free-chat/internal/user"
)

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	userID   uint64
	rooms    map[uint64]bool
	username string
	userType user.UserType
}

func (c *Client) Read() {
	defer c.disconnect()

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		var msg Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			c.sendError("invalid message format")
			continue
		}

		switch msg.Type {
		case MessageTypeJoin:
			c.hub.JoinRoom(msg.RoomID, c)

		case MessageTypeLeave:
			c.hub.LeaveRoom(msg.RoomID, c)

		case MessageTypeChat, MessageTypeMedia:
			if msg.Type == MessageTypeChat && msg.Content == "" && msg.MediaID == 0 {
				c.sendError("message content or media is required")
				continue
			}
			c.hub.BroadcastToRoom(msg.RoomID, msg, c)

		default:
			c.sendError("unknown message type")
		}
	}
}

func (c *Client) Write() {
	defer c.conn.Close()

	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (c *Client) disconnect() {
	c.hub.unregister <- c
	c.conn.Close()
}

func (c *Client) sendError(errMsg string) {
	msg := ErrorMessage{
		Type:  "error",
		Error: errMsg,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("error marshaling error message: %v", err)
		return
	}

	select {
	case c.send <- data:
	default:
		log.Printf("send buffer full, dropping error for user %d", c.userID)
	}
}
