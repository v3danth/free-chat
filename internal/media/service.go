package media

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
)

var (
	ErrFileTooLarge      = errors.New("file size exceeds limit")
	ErrInvalidFileType   = errors.New("invalid file type")
	ErrMediaNotFound     = errors.New("media not found")
	ErrAlreadyFlagged    = errors.New("already flagged by this user")
	ErrInvalidFlagReason = errors.New("invalid flag reason")
)

type Service struct {
	repo    Repository
	storage Storage
	config  UploadConfig
}

func NewService(repo Repository, storage Storage, config UploadConfig) *Service {
	if config.MaxImageSize == 0 {
		config.MaxImageSize = DefaultUploadConfig().MaxImageSize
	}
	if config.MaxGIFSize == 0 {
		config.MaxGIFSize = DefaultUploadConfig().MaxGIFSize
	}
	if config.MaxVoiceSize == 0 {
		config.MaxVoiceSize = DefaultUploadConfig().MaxVoiceSize
	}
	if config.FlagThreshold == 0 {
		config.FlagThreshold = DefaultUploadConfig().FlagThreshold
	}
	if len(config.AllowedImageMIME) == 0 {
		config.AllowedImageMIME = DefaultUploadConfig().AllowedImageMIME
	}
	if len(config.AllowedGIFMIME) == 0 {
		config.AllowedGIFMIME = DefaultUploadConfig().AllowedGIFMIME
	}
	if len(config.AllowedVoiceMIME) == 0 {
		config.AllowedVoiceMIME = DefaultUploadConfig().AllowedVoiceMIME
	}

	return &Service{
		repo:    repo,
		storage: storage,
		config:  config,
	}
}

func (s *Service) UploadImage(
	ctx context.Context,
	userID uint64,
	file *multipart.FileHeader,
) (*Media, error) {
	if file.Size > s.config.MaxImageSize {
		return nil, fmt.Errorf("%w: max %d bytes", ErrFileTooLarge, s.config.MaxImageSize)
	}

	if !s.isAllowedMIME(file.Header.Get("Content-Type"), s.config.AllowedImageMIME) {
		return nil, ErrInvalidFileType
	}

	return s.uploadFile(ctx, userID, MediaTypeImage, file)
}

func (s *Service) UploadGIF(
	ctx context.Context,
	userID uint64,
	file *multipart.FileHeader,
) (*Media, error) {
	if file.Size > s.config.MaxGIFSize {
		return nil, fmt.Errorf("%w: max %d bytes", ErrFileTooLarge, s.config.MaxGIFSize)
	}

	if !s.isAllowedMIME(file.Header.Get("Content-Type"), s.config.AllowedGIFMIME) {
		return nil, ErrInvalidFileType
	}

	return s.uploadFile(ctx, userID, MediaTypeGIF, file)
}

func (s *Service) UploadVoice(
	ctx context.Context,
	userID uint64,
	file *multipart.FileHeader,
) (*Media, error) {
	if file.Size > s.config.MaxVoiceSize {
		return nil, fmt.Errorf("%w: max %d bytes", ErrFileTooLarge, s.config.MaxVoiceSize)
	}

	if !s.isAllowedMIME(file.Header.Get("Content-Type"), s.config.AllowedVoiceMIME) {
		return nil, ErrInvalidFileType
	}

	return s.uploadFile(ctx, userID, MediaTypeVoice, file)
}

func (s *Service) uploadFile(
	ctx context.Context,
	userID uint64,
	mediaType MediaType,
	file *multipart.FileHeader,
) (*Media, error) {
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	ext := filepath.Ext(file.Filename)
	relPath, err := s.storage.Save(mediaType, src, ext)
	if err != nil {
		return nil, err
	}

	m := &Media{
		UserID:    userID,
		MediaType: mediaType,
		FileName:  file.Filename,
		FilePath:  relPath,
		FileSize:  uint64(file.Size),
		MimeType:  file.Header.Get("Content-Type"),
	}

	if err := s.repo.CreateMedia(ctx, m); err != nil {
		s.storage.Delete(relPath)
		return nil, err
	}

	return m, nil
}

func (s *Service) GetMedia(ctx context.Context, id uint64) (*Media, error) {
	m, err := s.repo.GetMediaByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrMediaNotFound
	}
	return m, nil
}

func (s *Service) FlagMedia(
	ctx context.Context,
	mediaID,
	reporterID uint64,
	reason FlagReason,
) error {
	if !isValidFlagReason(reason) {
		return ErrInvalidFlagReason
	}

	// Check if media exists
	media, err := s.repo.GetMediaByID(ctx, mediaID)
	if err != nil {
		return err
	}
	if media == nil {
		return ErrMediaNotFound
	}

	// Check if user already flagged
	alreadyFlagged, err := s.repo.HasUserFlagged(ctx, mediaID, reporterID)
	if err != nil {
		return err
	}
	if alreadyFlagged {
		return ErrAlreadyFlagged
	}

	// Create flag
	flag := &Flag{
		MediaID:    mediaID,
		ReporterID: reporterID,
		Reason:     reason,
	}

	if err := s.repo.CreateFlag(ctx, flag); err != nil {
		return err
	}

	// Increment flag count
	if err := s.repo.IncrementFlagCount(ctx, mediaID); err != nil {
		return err
	}

	// Check if should auto-flag
	media.FlagCount++
	if media.FlagCount >= s.config.FlagThreshold && !media.IsFlagged {
		if err := s.repo.SetFlagged(ctx, mediaID, true); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) DeleteMedia(ctx context.Context, id, userID uint64) error {
	media, err := s.repo.GetMediaByID(ctx, id)
	if err != nil {
		return err
	}
	if media == nil {
		return ErrMediaNotFound
	}

	// Only allow owner or admin to delete
	if media.UserID != userID {
		return errors.New("unauthorized")
	}

	if err := s.repo.DeleteMedia(ctx, id); err != nil {
		return err
	}

	return s.storage.Delete(media.FilePath)
}

func (s *Service) isAllowedMIME(mimeType string, allowed []string) bool {
	for _, a := range allowed {
		if strings.EqualFold(mimeType, a) {
			return true
		}
	}
	return false
}

func isValidFlagReason(reason FlagReason) bool {
	switch reason {
	case FlagReasonInappropriate,
		FlagReasonSpam,
		FlagReasonNudity,
		FlagReasonViolence,
		FlagReasonCopyright,
		FlagReasonOther:
		return true
	default:
		return false
	}
}
