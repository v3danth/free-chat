package site

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNameIsFilledInAndEscaped(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"index.html", "faces.html"} {
		os.WriteFile(filepath.Join(dir, f), []byte(`<title>{{.Name}}</title>`), 0o600)
	}
	mux := http.NewServeMux()
	if err := Routes(mux, dir, `Night & Day <3`); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/", "/faces"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if body := rec.Body.String(); !strings.Contains(body, "Night &amp; Day &lt;3") {
			t.Errorf("%s: got %q", path, body)
		}
	}
}
