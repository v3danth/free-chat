package websocket

type MessageType string

const (
	MessageTypeChat  MessageType = "chat"
	MessageTypeJoin  MessageType = "join"
	MessageTypeLeave MessageType = "leave"
	MessageTypeMedia MessageType = "media"
)

type Message struct {
	Type    MessageType `json:"type"`
	RoomID  uint64      `json:"room_id,omitempty"`
	Content string      `json:"content,omitempty"`
	MediaID uint64      `json:"media_id,omitempty"`
}

type OutgoingMessage struct {
	Type        string `json:"type"`
	RoomID      uint64 `json:"room_id"`
	MessageID   uint64 `json:"message_id,omitempty"`
	SenderID    uint64 `json:"sender_id"`
	Username    string `json:"username"`
	Content     string `json:"content,omitempty"`
	MediaURL    string `json:"media_url,omitempty"`
	MediaType   string `json:"media_type,omitempty"`
	Timestamp   int64  `json:"timestamp,omitempty"`
	RateLimited bool   `json:"rate_limited,omitempty"`
	Filtered    bool   `json:"filtered,omitempty"`
}

type HistoryMessage struct {
	Type      string `json:"type"`
	RoomID    uint64 `json:"room_id"`
	MessageID uint64 `json:"message_id"`
	SenderID  uint64 `json:"sender_id"`
	Username  string `json:"username"`
	Content   string `json:"content,omitempty"`
	MediaURL  string `json:"media_url,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

type RateLimitInfo struct {
	Type      string `json:"type"`
	Remaining int    `json:"remaining"`
	ResetIn   int64  `json:"reset_in_seconds"`
}

type ErrorMessage struct {
	Type  string `json:"type"`
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}
