package static

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newSite(t *testing.T) (string, *Server) {
	t.Helper()
	root := t.TempDir()
	for name, body := range map[string]string{
		"index.html":    "<!doctype html><title>Drift</title>",
		"app.js":        strings.Repeat("console.log('drift');\n", 200),
		"flags/in.svg":  strings.Repeat("<svg><rect/></svg>", 50),
		"../secret.txt": "nope",
	} {
		file := filepath.Join(root, "web", name)
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(root, "web"), New(filepath.Join(root, "web"))
}

func get(s *Server, target string, header map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec
}

func TestServesGzipToClientsThatAcceptIt(t *testing.T) {
	_, s := newSite(t)
	rec := get(s, "/app.js", map[string]string{"Accept-Encoding": "gzip, br"})
	if rec.Header().Get("Content-Encoding") != "gzip" || rec.Header().Get("Vary") != "Accept-Encoding" {
		t.Fatalf("headers: %v", rec.Header())
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	plain, _ := io.ReadAll(zr)
	if !bytes.HasPrefix(plain, []byte("console.log")) {
		t.Fatalf("body: %q", plain[:20])
	}
	if rec := get(s, "/app.js", nil); rec.Header().Get("Content-Encoding") != "" {
		t.Fatal("no Accept-Encoding must get plain text")
	}
}

func TestRevalidatesAppFilesButKeepsFlagsForAYear(t *testing.T) {
	_, s := newSite(t)
	if cc := get(s, "/", nil).Header().Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("index: %q", cc)
	}
	if cc := get(s, "/flags/in.svg", nil).Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Fatalf("flag: %q", cc)
	}
	etag := get(s, "/app.js", nil).Header().Get("ETag")
	if rec := get(s, "/app.js", map[string]string{"If-None-Match": etag}); rec.Code != http.StatusNotModified || rec.Body.Len() != 0 {
		t.Fatalf("revalidation: %d, %d bytes", rec.Code, rec.Body.Len())
	}
}

func TestPicksUpEditsOnDisk(t *testing.T) {
	root, s := newSite(t)
	before := get(s, "/app.js", nil).Header().Get("ETag")
	file := filepath.Join(root, "app.js")
	if err := os.WriteFile(file, []byte("console.log('new');"), 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Second)
	if err := os.Chtimes(file, later, later); err != nil {
		t.Fatal(err)
	}
	rec := get(s, "/app.js", nil)
	if rec.Header().Get("ETag") == before || rec.Body.String() != "console.log('new');" {
		t.Fatalf("stale file served: %q", rec.Body.String())
	}
}

func TestStaysInsideRoot(t *testing.T) {
	_, s := newSite(t)
	for _, target := range []string{"/../secret.txt", "/flags/../../secret.txt", "/missing.js"} {
		if rec := get(s, target, nil); rec.Code != http.StatusNotFound {
			t.Fatalf("%s: %d", target, rec.Code)
		}
	}
}
