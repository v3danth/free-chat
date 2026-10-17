package websocket

import (
	"encoding/json"
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
	// compressFrom is the smallest frame worth deflating. Chat lines are a
	// few hundred bytes and go out as they are; hello and history are tens
	// to hundreds of kilobytes of JSON and shrink several times over.
	compressFrom = 2 << 10
	// maxBatch caps one coalesced message (see batchQueued).
	maxBatch = 64 << 10
)

type Client struct {
	uid  uint64
	conn *websocket.Conn
	send chan []byte

	mu     sync.Mutex // guards closed and the close of send
	closed bool

	// Guarded by Hub.mu.
	user  user.User
	since time.Time
	// card is user's Card, encoded once per change rather than once per
	// reader: every arrival sends the newcomer everyone's card.
	card    json.RawMessage
	rooms   map[uint64]struct{}
	blocked map[uint64]struct{}
}

func newClient(u user.User, blocked map[uint64]struct{}, conn *websocket.Conn) *Client {
	c := &Client{
		uid:     u.ID,
		conn:    conn,
		send:    make(chan []byte, sendBuffer),
		user:    u,
		since:   time.Now(),
		rooms:   make(map[uint64]struct{}),
		blocked: blocked,
	}
	c.refreshCardLocked()
	return c
}

// refreshCardLocked re-encodes the card after c.user changes. The caller
// holds Hub.mu, or owns c before it is registered.
func (c *Client) refreshCardLocked() {
	c.card = encode(user.ToCard(c.user, c.since))
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
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return
	}
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
		_ = c.conn.Close()
	}()

	var scratch []byte
	for {
		select {
		case frame, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			msg, buf, closed := batchQueued(scratch, frame, c.send)
			scratch = buf
			c.conn.EnableWriteCompression(len(msg) >= compressFrom)
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
			if closed {
				_ = c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// batchQueued joins first with the frames already waiting behind it, one
// per line, so they go out in one write. Each write is a system call, and
// a busy room made those 91% of the server's CPU. With nothing waiting it
// returns first as it is, so a quiet connection gains no delay and no copy;
// under load batches grow by themselves. JSON never holds a raw newline,
// which makes it a safe separator. scratch is reused between calls and
// returned grown; closed reports that the queue was closed while draining.
func batchQueued(scratch, first []byte, queue <-chan []byte) (msg, grown []byte, closed bool) {
	msg, grown = first, scratch[:0]
	for len(msg) < maxBatch {
		select {
		case next, ok := <-queue:
			if !ok {
				return msg, grown, true
			}
			if len(grown) == 0 {
				grown = append(grown, first...)
			}
			grown = append(append(grown, '\n'), next...)
			msg = grown
		default:
			return msg, grown, false
		}
	}
	return msg, grown, false
}
