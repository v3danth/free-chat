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
		// Each decode can take up to maxDecoded bytes; two at a time bounds memory.
		cpu:     make(chan struct{}, min(2, runtime.NumCPU())),
		uploads: ratelimit.New(ratelimit.Config{Rate: 20, Window: 10 * time.Minute}),
	}
}

// Admit spends one of owner's uploads; call it before reading the body, so
// a refused upload never holds its bytes in memory.
func (s *Service) Admit(owner uint64) error {
	if !s.uploads.Allow(owner) {
		return ErrUploadRate
	}
	return nil
}

// Upload processes and stores an image; the caller has called Admit.
func (s *Service) Upload(ctx context.Context, owner uint64, raw []byte) (Media, error) {

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
		mediapath.Full: p.Full, mediapath.Thumb: p.Thumb,
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

// Hide takes an image off the site but keeps its files, so it can be
// restored: used when enough people report it, pending a moderator.
func (s *Service) Hide(ctx context.Context, id uint64) (Media, error) {
	m, err := s.repo.MarkRemoved(ctx, id)
	if err != nil {
		return Media{}, err
	}
	return m, s.storage.Hide(m.Key)
}

// Restore brings back an image Hide took down; false if it was not hidden
// (never reported enough, or already removed for good by a moderator).
func (s *Service) Restore(ctx context.Context, id uint64) (bool, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return false, err
	}
	restored, err := s.storage.Restore(m.Key)
	if err != nil || !restored {
		return false, err
	}
	return true, s.repo.ClearRemoved(ctx, id)
}

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
