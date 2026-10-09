// Package ratelimit is a fixed-window limiter. The decision logic is the pure
// function step; Limiter only adds a lock and a map around it.
package ratelimit

import (
	"sync"
	"time"
)

type Config struct {
	Rate   int
	Window time.Duration
}

type window struct {
	start time.Time
	count int
}

// step returns the user's next window and whether this event is allowed.
func step(w window, now time.Time, cfg Config) (window, bool) {
	if now.Sub(w.start) >= cfg.Window {
		w = window{start: now}
	}
	if w.count >= cfg.Rate {
		return w, false
	}
	w.count++
	return w, true
}

func remaining(w window, now time.Time, cfg Config) int {
	if now.Sub(w.start) >= cfg.Window {
		return cfg.Rate
	}
	return max(cfg.Rate-w.count, 0)
}

type Limiter struct {
	cfg Config

	mu        sync.Mutex
	windows   map[uint64]window
	lastSweep time.Time
}

func New(cfg Config) *Limiter {
	if cfg.Rate <= 0 {
		cfg.Rate = 30
	}
	if cfg.Window <= 0 {
		cfg.Window = time.Minute
	}
	return &Limiter{cfg: cfg, windows: make(map[uint64]window)}
}

func (l *Limiter) Config() Config { return l.cfg }

func (l *Limiter) Allow(userID uint64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	l.sweep(now)

	w, ok := step(l.windows[userID], now, l.cfg)
	l.windows[userID] = w
	return ok
}

func (l *Limiter) Remaining(userID uint64) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return remaining(l.windows[userID], time.Now(), l.cfg)
}

// sweep drops expired windows at most once per window, keeping memory
// bounded without a background goroutine.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < l.cfg.Window {
		return
	}
	for id, w := range l.windows {
		if now.Sub(w.start) >= l.cfg.Window {
			delete(l.windows, id)
		}
	}
	l.lastSweep = now
}
