package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/v3danth/free-chat/internal/filter"
	"github.com/v3danth/free-chat/internal/media"
	"github.com/v3danth/free-chat/internal/message"
	"github.com/v3danth/free-chat/internal/ratelimit"
	"github.com/v3danth/free-chat/internal/user"
)

type Hub struct {
	users      map[uint64]*Client
	rooms      map[uint64]*Room
	userRepo   user.Repository
	msgRepo    message.Repository
	mediaRepo  media.Repository
	storage    media.Storage
	historyLim int
	limiter    *ratelimit.Limiter
	msgFilter  *filter.Filter
	mu         sync.RWMutex
	rateConfig ratelimit.Config

	register   chan *Client
	unregister chan *Client
}

func NewHub(
	userRepo user.Repository,
	msgRepo message.Repository,
	mediaRepo media.Repository,
	storage media.Storage,
	historyLimit int,
	rateCfg ratelimit.Config,
	filterCfg filter.Config,
) *Hub {
	if historyLimit <= 0 || historyLimit > 100 {
		historyLimit = 50
	}

	return &Hub{
		users:      make(map[uint64]*Client),
		rooms:      make(map[uint64]*Room),
		userRepo:   userRepo,
		msgRepo:    msgRepo,
		mediaRepo:  mediaRepo,
		storage:    storage,
		historyLim: historyLimit,
		limiter:    ratelimit.NewLimiter(rateCfg),
		msgFilter:  filter.NewFilter(filterCfg),
		rateConfig: rateCfg,
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.handleRegister(client)

		case client := <-h.unregister:
			h.handleUnregister(client)
		}
	}
}

func (h *Hub) handleRegister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if existing, ok := h.users[client.userID]; ok && existing != client {
		log.Printf("replacing existing connection for user %d", client.userID)
	}

	h.users[client.userID] = client
	log.Printf("user %d (%s) connected", client.userID, client.username)

	// Send rate limit info on connect
	go h.sendRateLimitInfo(client)
}

func (h *Hub) handleUnregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if existing, ok := h.users[client.userID]; !ok || existing != client {
		return
	}

	delete(h.users, client.userID)

	for roomID := range client.rooms {
		h.leaveRoomUnsafe(roomID, client)
	}

	if client.userType == user.TypeGuest {
		go h.markGuestInactive(client.userID)
	}

	log.Printf("user %d (%s) disconnected", client.userID, client.username)
}

func (h *Hub) markGuestInactive(userID uint64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.userRepo.SetInactive(ctx, userID); err != nil {
		log.Printf("failed to mark guest %d inactive: %v", userID, err)
	}
}

func (h *Hub) IsUserOnline(userID uint64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	_, ok := h.users[userID]
	return ok
}

func (h *Hub) sendRateLimitInfo(client *Client) {
	info := RateLimitInfo{
		Type:      "rate_limit",
		Remaining: h.limiter.Remaining(client.userID),
		ResetIn:   int64(h.rateConfig.Window.Seconds()),
	}

	data, err := json.Marshal(info)
	if err != nil {
		return
	}

	select {
	case client.send <- data:
	default:
	}
}
