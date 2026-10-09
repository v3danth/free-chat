package media

import (
	"time"

	"github.com/v3danth/free-chat/internal/apperr"
	"github.com/v3danth/free-chat/internal/mediapath"
)

type Media struct {
	ID        uint64
	OwnerID   uint64
	Key       string
	SHA256    []byte
	Width     int
	Height    int
	Bytes     int
	RemovedAt *time.Time
	CreatedAt time.Time
}

// Attachment is an image cleared for sharing in chat.
type Attachment struct {
	ID    uint64
	Full  string
	Thumb string
}

func AttachmentOf(id uint64, key string) Attachment {
	return Attachment{ID: id, Full: mediapath.URL(mediapath.Full, key), Thumb: mediapath.URL(mediapath.Thumb, key)}
}

var (
	ErrNotFound    = apperr.New(apperr.NotFound, "image not found")
	ErrTooLarge    = apperr.New(apperr.TooLarge, "image is too large")
	ErrInvalidType = apperr.New(apperr.Unsupported, "only JPEG, PNG, GIF and WebP images are allowed")
	ErrBannedImage = apperr.New(apperr.Forbidden, "this image is not allowed")
	ErrNotOwner    = apperr.New(apperr.Forbidden, "you can only share images you uploaded")
	ErrRemoved     = apperr.New(apperr.Forbidden, "this image was removed")
	ErrMissingFile = apperr.New(apperr.Invalid, "an image file is required")
	ErrUploadRate  = apperr.New(apperr.RateLimited, "too many uploads, try again in a few minutes")
)
