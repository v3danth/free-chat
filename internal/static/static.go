// Package static serves the web client. Each file is read, compressed and
// hashed once per version, not once per request: app.js and the flags are
// text that gzips four to six times smaller, and serving from memory skips
// a disk read per request. A file edited on disk is picked up by its
// modification time, so a running server never serves stale code.
package static

import (
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/v3danth/free-chat/internal/httpx"
)

// maxCached bounds what is held in memory; larger files stream from disk.
const maxCached = 2 << 20

// immutablePrefixes never change once shipped, so browsers may keep them a
// year without asking again. Everything else revalidates: the app's files
// carry no version in their names, and an old app.js next to a new server
// breaks the protocol.
var immutablePrefixes = []string{"/flags/"}

type asset struct {
	mod  time.Time
	size int64
	httpx.Payload
}

type Server struct {
	root  string
	mu    sync.RWMutex
	cache map[string]*asset
}

func New(root string) *Server {
	return &Server{root: root, cache: map[string]*asset{}}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := path.Clean("/" + r.URL.Path)
	file := filepath.Join(s.root, filepath.FromSlash(name))
	info, err := os.Stat(file)
	if err == nil && info.IsDir() {
		name, file = path.Join(name, "index.html"), filepath.Join(file, "index.html")
		info, err = os.Stat(file)
	}
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", cacheControl(name))
	if info.Size() > maxCached {
		http.ServeFile(w, r, file)
		return
	}
	a, err := s.load(name, file, info)
	if err != nil {
		http.Error(w, "could not read file", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Last-Modified", a.mod.UTC().Format(http.TimeFormat))
	a.Serve(w, r)
}

func cacheControl(name string) string {
	for _, p := range immutablePrefixes {
		if strings.HasPrefix(name, p) {
			return "public, max-age=31536000, immutable"
		}
	}
	return "no-cache"
}

// load returns the cached asset for name, rebuilding it when the file on
// disk changed since it was cached.
func (s *Server) load(name, file string, info fs.FileInfo) (*asset, error) {
	s.mu.RLock()
	a := s.cache[name]
	s.mu.RUnlock()
	if a != nil && a.mod.Equal(info.ModTime()) && a.size == info.Size() {
		return a, nil
	}
	body, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	ctype := mime.TypeByExtension(path.Ext(name))
	if ctype == "" {
		ctype = http.DetectContentType(body)
	}
	a = &asset{mod: info.ModTime(), size: info.Size(), Payload: httpx.NewPayload(ctype, body)}
	s.mu.Lock()
	s.cache[name] = a
	s.mu.Unlock()
	return a, nil
}
