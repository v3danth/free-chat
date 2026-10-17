package httpx

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

// Payload is a response body prepared once and served many times: the
// static files and the rendered faces. Compressing and hashing at build
// time keeps both off the request path.
type Payload struct {
	Type string
	ETag string
	Body []byte
	Gzip []byte // nil when compressing does not pay
}

func NewPayload(contentType string, body []byte) Payload {
	sum := sha256.Sum256(body)
	p := Payload{Type: contentType, Body: body, ETag: `"` + hex.EncodeToString(sum[:8]) + `"`}
	if isText(contentType) {
		if gz := gzipped(body); len(gz) < len(body)*9/10 {
			p.Gzip = gz
		}
	}
	return p
}

// Size is what the payload holds in memory, for caches with a byte budget.
func (p Payload) Size() int { return len(p.Body) + len(p.Gzip) }

// Serve writes p, or 304 when the browser already has it. The caller sets
// Cache-Control first.
func (p Payload) Serve(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Content-Type", p.Type)
	h.Set("Vary", "Accept-Encoding")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("ETag", p.ETag)
	if etagMatches(r.Header.Get("If-None-Match"), p.ETag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	body := p.Body
	if p.Gzip != nil && acceptsGzip(r) {
		h.Set("Content-Encoding", "gzip")
		body = p.Gzip
	}
	if r.Method != http.MethodHead {
		_, _ = w.Write(body)
	}
}

func isText(ctype string) bool {
	return strings.HasPrefix(ctype, "text/") || strings.Contains(ctype, "javascript") ||
		strings.Contains(ctype, "json") || strings.Contains(ctype, "svg")
}

func gzipped(body []byte) []byte {
	var buf bytes.Buffer
	buf.Grow(len(body) / 3)
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil
	}
	if _, err := zw.Write(body); err != nil {
		return nil
	}
	if err := zw.Close(); err != nil {
		return nil
	}
	return buf.Bytes()
}

func etagMatches(header, etag string) bool {
	if header == "" {
		return false
	}
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimPrefix(strings.TrimSpace(candidate), "W/")
		if candidate == etag || candidate == "*" {
			return true
		}
	}
	return false
}

func acceptsGzip(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		enc, q, _ := strings.Cut(strings.TrimSpace(part), ";")
		if strings.EqualFold(strings.TrimSpace(enc), "gzip") && strings.TrimSpace(q) != "q=0" {
			return true
		}
	}
	return false
}
