// Package moderation is the trust-and-safety core: reports with evidence,
// moderator actions ranked by role, the live banned-word list, IP bans and
// an append-only audit log. Records here are admin-facing and carry no
// secrets, so they double as their own wire format.
package moderation

import (
	"encoding/json"
	"time"

	"github.com/v3danth/free-chat/internal/apperr"
	"github.com/v3danth/free-chat/internal/filter"
)

type TargetType string

const (
	TargetMessage TargetType = "message"
	TargetMedia   TargetType = "media"
	TargetUser    TargetType = "user"
)

type Reason string

var reasons = map[Reason]bool{
	"spam": true, "harassment": true, "nudity": true, "violence": true,
	"hate": true, "underage": true, "scam": true, "other": true,
}

type Status string

const (
	StatusOpen      Status = "open"
	StatusActioned  Status = "actioned"
	StatusDismissed Status = "dismissed"
)

type Report struct {
	ID           uint64          `json:"id"`
	ReporterID   *uint64         `json:"reporter_id"`
	TargetType   TargetType      `json:"target_type"`
	TargetID     uint64          `json:"target_id"`
	TargetUserID *uint64         `json:"target_user_id"`
	Reason       Reason          `json:"reason"`
	Note         string          `json:"note,omitempty"`
	Evidence     json.RawMessage `json:"evidence"`
	Status       Status          `json:"status"`
	HandledBy    *uint64         `json:"handled_by,omitempty"`
	HandledAt    *time.Time      `json:"handled_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// Evidence is captured when a report is filed, so moderators see exactly
// what was reported even after it is deleted or purged.
type Evidence struct {
	Message *EvidenceMessage  `json:"message,omitempty"`
	Context []EvidenceMessage `json:"context,omitempty"` // newest first
	Image   *EvidenceImage    `json:"image,omitempty"`
	User    *EvidenceUser     `json:"user,omitempty"`
}

type EvidenceMessage struct {
	ID          uint64 `json:"id"`
	RoomID      uint64 `json:"room_id,omitempty"`
	RecipientID uint64 `json:"recipient_id,omitempty"`
	SenderID    uint64 `json:"sender_id"`
	SenderName  string `json:"sender_name"`
	Body        string `json:"body"`
	MediaID     uint64 `json:"media_id,omitempty"`
}

type EvidenceImage struct {
	ID      uint64 `json:"id"`
	OwnerID uint64 `json:"owner_id"`
	URL     string `json:"url"`
}

type EvidenceUser struct {
	ID       uint64 `json:"id"`
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	Age      uint8  `json:"age"`
	About    string `json:"about"`
	Location string `json:"location"`
	Country  string `json:"country"`
}

type Word struct {
	ID        uint64        `json:"id"`
	Word      string        `json:"word"`
	Action    filter.Action `json:"action"`
	CreatedAt time.Time     `json:"created_at"`
}

type ActionKind string

const (
	ActRemoveMessage ActionKind = "remove_message"
	ActRemoveMedia   ActionKind = "remove_media"
	ActKick          ActionKind = "kick"
	ActMute          ActionKind = "mute"
	ActBan           ActionKind = "ban"
	ActUnban         ActionKind = "unban"
	ActSetRole       ActionKind = "set_role"
	ActAddWord       ActionKind = "add_word"
	ActRemoveWord    ActionKind = "remove_word"
	ActDismiss       ActionKind = "dismiss_report"
	ActAutoHide      ActionKind = "auto_hide"
)

// Action is one audit-log row. ActorID 0 means the system did it.
type Action struct {
	ID           uint64     `json:"id"`
	ActorID      uint64     `json:"actor_id"`
	Action       ActionKind `json:"action"`
	TargetUserID uint64     `json:"target_user_id,omitempty"`
	TargetName   string     `json:"target_name,omitempty"`
	TargetID     uint64     `json:"target_id,omitempty"`
	Detail       string     `json:"detail,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

var (
	ErrAlreadyReported = apperr.New(apperr.Conflict, "you already reported this")
	ErrReportNotFound  = apperr.New(apperr.NotFound, "report not found")
	ErrInvalidReport   = apperr.New(apperr.Invalid, "target_type (message, media, user), target_id and a valid reason are required")
	ErrReportSelf      = apperr.New(apperr.Invalid, "you cannot report yourself")
	ErrReportRate      = apperr.New(apperr.RateLimited, "too many reports, try again later")
	ErrNotVisible      = apperr.New(apperr.NotFound, "message not found")
	ErrSelf            = apperr.New(apperr.Invalid, "you cannot moderate yourself")
	ErrOutranked       = apperr.New(apperr.Forbidden, "you cannot moderate someone with an equal or higher role")
	ErrWordExists      = apperr.New(apperr.Conflict, "that word is already on the list")
	ErrWordNotFound    = apperr.New(apperr.NotFound, "word not found")
	ErrInvalidWord     = apperr.New(apperr.Invalid, "word must be 1-64 characters and action mask or block; phrases can only block")
	ErrMembersOnly     = apperr.New(apperr.Invalid, "only members can hold a staff role")
)
