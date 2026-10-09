package websocket

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/v3danth/free-chat/internal/user"
)

const (
	writeWait     = 10 * time.Second
	pongWait      = 60 * time.Second
	pingPeriod    = pongWait * 9 / 10
	maxFrameBytes = 16 << 10
	sendBuffer    = 256
)

type Client struct {
	uid  uint64
	conn *websocket.Conn
	send chan []byte

	mu     sync.Mutex // guards closed and the close of send
	closed bool

	// Guarded by Hub.mu.
	user    user.User
	since   time.Time
	rooms   map[uint64]struct{}
	blocked map[uint64]struct{}
}

func newClient(u user.User, blocked map[uint64]struct{}, conn *websocket.Conn) *Client {
	return &Client{
		uid:     u.ID,
		conn:    conn,
		send:    make(chan []byte, sendBuffer),
		user:    u,
		since:   time.Now(),
		rooms:   make(map[uint64]struct{}),
		blocked: blocked,
	}
}

// id never changes, so it needs no lock (c.user is replaced under Hub.mu).
func (c *Client) id() uint64 { return c.uid }

// deliver queues a frame without blocking. A client that cannot keep up is
// disconnected rather than silently missing messages.
func (c *Client) deliver(frame []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	select {
	case c.send <- frame:
	default:
		c.closeLocked()
	}
}

// close is idempotent. Frames already queued are still written first, then
// writePump closes the socket.
func (c *Client) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeLocked()
}

func (c *Client) closeLocked() {
	if !c.closed {
		c.closed = true
		close(c.send)
	}
}

// readPump owns all reads. Deadlines are refreshed by pongs, so a dead peer
// is detected within pongWait instead of holding the connection forever.
func (c *Client) readPump(h *Hub) {
	defer h.unregister(c)

	c.conn.SetReadLimit(maxFrameBytes)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		cmd, err := parseCommand(raw)
		if err == nil {
			err = h.dispatch(c, cmd)
		}
		if err != nil {
			c.deliver(errorFrame(err))
		}
	}
}

// writePump owns all writes, including keep-alive pings.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case frame, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, frame); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
