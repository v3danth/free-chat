// Command loadtest measures what a visitor waits for, as percentiles:
// static files, faces, a signed-in API call, and the round trip of a room
// message through the websocket (sent, fanned out, and back to its sender).
// Run it against localhost for the server's own cost, and through a tunnel
// for what a visitor across the internet sees.
//
//	go run ./cmd/loadtest -url http://localhost:8080
//	go run ./cmd/loadtest -url https://example.trycloudflare.com
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/v3danth/free-chat/internal/botclient"
)

func main() {
	server := flag.String("url", "http://localhost:8080", "server to measure")
	workers := flag.Int("c", 16, "concurrent HTTP clients per target")
	each := flag.Duration("d", 15*time.Second, "how long to load each HTTP target")
	users := flag.Int("users", 8, "people sending room messages (each is a guest sign-up; the server allows 30 per 10 minutes per IP)")
	chatFor := flag.Duration("chat", 45*time.Second, "how long the room messages run")
	hammerMode := flag.Bool("hammer", false, "find the breaking points instead (needs a server started with BEHIND_PROXY=true)")
	levels := flag.String("levels", "16,64,256,1024", "hammer: API concurrency steps")
	rooms := flag.String("rooms", "100,500,1000", "hammer: room sizes")
	talkers := flag.Int("talkers", 0, "hammer: how many in the room talk, the rest listen (0: everyone)")
	flag.Parse()
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	ctx := context.Background()
	token, err := botclient.SignUp(ctx, *server, guest("probe"))
	if err != nil {
		slog.Error("could not sign up", "err", err)
		os.Exit(1)
	}

	if *hammerMode {
		hammer(ctx, *server, token, ints(*levels), ints(*rooms), *talkers, *each, *chatFor)
		return
	}

	targets := []target{
		{name: "GET / (page)", path: func(int) string { return "/" }},
		{name: "GET /app.js (gzip)", path: func(int) string { return "/app.js" }},
		{name: "GET /avatar (200 faces)", path: func(i int) string { return "/avatar/v2/load-" + strconv.Itoa(i%200) }},
		{name: "GET /me (signed in)", path: func(int) string { return "/me" }, token: token},
	}
	rows := make([]summary, 0, len(targets)+2)
	for _, t := range targets {
		rows = append(rows, loadHTTP(*server, t, *workers, *each))
	}
	rows = append(rows, chatRoundTrips(ctx, *server, *users, *chatFor)...)
	fmt.Print(table(fmt.Sprintf("Latency at %s (%d clients per HTTP target, %d chatting)", *server, *workers, *users), rows))
}

// ints parses a comma-separated list of positive numbers, ascending.
func ints(list string) []int {
	var out []int
	for _, f := range strings.Split(list, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(f)); err == nil && n > 0 {
			out = append(out, n)
		}
	}
	slices.Sort(out)
	return out
}

func guest(name string) botclient.Guest {
	return botclient.Guest{Name: name, Gender: "other", Age: 30, Tags: []string{"just talk"}}
}

type target struct {
	name  string
	path  func(i int) string
	token string
}

