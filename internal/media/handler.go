package media

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/v3danth/free-chat/internal/auth"
)

type Handler struct {
	service *Service
	jwt     *auth.JWTManager
	storage Storage
}

func NewHandler(
	service *Service,
	jwt *auth.JWTManager,
	storage Storage,
) *Handler {
	return &Handler{
		service: service,
		jwt:     jwt,
		storage: storage,
	}
}

func (h *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
	claims, err := h.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(h.service.config.MaxImageSize); err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "image file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	media, err := h.service.UploadImage(r.Context(), claims.UserID, header)
	if err != nil {
		h.handleUploadError(w, err)
		return
	}

	h.respondJSON(w, http.StatusCreated, uploadResponse{
		ID:       media.ID,
		URL:      h.storage.GetURL(media.FilePath),
		Type:     string(media.MediaType),
		FileSize: media.FileSize,
	})
}

func (h *Handler) UploadGIF(w http.ResponseWriter, r *http.Request) {
	claims, err := h.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(h.service.config.MaxGIFSize); err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("gif")
	if err != nil {
		http.Error(w, "gif file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	media, err := h.service.UploadGIF(r.Context(), claims.UserID, header)
	if err != nil {
		h.handleUploadError(w, err)
		return
	}

	h.respondJSON(w, http.StatusCreated, uploadResponse{
		ID:       media.ID,
		URL:      h.storage.GetURL(media.FilePath),
		Type:     string(media.MediaType),
		FileSize: media.FileSize,
	})
}

func (h *Handler) UploadVoice(w http.ResponseWriter, r *http.Request) {
	claims, err := h.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(h.service.config.MaxVoiceSize); err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("voice")
	if err != nil {
		http.Error(w, "voice file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	media, err := h.service.UploadVoice(r.Context(), claims.UserID, header)
	if err != nil {
		h.handleUploadError(w, err)
		return
	}

	h.respondJSON(w, http.StatusCreated, uploadResponse{
		ID:       media.ID,
		URL:      h.storage.GetURL(media.FilePath),
		Type:     string(media.MediaType),
		FileSize: media.FileSize,
	})
}

func (h *Handler) FlagMedia(w http.ResponseWriter, r *http.Request) {
	claims, err := h.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req flagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.MediaID == 0 || req.Reason == "" {
		http.Error(w, "media_id and reason are required", http.StatusBadRequest)
		return
	}

	err = h.service.FlagMedia(
		r.Context(),
		req.MediaID,
		claims.UserID,
		FlagReason(req.Reason),
	)
	if err != nil {
		h.handleFlagError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ServeMedia(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/media/"):]
	if path == "" {
		http.Error(w, "media path is required", http.StatusBadRequest)
		return
	}

	// Security: prevent directory traversal
	if containsDotDot(path) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	http.ServeFile(w, r, "./uploads/"+path)
}

func (h *Handler) authenticate(r *http.Request) (*auth.Claims, error) {
	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	if token == "" {
		return nil, http.ErrNoCookie
	}

	// Remove "Bearer " prefix if present
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	return h.jwt.Validate(token)
}

func (h *Handler) handleUploadError(w http.ResponseWriter, err error) {
	switch err {
	case ErrFileTooLarge:
		http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
	case ErrInvalidFileType:
		http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
	default:
		http.Error(w, "upload failed", http.StatusInternalServerError)
	}
}

func (h *Handler) handleFlagError(w http.ResponseWriter, err error) {
	switch err {
	case ErrMediaNotFound:
		http.Error(w, err.Error(), http.StatusNotFound)
	case ErrAlreadyFlagged:
		http.Error(w, err.Error(), http.StatusConflict)
	case ErrInvalidFlagReason:
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "failed to flag media", http.StatusInternalServerError)
	}
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func containsDotDot(path string) bool {
	return len(path) >= 3 && (path == ".." ||
		strings.Contains(path, "/..") ||
		strings.Contains(path, "../"))
}

type uploadResponse struct {
	ID       uint64 `json:"id"`
	URL      string `json:"url"`
	Type     string `json:"type"`
	FileSize uint64 `json:"file_size"`
}

type flagRequest struct {
	MediaID uint64 `json:"media_id"`
	Reason  string `json:"reason"`
}
