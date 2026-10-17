package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/v3danth/free-chat/internal/botclient"
	"log/slog"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type config struct {
	server string
	target string
	pace   time.Duration
}

// bot is the I/O edge around the pure choices in decide.go: it owns the
// socket, the timers and the dice. One goroutine runs it, so its world
// needs no lock.
type bot struct {
	cfg    config
	name   string
	token  string
	tally  *tally
	rng    *rand.Rand
	conn   *websocket.Conn
	stop   chan struct{} // closed to retire the current connection's reader
	frames chan frame
	due    chan move
	w      world
}

func newBot(ctx context.Context, cfg config, p persona, t *tally, seed uint64) (*bot, error) {
	token, err := botclient.SignUp(ctx, cfg.server, p.Guest)
	if err != nil {
		return nil, err
	}
	if p.Aura != "" {
		if err := botclient.SetAura(ctx, cfg.server, token, p.Aura); err != nil {
			return nil, err
		}
	}
	b := &bot{cfg: cfg, name: p.Name, token: token, tally: t, due: make(chan move, 16),
		rng: rand.New(rand.NewPCG(seed, uint64(time.Now().UnixNano())))}
	if err := b.connect(ctx); err != nil {
		return nil, err
	}
	slog.Info("bot joined", "bot", b.name)
	return b, nil
}

// connect opens a fresh socket and starts its reader; the old one, if any,
// is retired first, as a browser reload does.
func (b *bot) connect(ctx context.Context) error {
	b.retire()
	addr, err := botclient.SocketURL(b.cfg.server, b.token)
	if err != nil {
		return err
	}
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, _, err := websocket.DefaultDialer.DialContext(dialCtx, addr, nil)
	if err != nil {
		return fmt.Errorf("connect %s: %w", b.name, err)
	}
	b.conn, b.stop, b.frames = conn, make(chan struct{}), make(chan frame, 32)
	go read(conn, b.frames, b.stop)
	return nil
}

func (b *bot) retire() {
	if b.conn == nil {
		return
	}
	close(b.stop)
	_ = b.conn.Close()
	b.conn = nil
}

func read(conn *websocket.Conn, out chan<- frame, stop <-chan struct{}) {
	defer close(out)
	for {
		raw, err := botclient.ReadFrames(conn)
		if err != nil {
			return
		}
		for _, r := range raw {
			var f frame
			if json.Unmarshal(r, &f) != nil {
				continue
			}
			select {
			case out <- f:
			case <-stop:
				return
			}
		}
	}
}

func (b *bot) run(ctx context.Context) {
	defer b.retire()
	next := time.NewTimer(b.gap())
	defer next.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case f, ok := <-b.frames:
			if !ok {
				b.tally.add("dropped by server")
				if err := b.connect(ctx); err != nil {
					slog.Warn("bot gave up", "bot", b.name, "err", err)
					return
				}
				continue
			}
			var moves []move
			b.w, moves = react(b.w, f, b.rng.IntN(100))
			b.schedule(ctx, moves)
		case m := <-b.due:
			b.apply(ctx, m)
		case <-next.C:
			b.schedule(ctx, initiative(b.w, b.cfg.target, b.rng.IntN(100)))
			next.Reset(b.gap())
		}
	}
}

// gap spreads a bot's own moves around the configured pace.
func (b *bot) gap() time.Duration {
	return b.cfg.pace/2 + time.Duration(b.rng.Int64N(int64(b.cfg.pace)))
}

func (b *bot) schedule(ctx context.Context, moves []move) {
	for _, m := range moves {
		if m.Delay == 0 {
			b.apply(ctx, m)
			continue
		}
		time.AfterFunc(m.Delay, func() {
			select {
			case b.due <- m:
			case <-ctx.Done():
			}
		})
	}
}

func (b *bot) apply(ctx context.Context, m move) {
	if m.Tally != "" {
		b.tally.add(m.Tally)
		if strings.HasPrefix(m.Tally, "refused") {
			slog.Warn("server refused a bot", "bot", b.name, "why", m.Tally)
		} else {
			slog.Debug("bot move", "bot", b.name, "move", m.Tally)
		}
	}
	switch {
	case m.Reload:
		b.reconnectAfter(ctx, 0)
	case m.Leave:
		b.reconnectAfter(ctx, 10*time.Second+time.Duration(b.rng.Int64N(int64(10*time.Second))))
	case m.Frame != nil && b.conn != nil:
		if err := b.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
			return
		}
		if err := b.conn.WriteJSON(m.Frame); err != nil {
			slog.Warn("bot write failed", "bot", b.name, "err", err)
		}
	}
}

// reconnectAfter drops the socket and comes back after away: zero is a
// page reload, longer is someone stepping out and returning.
func (b *bot) reconnectAfter(ctx context.Context, away time.Duration) {
	b.retire()
	select {
	case <-ctx.Done():
		return
	case <-time.After(away):
	}
	if err := b.connect(ctx); err != nil {
		slog.Warn("bot could not come back", "bot", b.name, "err", err)
	}
}
