package avatar

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/url"
	"strconv"

	"github.com/v3danth/free-chat/internal/apperr"
	"github.com/v3danth/free-chat/internal/httpx"
)

const (
	maxNameBytes = 128
	maxScale     = 16 // 24 x 16 = 384 px, enough for a story card
)

var (
	errName  = apperr.New(apperr.Invalid, "name must be at most 128 bytes")
	errScale = apperr.New(apperr.Invalid, "scale must be between 1 and 16")
)

// URL is where the face for name is served. The version keeps old cached
// images from being reused after the art changes.
func URL(name string) string {
	return "/avatar/" + Version + "/" + url.PathEscape(Normalize(name))
}

// Routes serves faces. Faces are pure functions of the path, so they
// are cacheable forever.
//
//	GET /avatar/v1/{name}?scale=1..16&blink=1  PNG, 24 px per scale step
//	GET /avatar/v1/{name}/info                 JSON description of the face
//	GET /avatar/rules                          the rules every animal follows
func Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /avatar/"+Version+"/{name}", servePNG)
	mux.HandleFunc("GET /avatar/"+Version+"/{name}/info", serveInfo)
	mux.HandleFunc("GET /avatar/rules", serveRules)
}

func nameOf(r *http.Request) (string, error) {
	name := r.PathValue("name")
	if len(name) > maxNameBytes {
		return "", errName
	}
	return name, nil
}

func servePNG(w http.ResponseWriter, r *http.Request) {
	name, err := nameOf(r)
	scale := 1
	if err == nil && r.URL.Query().Has("scale") {
		if scale, err = strconv.Atoi(r.URL.Query().Get("scale")); err != nil || scale < 1 || scale > maxScale {
			err = errScale
		}
	}
	if err != nil {
		httpx.Error(w, err)
		return
	}
	data, err := PNG(For(name), scale, r.URL.Query().Get("blink") == "1")
	if err != nil {
		httpx.Error(w, err)
		return
	}
	cacheForever(w)
	w.Header().Set("Content-Type", "image/png")
	w.Write(data)
}

func serveInfo(w http.ResponseWriter, r *http.Request) {
	name, err := nameOf(r)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	cacheForever(w)
	httpx.JSON(w, http.StatusOK, For(name).Summary())
}

func cacheForever(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func serveRules(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=3600")
	httpx.JSON(w, http.StatusOK, Rules())
}

// PNG renders a face, each pixel scaled to a scale x scale block so the
// pixel art stays sharp at any size. blink draws the eyes closed.
func PNG(t Traits, scale int, blink bool) ([]byte, error) {
	g := paintFace(t, blink)
	img := image.NewNRGBA(image.Rect(0, 0, Size*scale, Size*scale))
	for y := range Size {
		for x := range Size {
			hex := g[y*Size+x]
			if hex == "" {
				hex = t.background(x, y)
			}
			c := parseHex(hex)
			for dy := range scale {
				for dx := range scale {
					img.SetNRGBA(x*scale+dx, y*scale+dy, c)
				}
			}
		}
	}
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// parseHex reads "#RRGGBB"; every colour in this package is written that way.
func parseHex(s string) color.NRGBA {
	v, _ := strconv.ParseUint(s[1:], 16, 32)
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xFF}
}
