package media

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalStorage struct {
	basePath string
	baseURL  string
}

func NewLocalStorage(cfg StorageConfig) (*LocalStorage, error) {
	if cfg.BasePath == "" {
		cfg.BasePath = "./uploads"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "/uploads"
	}

	if err := os.MkdirAll(filepath.Join(cfg.BasePath, "images"), 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(cfg.BasePath, "gifs"), 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(cfg.BasePath, "voice"), 0755); err != nil {
		return nil, err
	}

	return &LocalStorage{
		basePath: cfg.BasePath,
		baseURL:  cfg.BaseURL,
	}, nil
}

func (s *LocalStorage) Save(mediaType MediaType, reader io.Reader, ext string) (string, error) {
	var subDir string
	switch mediaType {
	case MediaTypeImage:
		subDir = "images"
	case MediaTypeGIF:
		subDir = "gifs"
	case MediaTypeVoice:
		subDir = "voice"
	default:
		subDir = "other"
	}

	filename := s.generateFilename(ext)
	relPath := filepath.Join(subDir, filename)
	fullPath := filepath.Join(s.basePath, relPath)

	file, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		os.Remove(fullPath)
		return "", err
	}

	return relPath, nil
}

func (s *LocalStorage) Delete(filePath string) error {
	fullPath := filepath.Join(s.basePath, filePath)
	return os.Remove(fullPath)
}

func (s *LocalStorage) GetURL(filePath string) string {
	return s.baseURL + "/" + filePath
}

func (s *LocalStorage) generateFilename(ext string) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp if crypto/rand fails
		return strings.ReplaceAll(
			filepath.Base(ext),
			".",
			"-",
		)
	}

	ext = strings.TrimPrefix(ext, ".")
	if ext == "" {
		ext = "bin"
	}

	return hex.EncodeToString(b) + "." + ext
}
