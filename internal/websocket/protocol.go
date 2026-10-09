package websocket

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/v3danth/free-chat/internal/apperr"
	"github.com/v3danth/free-chat/internal/id"
	"github.com/v3danth/free-chat/internal/media"
	"github.com/v3danth/free-chat/internal/message"
	"github.com/v3danth/free-chat/internal/user"
)

// ---------------------------------------------------------------------------
// Inbound boundary: raw frame → validated command.
// ---------------------------------------------------------------------------

type inbound struct {
	Type    string `json:"type"`
	RoomID  uint64 `json:"room_id"`
	To      uint64 `json:"to"`
	UserID  uint64 `json:"user_id"`
	Content string `json:"content"`
	MediaID uint64 `json:"media_id"`
}

// command is a closed set of validated client intents.
type command interface{ isCommand() }

type (
	joinCmd     struct{ room uint64 }
	leaveCmd    struct{ room uint64 }
	roomSendCmd struct {
		room    uint64
		content string
		mediaID uint64
	}
	dmCmd struct {
		to      uint64
		content string
		mediaID uint64
	}
	blockCmd   struct{ user uint64 }
	unblockCmd struct{ user uint64 }
)

func (joinCmd) isCommand()     {}
func (leaveCmd) isCommand()    {}
func (roomSendCmd) isCommand() {}
func (dmCmd) isCommand()       {}
func (blockCmd) isCommand()    {}
func (unblockCmd) isCommand()  {}

var (
	errFrame   = apperr.New(apperr.Invalid, "invalid message format")
	errRoom    = apperr.New(apperr.Invalid, "room_id is required")
	errTarget  = apperr.New(apperr.Invalid, "a user id is required")
	errEmpty   = apperr.New(apperr.Invalid, "message text or an image is required")
	errUnknown = apperr.New(apperr.Invalid, "unknown message type")
)

func parseCommand(raw []byte) (command, error) {
	var in inbound
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, errFrame
	}
	hasBody := strings.TrimSpace(in.Content) != "" || in.MediaID != 0

	switch in.Type {
	case "join", "leave":
		if in.RoomID == 0 {
			return nil, errRoom
		}
		if in.Type == "join" {
			return joinCmd{room: in.RoomID}, nil
		}
		return leaveCmd{room: in.RoomID}, nil
	case "chat", "media":
		switch {
		case in.RoomID == 0:
			return nil, errRoom
		case !hasBody:
			return nil, errEmpty
		}
		return roomSendCmd{room: in.RoomID, content: in.Content, mediaID: in.MediaID}, nil
	case "dm":
		switch {
		case in.To == 0:
			return nil, errTarget
		case !hasBody:
			return nil, errEmpty
		}
		return dmCmd{to: in.To, content: in.Content, mediaID: in.MediaID}, nil
	case "block", "unblock":
		if in.UserID == 0 {
			return nil, errTarget
		}
		if in.Type == "block" {
			return blockCmd{user: in.UserID}, nil
		}
		return unblockCmd{user: in.UserID}, nil
	}
	return nil, errUnknown
}

// ---------------------------------------------------------------------------
// Outbound boundary: domain values → wire frames.
// ---------------------------------------------------------------------------

type imageView struct {
	URL      string `json:"url"`
	ThumbURL string `json:"thumb_url"`
}

func imageOf(a media.Attachment) *imageView {
	if a.ID == 0 {
		return nil
	}
	return &imageView{URL: a.Full, ThumbURL: a.Thumb}
}

type chatEvent struct {
	Type      string     `json:"type"`
	ID        uint64     `json:"id"`
	RoomID    uint64     `json:"room_id"`
	SenderID  uint64     `json:"sender_id"`
	Name      string     `json:"name"`
	Color     string     `json:"color,omitempty"` // the sender's name colour
	Content   string     `json:"content,omitempty"`
	Image     *imageView `json:"image,omitempty"`
	Timestamp int64      `json:"ts"`
	Filtered  bool       `json:"filtered,omitempty"`
	mediaID   uint64     // for moderation lookups; never serialized
}

type dmEvent struct {
	Type      string     `json:"type"`
	ID        uint64     `json:"id"`
	From      uint64     `json:"from"`
	To        uint64     `json:"to"`
	Content   string     `json:"content,omitempty"`
	Image     *imageView `json:"image,omitempty"`
	Timestamp int64      `json:"ts"`
	// Knock marks a first message the recipient has not answered yet.
	Knock    bool `json:"knock,omitempty"`
	Filtered bool `json:"filtered,omitempty"`
}

type roomView struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

