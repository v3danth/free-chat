package media

import "time"

type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeGIF   MediaType = "gif"
	MediaTypeVoice MediaType = "voice"
)

type FlagReason string

const (
	FlagReasonInappropriate FlagReason = "inappropriate"
	FlagReasonSpam          FlagReason = "spam"
	FlagReasonNudity        FlagReason = "nudity"
	FlagReasonViolence      FlagReason = "violence"
	FlagReasonCopyright     FlagReason = "copyright"
	FlagReasonOther         FlagReason = "other"
)

type Media struct {
	ID         uint64
	UserID     uint64
	MediaType  MediaType
	FileName   string
	FilePath   string
	FileSize   uint64
	MimeType   string
	Width      *uint
	Height     *uint
	DurationMs *uint
	FlagCount  uint
	IsFlagged  bool
	CreatedAt  time.Time
}

type Flag struct {
	ID         uint64
	MediaID    uint64
	ReporterID uint64
	Reason     FlagReason
	CreatedAt  time.Time
}

func (m *Media) URL() string {
	return "/media/" + m.FilePath
}

func (m *Media) IsImage() bool {
	return m.MediaType == MediaTypeImage || m.MediaType == MediaTypeGIF
}

func (m *Media) IsVoice() bool {
	return m.MediaType == MediaTypeVoice
}

type UploadConfig struct {
	MaxImageSize     int64
	MaxGIFSize       int64
	MaxVoiceSize     int64
	MaxVoiceSeconds  int
	AllowedImageMIME []string
	AllowedGIFMIME   []string
	AllowedVoiceMIME []string
	FlagThreshold    uint
}

func DefaultUploadConfig() UploadConfig {
	return UploadConfig{
		MaxImageSize:    10 * 1024 * 1024, // 10MB
		MaxGIFSize:      15 * 1024 * 1024, // 15MB
		MaxVoiceSize:    5 * 1024 * 1024,  // 5MB
		MaxVoiceSeconds: 120,
		AllowedImageMIME: []string{
			"image/jpeg",
			"image/png",
			"image/webp",
		},
		AllowedGIFMIME: []string{
			"image/gif",
		},
		AllowedVoiceMIME: []string{
			"audio/webm",
			"audio/ogg",
			"audio/mp4",
			"audio/mpeg",
		},
		FlagThreshold: 3,
	}
}
