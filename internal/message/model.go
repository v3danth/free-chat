package message

import "time"

type Message struct {
	ID        uint64
	RoomID    uint64
	SenderID  uint64
	Content   string
	MediaID   *uint64
	CreatedAt time.Time
}
