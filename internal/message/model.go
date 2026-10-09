package message

import "github.com/v3danth/free-chat/internal/apperr"

// Message is one row of the log. Exactly one of RoomID and RecipientID is
// non-zero; use NewRoom / NewDirect so that holds by construction.
type Message struct {
	ID          uint64
	RoomID      uint64
	RecipientID uint64
	SenderID    uint64
	SenderName  string
	Body        string
	MediaID     uint64
}

func NewRoom(id, room, sender uint64, senderName, body string, mediaID uint64) Message {
	return Message{ID: id, RoomID: room, SenderID: sender, SenderName: senderName, Body: body, MediaID: mediaID}
}

func NewDirect(id, recipient, sender uint64, senderName, body string, mediaID uint64) Message {
	return Message{ID: id, RecipientID: recipient, SenderID: sender, SenderName: senderName, Body: body, MediaID: mediaID}
}

func (m Message) IsDirect() bool { return m.RecipientID != 0 }

// Involves reports whether user may see m (a room message is public).
func (m Message) Involves(user uint64) bool {
	return !m.IsDirect() || m.SenderID == user || m.RecipientID == user
}

// Entry is a stored message plus its image's file key while not removed.
type Entry struct {
	Message
	MediaKey     string
	SenderGender string // "" once the sender's account is gone
}

type Room struct {
	ID   uint64
	Slug string
	Name string
}

var ErrNotFound = apperr.New(apperr.NotFound, "message not found")
