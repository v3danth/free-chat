package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/v3danth/free-chat/internal/botclient"
)

// hammer looks for where the server bends: the signed-in API under rising
// concurrency, then a room growing to thousands of people who all talk.
// It needs a server started with BEHIND_PROXY=true so each person can sign
// up from their own address.
func hammer(ctx context.Context, server, token string, levels, rooms []int, talkers int, each, chatFor time.Duration) {
	api := make([]summary, 0, len(levels))
	for _, c := range levels {
		t := target{name: fmt.Sprintf("GET /me x%d", c), path: func(int) string { return "/me" }, token: token}
		api = append(api, loadHTTP(server, t, c, each))
	}
	fmt.Print(table("Signed-in API under rising concurrency (each GET /me is auth plus a user read)", api))

	largest := rooms[len(rooms)-1]
	people, err := signUpCrowd(ctx, server, largest)
	if err != nil {
		slog.Error("could not sign up the crowd", "err", err)
		return
	}
	room := make([]summary, 0, len(rooms))
	for _, n := range rooms {
		room = append(room, crowdTalks(ctx, server, people[:n], talkers, chatFor)...)
	}
	who := "everyone talks"
	if talkers > 0 {
		who = fmt.Sprintf("%d talk and the rest listen", talkers)
	}
	fmt.Print(table(fmt.Sprintf("A room where %s (one line per talker per %s; every line reaches everyone)", who, chatGap), room))
}

// signUpCrowd makes n guests, each from its own made-up address.
func signUpCrowd(ctx context.Context, server string, n int) ([]string, error) {
	run := strconv.Itoa(rand.IntN(9000) + 1000)
	tokens := make([]string, n)
	errs := make(chan error, n)
	sem := make(chan struct{}, 32)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			ip := fmt.Sprintf("10.%d.%d.%d", (i>>16)&255, (i>>8)&255, i&255)
			tok, err := botclient.SignUpFrom(ctx, server, ip, guest("h"+run+" "+strconv.Itoa(i)))
			if err != nil {
				errs <- err
				return
			}
			tokens[i] = tok
		})
	}
	wg.Wait()
	close(errs)
	if err := <-errs; err != nil {
		return nil, err
	}
	slog.Info("crowd signed up", "people", n)
	return tokens, nil
}

// crowdTalks connects everyone in tokens, then has the first talkers of
// them (all when talkers is 0) send a room line every chatGap for d, timing
// each line until it comes back to its sender. Everyone reads everything,
// from the moment they connect, as a browser does: a client that stops
// reading is dropped by the server, and unread data piling up in thousands
// of sockets exhausts the kernel's network memory and breaks unrelated
// connections, the database's included.
func crowdTalks(ctx context.Context, server string, tokens []string, talkers int, d time.Duration) []summary {
	if talkers <= 0 || talkers > len(tokens) {
		talkers = len(tokens)
	}
	var (
		mu        sync.Mutex
		connects  []time.Duration
		failed    atomic.Int64
		delivered atomic.Int64
	)
	members := make([]*member, len(tokens))
	// Connect in waves: the OS accept backlog is small (128 on macOS).
	sem := make(chan struct{}, 64)
	var wg sync.WaitGroup
	for i, tok := range tokens {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			conn, took, err := connect(ctx, server, tok)
			if err != nil {
				if failed.Add(1) == 1 {
					slog.Warn("first failed connect", "err", err)
				}
				return
			}
			members[i] = listen(conn, "x"+strconv.Itoa(i)+".", &delivered)
			mu.Lock()
			connects = append(connects, took)
			mu.Unlock()
		})
	}
	wg.Wait()

	start := time.Now()
	deadline := start.Add(d)
	for i, m := range members {
		if m != nil && i < talkers {
			wg.Go(func() { m.speak(i, talkers, deadline) })
		}
	}
	wg.Wait()
	time.Sleep(time.Until(deadline) + 3*time.Second) // listeners hear the last lines
	elapsed := time.Since(start)

	var trips []time.Duration
	dropped := 0
	for _, m := range members {
		if m == nil {
			continue
		}
		res := m.stop()
		trips = append(trips, res.trips...)
		if res.dropped {
			if dropped == 0 {
				slog.Warn("first dropped connection", "err", res.err)
			}
			dropped++
		}
	}
	slog.Info("room run", "people", len(tokens), "connected", len(connects), "dropped_by_server", dropped,
		"frames_delivered_per_s", int(float64(delivered.Load())/elapsed.Seconds()))
	n := len(tokens)
	return []summary{
		summarize(fmt.Sprintf("%d people: connect", n), connects, int(failed.Load()), 0),
		summarize(fmt.Sprintf("%d people: message trip", n), trips, dropped, elapsed),
	}
}

