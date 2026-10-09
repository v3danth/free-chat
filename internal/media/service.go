package media

import (
	"context"
	"crypto/rand"
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/v3danth/free-chat/internal/mediapath"
	"github.com/v3danth/free-chat/internal/ratelimit"
)

type Service struct {
	repo    Repository
	storage *Storage
	// cpu bounds concurrent decodes so a burst of uploads cannot starve chat.
	cpu     chan struct{}
	uploads *ratelimit.Limiter
}

func NewService(repo Repository, storage *Storage) *Service {
	return &Service{
		repo:    repo,
		storage: storage,
		cpu:     make(chan struct{}, runtime.NumCPU()),
		uploads: ratelimit.New(ratelimit.Config{Rate: 20, Window: 10 * time.Minute}),
	}
}

func (s *Service) Upload(ctx context.Context, owner uint64, raw []byte) (Media, error) {
	if !s.uploads.Allow(owner) {
		return Media{}, ErrUploadRate
	}

	select {
	case s.cpu <- struct{}{}:
	case <-ctx.Done():
		return Media{}, ctx.Err()
	}
	p, err := Process(raw)
	<-s.cpu
	if err != nil {
		return Media{}, err
	}

	banned, err := s.repo.IsHashBanned(ctx, p.SHA256[:])
	if err != nil {
		return Media{}, err
	}
	if banned {
		return Media{}, ErrBannedImage
	}

	// Random, unguessable names: a private photo's URL is its capability.
	key := strings.ToLower(rand.Text())
	if err := s.storage.SaveAll(key, map[mediapath.Variant][]byte{
		mediapath.Full: p.Full, mediapath.Thumb: p.Thumb, mediapath.Blur: p.Blur,
	}); err != nil {
		return Media{}, err
	}

	m, err := s.repo.Create(ctx, Media{
		OwnerID: owner,
		Key:     key,
		SHA256:  p.SHA256[:],
		Width:   p.Width,
		Height:  p.Height,
		Bytes:   len(p.Full),
	})
	if err != nil {
		s.deleteFiles(key)
		return Media{}, err
	}
	return m, nil
}

// Attach clears an image for sharing by its owner.
func (s *Service) Attach(ctx context.Context, mediaID, owner uint64) (Attachment, error) {
	m, err := s.repo.GetByID(ctx, mediaID)
	if err != nil {
		return Attachment{}, err
	}
	if m.OwnerID != owner {
		return Attachment{}, ErrNotOwner
	}
	if m.RemovedAt != nil {
		return Attachment{}, ErrRemoved
	}
	return AttachmentOf(m.ID, m.Key), nil
}

func (s *Service) Get(ctx context.Context, id uint64) (Media, error) { return s.repo.GetByID(ctx, id) }

// Remove takes an image down everywhere: the row is marked removed, the
// files are deleted, and with banHash the exact file can never return.
func (s *Service) Remove(ctx context.Context, id, actorID uint64, banHash bool) (Media, error) {
	m, err := s.repo.MarkRemoved(ctx, id)
	if err != nil {
		return Media{}, err
	}
	if banHash {
		if err := s.repo.BanHash(ctx, m.SHA256, actorID); err != nil {
			return Media{}, err
		}
	}
	s.deleteFiles(m.Key)
	return m, nil
}

// DeleteFiles removes a key's files; used when the rows are already gone.
func (s *Service) DeleteFiles(key string) { s.deleteFiles(key) }

func (s *Service) deleteFiles(key string) {
	if err := s.storage.DeleteAll(key); err != nil {
		log.Printf("media: delete files for %s: %v", key, err)
	}
}
