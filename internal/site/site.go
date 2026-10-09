// Package site serves the HTML pages with the brand filled in, so the name
// comes from config and search engines see it in the title.
package site

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
)

type page struct{ Name string }

// Routes renders each page once at startup; restart to pick up HTML edits.
func Routes(mux *http.ServeMux, dir, appName string) error {
	for pattern, file := range map[string]string{"GET /{$}": "index.html", "GET /faces": "faces.html"} {
		t, err := template.ParseFiles(filepath.Join(dir, file))
		if err != nil {
			return fmt.Errorf("parse %s: %w", file, err)
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, page{Name: appName}); err != nil {
			return fmt.Errorf("render %s: %w", file, err)
		}
		body := buf.Bytes()
		mux.HandleFunc(pattern, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			w.Write(body)
		})
	}
	return nil
}
