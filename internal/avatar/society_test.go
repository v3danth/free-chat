package avatar

import (
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

// testdata/golden_v2.json was produced by the society marks prototype. A
// match means the same name draws the same mark in Go and in JavaScript;
// a mismatch after a change means existing marks would be redrawn.
func TestMarksMatchPrototype(t *testing.T) {
	raw, err := os.ReadFile("testdata/golden_v2.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden []struct {
		Name, Rank, Animal, Metal, Title, Number, Pixels string
		Seed                                             uint64
	}
	if err := json.Unmarshal(raw, &golden); err != nil {
		t.Fatal(err)
	}
	for _, want := range golden {
		m := For2(want.Name)
		g := markGrid(m)
		sum := sha256.Sum256([]byte(strings.Join(g[:], ",")))
		got := []string{m.Rank, m.Animal.Key, m.Metal.Name, m.Title, m.Number, hex.EncodeToString(sum[:])[:16]}
		exp := []string{want.Rank, want.Animal, want.Metal, want.Title, want.Number, want.Pixels}
		if m.Seed != want.Seed || strings.Join(got, "|") != strings.Join(exp, "|") {
			t.Errorf("%q:\n got %v\nwant %v", want.Name, got, exp)
		}
	}
}

func TestMarkEndpoints(t *testing.T) {
	mux := http.NewServeMux()
	Routes(mux, StylePixel)
	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		return rec
	}

	if rec := get(URLV2("night owl")); rec.Header().Get("Content-Type") != "image/png" || strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("default style: %s, %s", rec.Header().Get("Content-Type"), rec.Header().Get("Cache-Control"))
	}
	rec := get(URLV2("night owl") + "?style=pixel&scale=4")
	if img, err := png.Decode(rec.Body); err != nil || img.Bounds().Dx() != markPixels*4 {
		t.Fatalf("pixel: %v", err)
	}
	rec = get(URLV2("night owl") + "?style=poly")
	if body := rec.Body.String(); rec.Header().Get("Content-Type") != "image/svg+xml" || !strings.HasPrefix(body, "<svg") || strings.Contains(body, "night owl") {
		t.Fatalf("poly: %s %.80s", rec.Header().Get("Content-Type"), body)
	}
	var info MarkSummary
	if err := json.Unmarshal(get(URLV2("Night Owl")+"/info").Body.Bytes(), &info); err != nil || info.Name != "night owl" || info.Rank == "" || len(info.Number) != 4 {
		t.Fatalf("info: %+v %v", info, err)
	}
	var parts MarkParts
	if err := json.Unmarshal(get("/avatar/parts").Body.Bytes(), &parts); err != nil || len(parts.Animals) != len(maskAnimals) || parts.Ranks["initiate"] != 0.75 {
		t.Fatalf("parts: %+v %v", parts, err)
	}
	if rec := get(URLV2("a") + "?style=oil"); rec.Code != http.StatusBadRequest {
		t.Errorf("bad style: %d", rec.Code)
	}
}

func TestGoldIsOnlyForGrandmasters(t *testing.T) {
	for i := range 3000 {
		m := For2("rank_check_" + strings.Repeat("z", i%5) + string(rune('a'+i%26)) + string(rune('0'+i/26%10)) + string(rune('A'+i/260)))
		if (m.Metal.Name == "Gold") != (m.Rank == "grandmaster") {
			t.Fatalf("%s: %s metal for a %s", m.Name, m.Metal.Name, m.Rank)
		}
	}
}
