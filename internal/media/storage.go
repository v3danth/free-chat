package media

import "io"

type Storage interface {
	Save(mediaType MediaType, reader io.Reader, ext string) (string, error)
	Delete(filePath string) error
	GetURL(filePath string) string
}

type StorageConfig struct {
	BasePath string
	BaseURL  string
}