type rateLimitView struct {
	Remaining int   `json:"remaining"`
	ResetIn   int64 `json:"reset_in_seconds"`
}

type helloEvent struct {
	Type      string        `json:"type"`
	You       user.Self     `json:"you"`
	Online    []user.Card   `json:"online"`
	Rooms     []roomView    `json:"rooms"`
	RateLimit rateLimitView `json:"rate_limit"`
	// Doors are this user's private chats that survive a reconnect.
	Doors []doorView `json:"doors"`
}

type doorView struct {
	With uint64 `json:"with"`
	Open bool   `json:"open"`
	// KnockedByMe: still a knock, sent by this user (waiting on them).
	KnockedByMe bool `json:"knocked_by_me,omitempty"`
}

type presenceEvent struct {
	Type   string     `json:"type"`
	Event  string     `json:"event"` // "join" (also re-join), "update", "leave"
	UserID uint64     `json:"user_id"`
	User   *user.Card `json:"user,omitempty"`
}

type historyEvent struct {
	Type     string      `json:"type"`
	RoomID   uint64      `json:"room_id"`
	Messages []chatEvent `json:"messages"` // newest first
}

type doorEvent struct {
	Type string        `json:"type"`
	With user.Revealed `json:"with"`
}

type removedEvent struct {
	Type      string `json:"type"`
	MessageID uint64 `json:"message_id"`
}

type noticeEvent struct {
	Type  string `json:"type"`
	Code  string `json:"code"`
	Text  string `json:"text"`
	Until int64  `json:"until,omitempty"`
}

type errorEvent struct {
	Type  string `json:"type"`
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

var codeByKind = map[apperr.Kind]string{
	apperr.Invalid:      "INVALID",
	apperr.Unauthorized: "UNAUTHORIZED",
	apperr.Forbidden:    "FORBIDDEN",
	apperr.NotFound:     "NOT_FOUND",
	apperr.Conflict:     "CONFLICT",
	apperr.TooLarge:     "TOO_LARGE",
	apperr.Unsupported:  "UNSUPPORTED",
	apperr.RateLimited:  "RATE_LIMITED",
}

func roomEvent(m message.Message, color string, att media.Attachment, filtered bool) chatEvent {
	return chatEvent{
		Type:      "chat",
		ID:        m.ID,
		RoomID:    m.RoomID,
		SenderID:  m.SenderID,
		Name:      m.SenderName,
		Color:     color,
		Content:   m.Body,
		Image:     imageOf(att),
		Timestamp: id.Time(m.ID).Unix(),
		Filtered:  filtered,
		mediaID:   m.MediaID,
	}
}

func entryEvent(e message.Entry) chatEvent {
	var att media.Attachment
	if e.MediaKey != "" {
		att = media.AttachmentOf(e.MediaID, e.MediaKey)
	}
	return roomEvent(e.Message, e.SenderColor, att, false)
}

func directEvent(m message.Message, att media.Attachment, knock, filtered bool) dmEvent {
	return dmEvent{
		Type:      "dm",
		ID:        m.ID,
		From:      m.SenderID,
		To:        m.RecipientID,
		Content:   m.Body,
		Image:     imageOf(att),
		Timestamp: id.Time(m.ID).Unix(),
		Knock:     knock,
		Filtered:  filtered,
	}
}

func historyFrame(room uint64, oldestFirst []chatEvent) []byte {
	msgs := make([]chatEvent, len(oldestFirst))
	for i, ev := range oldestFirst {
		msgs[len(msgs)-1-i] = ev
	}
	return encode(historyEvent{Type: "history", RoomID: room, Messages: msgs})
}

func presenceFrame(event string, userID uint64, card *user.Card) []byte {
	return encode(presenceEvent{Type: "presence", Event: event, UserID: userID, User: card})
}

func noticeFrame(code, text string, until time.Time) []byte {
	ev := noticeEvent{Type: "notice", Code: code, Text: text}
	if !until.IsZero() {
		ev.Until = until.Unix()
	}
	return encode(ev)
}

// errorFrame exposes client-safe errors verbatim and hides everything else.
func errorFrame(err error) []byte {
	if e, ok := apperr.As(err); ok {
		return encode(errorEvent{Type: "error", Error: e.Msg, Code: codeByKind[e.Kind]})
	}
	log.Printf("websocket: internal error: %v", err)
	return encode(errorEvent{Type: "error", Error: "internal server error", Code: "INTERNAL"})
}

func encode(v any) []byte {
	// Every frame type is a plain struct of strings, numbers and slices of
	// such, which json.Marshal cannot fail on.
	data, _ := json.Marshal(v)
	return data
}
