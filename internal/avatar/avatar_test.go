package avatar

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// testdata/golden.json was produced by the browser prototype. Matching it
// proves the same name draws the same face in Go and in JavaScript, and
// guards against any change that would silently redraw existing faces.
func TestMatchesPrototype(t *testing.T) {
	raw, err := os.ReadFile("testdata/golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden []struct {
		Name, Normalized, Tier, Species, Grid, Blink string
		Seed                                         uint64
	}
	if err := json.Unmarshal(raw, &golden); err != nil {
		t.Fatal(err)
	}
	fingerprint := func(g grid) string {
		sum := sha256.Sum256([]byte(strings.Join(g[:], ",")))
		return hex.EncodeToString(sum[:])[:16]
	}
	for _, want := range golden {
		tr := For(want.Name)
		got := struct{ Normalized, Tier, Species, Grid, Blink string }{
			tr.Name, tr.Tier, tr.Species.Key, fingerprint(paintFace(tr, false)), fingerprint(paintFace(tr, true)),
		}
		exp := struct{ Normalized, Tier, Species, Grid, Blink string }{
			want.Normalized, want.Tier, want.Species, want.Grid, want.Blink,
		}
		if tr.Seed != want.Seed || got != exp {
			t.Errorf("%q: got seed %d %+v, want seed %d %+v", want.Name, tr.Seed, got, want.Seed, exp)
		}
	}
}

func TestSameNameSameFace(t *testing.T) {
	a, _ := PNG(For("Night_Owl "), 1, false)
	b, _ := PNG(For("night_owl"), 1, false)
	c, _ := PNG(For("night_owl2"), 1, false)
	if !bytes.Equal(a, b) {
		t.Error("case and spacing changed the face")
	}
	if bytes.Equal(a, c) {
		t.Error("different names drew the same face")
	}
}

func TestEveryFaceStaysInsideItsRules(t *testing.T) {
	inRange := func(v int, r [2]int) bool { return v >= r[0] && v <= r[1] }
	for i := range 5000 {
		tr := For("rules_" + strings.Repeat("x", i%7) + string(rune('a'+i%26)) + string(rune('0'+i%10)) + strings.Repeat("y", i/260))
		sp := tr.Species
		if !inRange(tr.HeadW, sp.HeadW) || !inRange(tr.HeadH, sp.HeadH) || !inRange(tr.EarSize, sp.EarSize) ||
			!inRange(tr.EyeGap, sp.EyeGap) || !inRange(tr.EyeY, sp.EyeY) {
			t.Fatalf("%s: a size is outside the %s rules: %+v", tr.Name, sp.Key, tr)
		}
		if tr.Round < sp.Round[0] || tr.Round > sp.Round[1] {
			t.Fatalf("%s: roundness %v outside %v", tr.Name, tr.Round, sp.Round)
		}
		lim := eyeLimits[tr.EyeShape]
		if tr.EyeSize < lim[0] || tr.EyeSize > lim[1] {
			t.Fatalf("%s: %s eye size %d outside %v", tr.Name, tr.EyeShape, tr.EyeSize, lim)
		}
		for _, banned := range sp.noAcc {
			if tr.Acc == banned {
				t.Fatalf("%s: a %s wearing a %s", tr.Name, sp.Key, banned)
			}
		}
	}
}

func TestEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	Routes(mux, StylePoly)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", URL("Night Owl")+"?scale=4", nil))
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("png: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	img, err := png.Decode(rec.Body)
	if err != nil || img.Bounds().Dx() != Size*4 {
		t.Fatalf("decode: %v, width %d", err, img.Bounds().Dx())
	}
	if !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Error("faces must be cached forever")
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", URL("night owl")+"/info", nil))
	var info Summary
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil || info.Name != "night owl" || info.Species == "" {
		t.Fatalf("info: %d %s", rec.Code, rec.Body)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/avatar/rules", nil))
	var rules []Rule
	if err := json.Unmarshal(rec.Body.Bytes(), &rules); err != nil || len(rules) != len(species) {
		t.Fatalf("rules: %d %v", rec.Code, err)
	}
	for _, r := range rules {
		if For(r.Example).Species.Key != r.Species {
			t.Errorf("example %q does not draw a %s", r.Example, r.Species)
		}
	}

	for _, bad := range []string{"?scale=0", "?scale=17", "?scale=x"} {
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", URL("a")+bad, nil))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", bad, rec.Code)
		}
	}
}