func connect(ctx context.Context, server, token string) (*websocket.Conn, time.Duration, error) {
	addr, err := botclient.SocketURL(server, token)
	if err != nil {
		return nil, 0, err
	}
	dialer := websocket.Dialer{HandshakeTimeout: 30 * time.Second, EnableCompression: true}
	began := time.Now()
	conn, _, err := dialer.DialContext(ctx, addr, nil)
	if err != nil {
		return nil, 0, err
	}
	if _, first, err := conn.ReadMessage(); err != nil || !bytes.Contains(first, []byte(`"type":"hello"`)) {
		_ = conn.Close()
		return nil, 0, fmt.Errorf("no hello: %v", err)
	}
	took := time.Since(began)
	if err := conn.WriteJSON(map[string]any{"type": "join", "room_id": 1}); err != nil {
		_ = conn.Close()
		return nil, 0, err
	}
	return conn, took, nil
}

// member is one person in the crowd: always reading, sometimes talking.
type member struct {
	conn   *websocket.Conn
	marker []byte // how this member's own lines look in a frame
	done   chan struct{}

	mu   sync.Mutex
	sent map[string]time.Time // sequence number -> when sent
	res  talkResult
}

type talkResult struct {
	trips   []time.Duration
	dropped bool
	err     error
}

// listen starts reading conn at once and keeps reading until stop.
// Frames are matched by a marker in the raw bytes, not decoded: with
// thousands of people the client would otherwise spend more CPU on JSON
// than the server does, and measure itself.
func listen(conn *websocket.Conn, id string, delivered *atomic.Int64) *member {
	m := &member{conn: conn, marker: []byte(`"content":"lt ` + id), done: make(chan struct{}), sent: map[string]time.Time{}}
	go func() {
		defer close(m.done)
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				m.mu.Lock()
				m.res.err = err
				m.mu.Unlock()
				return
			}
			// A message may hold several frames, one per line.
			delivered.Add(int64(bytes.Count(msg, []byte{'\n'}) + 1))
			m.match(msg)
		}
	}()
	return m
}

// match times every one of the member's own lines found in msg.
func (m *member) match(msg []byte) {
	for {
		at := bytes.Index(msg, m.marker)
		if at < 0 {
			return
		}
		msg = msg[at+len(m.marker):]
		end := bytes.IndexByte(msg, '"')
		if end < 0 {
			return
		}
		key := string(msg[:end])
		m.mu.Lock()
		if t, ok := m.sent[key]; ok {
			m.res.trips = append(m.res.trips, time.Since(t))
			delete(m.sent, key)
		}
		m.mu.Unlock()
	}
}

// speak sends a line every chatGap until deadline. The i-th of n talkers
// starts i/n of a gap in, so lines are spread evenly.
func (m *member) speak(i, n int, deadline time.Time) {
	time.Sleep(time.Duration(i) * chatGap / time.Duration(n))
	tick := time.NewTicker(chatGap)
	defer tick.Stop()
	for seq := 0; time.Now().Before(deadline); seq++ {
		key := strconv.Itoa(seq)
		m.mu.Lock()
		m.sent[key] = time.Now()
		m.mu.Unlock()
		if err := m.conn.WriteJSON(map[string]any{"type": "chat", "room_id": 1, "content": "lt " + string(m.marker[len(`"content":"lt `):]) + key}); err != nil {
			return
		}
		<-tick.C
	}
}

// stop closes the connection and reports what the member saw. A
// connection that ended before stop was dropped by the server.
func (m *member) stop() talkResult {
	select {
	case <-m.done:
		m.mu.Lock()
		m.res.dropped = true
		m.mu.Unlock()
	default:
	}
	_ = m.conn.Close()
	<-m.done
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.res
}
