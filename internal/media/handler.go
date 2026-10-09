package media

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/v3danth/free-chat/internal/auth"
	"github.com/v3danth/free-chat/internal/httpx"
	"github.com/v3danth/free-chat/internal/mediapath"
)

const multipartOverhead = 64 << 10 // headers and boundaries around the file part

func Routes(mux *http.ServeMux, svc *Service, authSvc *auth.Service, ipOf httpx.IPResolver, maxUpload int64) {
	h := handler{svc: svc, maxUpload: maxUpload}
	mux.Handle("POST /media/upload", authSvc.RequireBearer(ipOf, h.upload))
	mux.Handle("GET "+mediapath.URLPrefix, http.StripPrefix(mediapath.URLPrefix, staticFiles(svc.storage.Root())))
}

type handler struct {
	svc       *Service
	maxUpload int64
}

func (h handler) upload(w http.ResponseWriter, r *http.Request, id auth.Identity) {
	if err := h.svc.Admit(id.UserID); err != nil {
		httpx.Error(w, err)
		return
	}
	raw, err := h.readUpload(w, r)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	m, err := h.svc.Upload(r.Context(), id.UserID, raw)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	a := AttachmentOf(m.ID, m.Key)
	httpx.JSON(w, http.StatusCreated, uploadView{ID: m.ID, URL: a.Full, ThumbURL: a.Thumb, Width: m.Width, Height: m.Height})
}

// readUpload is the upload boundary: bytes are capped on the wire and the
// single "image" part is streamed, never spilled to temp files.
func (h handler) readUpload(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUpload+multipartOverhead)
	mr, err := r.MultipartReader()
	if err != nil {
		return nil, ErrMissingFile
	}
	for {
		part, err := mr.NextPart()
		if err != nil {
			return nil, tooLargeOr(err, ErrMissingFile)
		}
		if part.FormName() != "image" {
			continue
		}
		raw, err := io.ReadAll(io.LimitReader(part, h.maxUpload+1))
		if err != nil {
			return nil, tooLargeOr(err, ErrMissingFile)
		}
		if int64(len(raw)) > h.maxUpload {
			return nil, ErrTooLarge
		}
		return raw, nil
	}
}

func tooLargeOr(err, fallback error) error {
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return ErrTooLarge
	}
	return fallback
}

// staticFiles serves image variants straight from disk (sendfile). Names
// are random and never reused, so responses are immutable; removing an
// image deletes its files.
func staticFiles(root string) http.Handler {
	fs := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fs.ServeHTTP(w, r)
	})
}

type uploadView struct {
	ID       uint64 `json:"id"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumb_url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}
