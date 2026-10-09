// Package httpx holds the HTTP boundary helpers: decoding untrusted request
// bodies and encoding responses/errors. Handlers never touch encoding/json
// or status codes for errors directly.
package httpx

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/v3danth/free-chat/internal/apperr"
)

const maxJSONBody = 1 << 20 // 1 MiB

var statusByKind = map[apperr.Kind]int{
	apperr.Invalid:      http.StatusBadRequest,
	apperr.Unauthorized: http.StatusUnauthorized,
	apperr.Forbidden:    http.StatusForbidden,
	apperr.NotFound:     http.StatusNotFound,
	apperr.Conflict:     http.StatusConflict,
	apperr.TooLarge:     http.StatusRequestEntityTooLarge,
	apperr.Unsupported:  http.StatusUnsupportedMediaType,
	apperr.RateLimited:  http.StatusTooManyRequests,
}

var errBadBody = apperr.New(apperr.Invalid, "invalid request body")

// DecodeJSON reads a size-limited JSON body into T.
func DecodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var v T
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxJSONBody)).Decode(&v); err != nil {
		return v, errBadBody
	}
	return v, nil
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("httpx: encode response: %v", err)
	}
}

// Error writes {"error": msg}. Client-safe errors keep their message;
// anything else is logged and reported as a generic 500.
func Error(w http.ResponseWriter, err error) {
	if e, ok := apperr.As(err); ok {
		JSON(w, statusByKind[e.Kind], errorBody{e.Msg})
		return
	}
	log.Printf("httpx: internal error: %v", err)
	JSON(w, http.StatusInternalServerError, errorBody{"internal server error"})
}

type errorBody struct {
	Error string `json:"error"`
}
