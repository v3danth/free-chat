package websocket

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/v3danth/free-chat/internal/media"
	"github.com/v3danth/free-chat/internal/message"
)

func (h *Hub) JoinRoom(roomID uint64, client *Client) {
	h.mu.Lock()

	room, ok := h.rooms[roomID]
	if !ok {
		room = NewRoom(roomID)
		h.rooms[roomID] = room
	}

	room.Clients[client] = true
	client.rooms[roomID] = true

	h.mu.Unlock()

	log.Printf(
		"user=%d joined room=%d room_size=%d",
		client.userID,
		roomID,
		len(room.Clients),
	)

	go h.sendRoomHistory(roomID, client)
}

func (h *Hub) LeaveRoom(roomID uint64, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.leaveRoomUnsafe(roomID, client)
}

func (h *Hub) leaveRoomUnsafe(roomID uint64, client *Client) {
	room, ok := h.rooms[roomID]
	if !ok {
		return
	}

	delete(room.Clients, client)
	delete(client.rooms, roomID)

	log.Printf("user %d left room %d", client.userID, roomID)

	if len(room.Clients) == 0 {
		delete(h.rooms, roomID)
		log.Printf("room %d is empty, removed", roomID)
	}
}

func (h *Hub) BroadcastToRoom(roomID uint64, msg Message, sender *Client) {
	// Rate limit check
	if !h.limiter.Allow(sender.userID) {
		h.sendRateLimitExceeded(sender)
		return
	}

	h.mu.RLock()
	room, ok := h.rooms[roomID]
	if !ok {
		h.mu.RUnlock()
		return
	}
	h.mu.RUnlock()

	// Filter content
	content := msg.Content
	filtered := false
	var violations []string

	if content != "" {
		result := h.msgFilter.Check(content)
		content = result.Content
		filtered = result.Filtered
		violations = result.Violations
	}

	// Handle media messages
	var mediaURL, mediaType string
	var dbMediaID *uint64

	if msg.MediaID > 0 {
		m, err := h.getMedia(msg.MediaID)
		if err != nil {
			log.Printf("failed to get media %d: %v", msg.MediaID, err)
			sender.sendError("media not found")
			return
		}
		if m.IsFlagged {
			sender.sendError("this media has been flagged and cannot be shared")
			return
		}
		mediaURL = h.storage.GetURL(m.FilePath)
		mediaType = string(m.MediaType)
		dbMediaID = &m.ID
	}

	// Persist message
	msgID, err := h.persistMessage(roomID, content, dbMediaID, sender)
	if err != nil {
		log.Printf("failed to persist message: %v", err)
		msgID = 0
	}

	out := OutgoingMessage{
		Type:      string(msg.Type),
		RoomID:    roomID,
		MessageID: msgID,
		SenderID:  sender.userID,
		Username:  sender.username,
		Content:   content,
		MediaURL:  mediaURL,
		MediaType: mediaType,
		Timestamp: time.Now().Unix(),
		Filtered:  filtered,
	}

	data, err := json.Marshal(out)
	if err != nil {
		log.Printf("marshal error: %v", err)
		return
	}

	h.mu.RLock()

	clients := make([]*Client, 0, len(room.Clients))
	for client := range room.Clients {
		clients = append(clients, client)
	}

	h.mu.RUnlock()
	for _, client := range clients {
		select {
		case client.send <- data:
		default:
			log.Printf("send buffer full for user %d", client.userID)
		}
	}

	if len(violations) > 0 {
		log.Printf("user %d message filtered: %v", sender.userID, violations)
	}
}

func (h *Hub) sendRateLimitExceeded(client *Client) {
	errMsg := ErrorMessage{
		Type:  "error",
		Error: "rate limit exceeded, slow down",
		Code:  "RATE_LIMITED",
	}

	data, err := json.Marshal(errMsg)
	if err != nil {
		return
	}

	select {
	case client.send <- data:
	default:
	}
}

func (h *Hub) getMedia(mediaID uint64) (*media.Media, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	m, err := h.mediaRepo.GetMediaByID(ctx, mediaID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, media.ErrMediaNotFound
	}

	return m, nil
}

func (h *Hub) persistMessage(
	roomID uint64,
	content string,
	mediaID *uint64,
	sender *Client,
) (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbMsg := &message.Message{
		RoomID:   roomID,
		SenderID: sender.userID,
		Content:  content,
		MediaID:  mediaID,
	}

	if err := h.msgRepo.Create(ctx, dbMsg); err != nil {
		return 0, err
	}

	return dbMsg.ID, nil
}

func (h *Hub) sendRoomHistory(roomID uint64, client *Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	messages, err := h.msgRepo.GetByRoom(ctx, roomID, h.historyLim)
	if err != nil {
		log.Printf("failed to get room history for room %d: %v", roomID, err)
		return
	}

	if len(messages) == 0 {
		return
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	history := make([]HistoryMessage, 0, len(messages))
	for _, msg := range messages {
		hm := HistoryMessage{
			Type:      "history",
			RoomID:    roomID,
			MessageID: msg.ID,
			SenderID:  msg.SenderID,
			Content:   msg.Content,
			Timestamp: msg.CreatedAt.Unix(),
		}

		if msg.MediaID != nil {
			m, err := h.getMedia(*msg.MediaID)
			if err == nil && !m.IsFlagged {
				hm.MediaURL = h.storage.GetURL(m.FilePath)
				hm.MediaType = string(m.MediaType)
			}
		}

		history = append(history, hm)
	}

	type historyPayload struct {
		Type     string           `json:"type"`
		RoomID   uint64           `json:"room_id"`
		Messages []HistoryMessage `json:"messages"`
	}

	payload := historyPayload{
		Type:     "history",
		RoomID:   roomID,
		Messages: history,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("failed to marshal history: %v", err)
		return
	}

	select {
	case client.send <- data:
	default:
		log.Printf("send buffer full for user %d, dropping history", client.userID)
	}
}
