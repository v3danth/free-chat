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
// Hidden images wait in a sibling directory ("<root>-hidden") that is never
// served, so a hide can be undone.
type Storage struct {
	root, hidden string
}

func NewStorage(root string) (*Storage, error) {
	s := &Storage{root: root, hidden: filepath.Clean(root) + "-hidden"}
	for _, dir := range []string{s.root, s.hidden} {
		for _, v := range mediapath.Variants {
			if err := os.MkdirAll(filepath.Join(dir, string(v)), 0o755); err != nil {
				return nil, err
			}
		}
	}
	return s, nil
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

// DeleteAll removes every variant of key, served or hidden; missing files
// are not an error.
func (s *Storage) DeleteAll(key string) error {
	var errs []error
	for _, dir := range []string{s.root, s.hidden} {
		for _, v := range mediapath.Variants {
			err := os.Remove(s.path(dir, v, key))
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

// Hide moves every variant of key out of the served directory.
func (s *Storage) Hide(key string) error { return s.moveAll(s.root, s.hidden, key) }

// Restore moves a hidden image back; false when key was not hidden.
func (s *Storage) Restore(key string) (bool, error) {
	if _, err := os.Stat(s.path(s.hidden, mediapath.Full, key)); errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return true, s.moveAll(s.hidden, s.root, key)
}

func (s *Storage) moveAll(from, to, key string) error {
	var errs []error
	for _, v := range mediapath.Variants {
		err := os.Rename(s.path(from, v, key), s.path(to, v, key))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (s *Storage) path(dir string, v mediapath.Variant, key string) string {
	return filepath.Join(dir, filepath.FromSlash(mediapath.File(v, key)))
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
