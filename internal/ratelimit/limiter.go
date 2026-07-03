package ratelimit

import (
	"sync"
	"time"
)

type Config struct {
	Rate         int
	Window       time.Duration
	CleanupEvery time.Duration
}

func DefaultConfig() Config {
	return Config{
		Rate:         30,
		Window:       60 * time.Second,
		CleanupEvery: 5 * time.Minute,
	}
}

type userBucket struct {
	count       int
	windowStart time.Time
}

type Limiter struct {
	mu     sync.RWMutex
	users  map[uint64]*userBucket
	config Config
	stopCh chan struct{}
}

func NewLimiter(cfg Config) *Limiter {
	if cfg.Rate <= 0 {
		cfg.Rate = DefaultConfig().Rate
	}
	if cfg.Window <= 0 {
		cfg.Window = DefaultConfig().Window
	}
	if cfg.CleanupEvery <= 0 {
		cfg.CleanupEvery = DefaultConfig().CleanupEvery
	}

	rl := &Limiter{
		users:  make(map[uint64]*userBucket),
		config: cfg,
		stopCh: make(chan struct{}),
	}

	go rl.cleanupLoop()

	return rl
}

func (rl *Limiter) Allow(userID uint64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	bucket, exists := rl.users[userID]
	if !exists {
		rl.users[userID] = &userBucket{
			count:       1,
			windowStart: now,
		}
		return true
	}

	if now.Sub(bucket.windowStart) >= rl.config.Window {
		bucket.count = 1
		bucket.windowStart = now
		return true
	}

	if bucket.count >= rl.config.Rate {
		return false
	}

	bucket.count++
	return true
}

func (rl *Limiter) Remaining(userID uint64) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	bucket, exists := rl.users[userID]
	if !exists {
		return rl.config.Rate
	}

	now := time.Now()
	if now.Sub(bucket.windowStart) >= rl.config.Window {
		return rl.config.Rate
	}

	remaining := rl.config.Rate - bucket.count
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

func (rl *Limiter) Reset(userID uint64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.users, userID)
}

func (rl *Limiter) Stop() {
	close(rl.stopCh)
}

func (rl *Limiter) cleanupLoop() {
	ticker := time.NewTicker(rl.config.CleanupEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopCh:
			return
		}
	}
}

func (rl *Limiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	threshold := rl.config.Window * 2

	for id, bucket := range rl.users {
		if now.Sub(bucket.windowStart) > threshold {
			delete(rl.users, id)
		}
	}
}
