package guest

import (
	"context"
	"log"
	"time"

	"github.com/v3danth/free-chat/internal/user"
)

type Cleaner struct {
	userRepo    user.Repository
	interval    time.Duration
	maxInactive time.Duration
	stopCh      chan struct{}
}

type CleanerConfig struct {
	SweepInterval   time.Duration
	MaxInactiveTime time.Duration
}

func DefaultCleanerConfig() CleanerConfig {
	return CleanerConfig{
		SweepInterval:   1 * time.Minute,
		MaxInactiveTime: 5 * time.Minute,
	}
}

func NewCleaner(userRepo user.Repository, cfg CleanerConfig) *Cleaner {
	if cfg.SweepInterval == 0 {
		cfg.SweepInterval = DefaultCleanerConfig().SweepInterval
	}
	if cfg.MaxInactiveTime == 0 {
		cfg.MaxInactiveTime = DefaultCleanerConfig().MaxInactiveTime
	}

	return &Cleaner{
		userRepo:    userRepo,
		interval:    cfg.SweepInterval,
		maxInactive: cfg.MaxInactiveTime,
		stopCh:      make(chan struct{}),
	}
}

func (c *Cleaner) Start() {
	log.Printf(
		"guest cleaner started: interval=%v, max_inactive=%v",
		c.interval,
		c.maxInactive,
	)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCh:
			log.Println("guest cleaner stopped")
			return
		}
	}
}

func (c *Cleaner) Stop() {
	close(c.stopCh)
}

func (c *Cleaner) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	deleted, err := c.userRepo.DeleteInactiveGuests(ctx, c.maxInactive)
	if err != nil {
		log.Printf("guest cleanup error: %v", err)
		return
	}

	if deleted > 0 {
		log.Printf("cleaned up %d inactive guest(s)", deleted)
	}
}
