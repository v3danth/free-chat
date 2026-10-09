package media

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/v3danth/free-chat/internal/mediapath"
)

// Storage writes image variants under one root directory, laid out exactly
// as mediapath describes, so a static file server can serve the root as is.
type Storage struct {
	root string
}

func NewStorage(root string) (*Storage, error) {
	for _, v := range mediapath.Variants {
		if err := os.MkdirAll(filepath.Join(root, string(v)), 0o755); err != nil {
			return nil, err
		}
	}
	return &Storage{root: root}, nil
}

func (s *Storage) Root() string { return s.root }

// SaveAll writes every variant of key, or none of them.
func (s *Storage) SaveAll(key string, files map[mediapath.Variant][]byte) error {
	for v, data := range files {
		if err := s.write(mediapath.File(v, key), data); err != nil {
			s.DeleteAll(key)
			return err
		}
	}
	return nil
}

// DeleteAll removes every variant of key; missing files are not an error.
func (s *Storage) DeleteAll(key string) error {
	var errs []error
	for _, v := range mediapath.Variants {
		err := os.Remove(filepath.Join(s.root, filepath.FromSlash(mediapath.File(v, key))))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (s *Storage) write(rel string, data []byte) error {
	f, err := os.OpenFile(filepath.Join(s.root, filepath.FromSlash(rel)), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, werr := f.Write(data)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	return werr
}