// loadHTTP runs workers clients against t for d, each sending its next
// request as soon as the last one finished, and times each to its last byte.
func loadHTTP(server string, t target, workers int, d time.Duration) summary {
	client := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: workers}}
	var mu sync.Mutex
	samples := make([]time.Duration, 0, 4096)
	failed := 0
	deadline := time.Now().Add(d)
	start := time.Now()
	var wg sync.WaitGroup
	for w := range workers {
		wg.Go(func() {
			for i := w; time.Now().Before(deadline); i += workers {
				took, err := timedGet(client, server+t.path(i), t.token)
				mu.Lock()
				if err != nil {
					failed++
				} else {
					samples = append(samples, took)
				}
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	return summarize(t.name, samples, failed, time.Since(start))
}

func timedGet(client *http.Client, target, token string) (time.Duration, error) {
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return 0, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	start := time.Now()
	res, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = res.Body.Close() }()
	if _, err := io.Copy(io.Discard, res.Body); err != nil {
		return 0, err
	}
	if res.StatusCode >= 400 {
		return 0, fmt.Errorf("%s", res.Status)
	}
	return time.Since(start), nil
}

// The server allows 30 room messages a minute per person; one every
// 2.5 s stays under it, so no sample is a rate-limit refusal.
const chatGap = 2500 * time.Millisecond

// chatRoundTrips signs up users people who each send a room line every
// chatGap and time it until the server echoes it back to them, which is
// after it has been fanned out to everyone. It also times connecting: the
// handshake through to the hello frame.
func chatRoundTrips(ctx context.Context, server string, users int, d time.Duration) []summary {
	var mu sync.Mutex
	var connects, trips []time.Duration
	failed := 0
	start := time.Now()
	var wg sync.WaitGroup
	for u := range users {
		wg.Go(func() {
			c, rt, err := chatter(ctx, server, u, d)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				slog.Warn("chatter failed", "user", u, "err", err)
				failed++
				return
			}
			connects = append(connects, c)
			trips = append(trips, rt...)
		})
	}
	wg.Wait()
	elapsed := time.Since(start)
	return []summary{
		summarize("WS connect to hello", connects, failed, 0),
		summarize("WS room message round trip", trips, 0, elapsed),
	}
}

func chatter(ctx context.Context, server string, n int, d time.Duration) (time.Duration, []time.Duration, error) {
	token, err := botclient.SignUp(ctx, server, guest("load "+strconv.Itoa(n)))
	if err != nil {
		return 0, nil, err
	}
	addr, err := botclient.SocketURL(server, token)
	if err != nil {
		return 0, nil, err
	}
	// Browsers negotiate compression, so the measurement does too.
	dialer := websocket.Dialer{HandshakeTimeout: 15 * time.Second, EnableCompression: true}
	began := time.Now()
	conn, _, err := dialer.DialContext(ctx, addr, nil)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = conn.Close() }()
	var hello struct {
		Type string `json:"type"`
		You  struct {
			ID uint64 `json:"id"`
		} `json:"you"`
	}
	first, err := botclient.ReadFrames(conn)
	if err != nil || json.Unmarshal(first[0], &hello) != nil || hello.Type != "hello" {
		return 0, nil, fmt.Errorf("no hello: %v", err)
	}
	connect := time.Since(began)
	if err := conn.WriteJSON(map[string]any{"type": "join", "room_id": 1}); err != nil {
		return 0, nil, err
	}

	sentAt := sync.Map{} // content -> time sent
	trips := make(chan time.Duration, 256)
	go func() {
		defer close(trips)
		for {
			frames, err := botclient.ReadFrames(conn)
			if err != nil {
				return
			}
			for _, raw := range frames {
				var f struct {
					Type     string `json:"type"`
					SenderID uint64 `json:"sender_id"`
					Content  string `json:"content"`
				}
				if json.Unmarshal(raw, &f) != nil || f.Type != "chat" || f.SenderID != hello.You.ID {
					continue
				}
				if t, ok := sentAt.LoadAndDelete(f.Content); ok {
					trips <- time.Since(t.(time.Time))
				}
			}
		}
	}()

	// Spread the first sends so the users do not all speak at once.
	time.Sleep(time.Duration(n) * chatGap / 8)
	tick := time.NewTicker(chatGap)
	defer tick.Stop()
	deadline := time.Now().Add(d)
	for seq := 0; time.Now().Before(deadline); seq++ {
		content := fmt.Sprintf("load test %d.%d", n, seq)
		sentAt.Store(content, time.Now())
		if err := conn.WriteJSON(map[string]any{"type": "chat", "room_id": 1, "content": content}); err != nil {
			return connect, nil, err
		}
		<-tick.C
	}
	time.Sleep(2 * time.Second) // let the last echoes land
	_ = conn.Close()
	var out []time.Duration
	for t := range trips {
		out = append(out, t)
	}
	return connect, out, nil
}
