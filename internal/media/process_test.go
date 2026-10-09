package media

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func encodePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for x := range w {
		img.Set(x, 0, color.NRGBA{R: 200, A: 255})
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func dims(t *testing.T, data []byte) (int, int) {
	t.Helper()
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != "jpeg" {
		t.Fatalf("variant is not a JPEG: %v %s", err, format)
	}
	return cfg.Width, cfg.Height
}

func TestProcessVariants(t *testing.T) {
	p, err := Process(encodePNG(t, 2000, 1000))
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []struct {
		name string
		data []byte
		w, h int
	}{{"full", p.Full, 1280, 640}, {"thumb", p.Thumb, 320, 160}} {
		if w, h := dims(t, v.data); w != v.w || h != v.h {
			t.Errorf("%s = %dx%d, want %dx%d", v.name, w, h, v.w, v.h)
		}
	}
	if p.Width != 1280 || p.Height != 640 {
		t.Errorf("reported size %dx%d", p.Width, p.Height)
	}

	small, err := Process(encodePNG(t, 100, 50))
	if err != nil {
		t.Fatal(err)
	}
	if w, h := dims(t, small.Full); w != 100 || h != 50 {
		t.Errorf("small images must not be upscaled: %dx%d", w, h)
	}
}

func TestProcessStripsMetadata(t *testing.T) {
	var buf bytes.Buffer
	jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 64, 64)), nil)
	// Splice an EXIF APP1 segment with a fake GPS tag right after SOI.
	payload := []byte("Exif\x00\x00GPSLatitude=19.07N")
	app1 := append([]byte{0xFF, 0xE1, 0x00, byte(len(payload) + 2)}, payload...)
	raw := append(append([]byte{}, buf.Bytes()[:2]...), append(app1, buf.Bytes()[2:]...)...)

	p, err := Process(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range [][]byte{p.Full, p.Thumb} {
		if bytes.Contains(v, []byte("GPSLatitude")) || bytes.Contains(v, []byte("Exif")) {
			t.Fatal("re-encoded image still carries EXIF data")
		}
	}
}

func TestProcessRejects(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
		want error
	}{
		{"html disguised as image", []byte("<!DOCTYPE html><script>alert(1)</script>"), ErrInvalidType},
		{"empty", nil, ErrInvalidType},
		// Tiny file, huge declared canvas: rejected from the header alone.
		{"decompression bomb", encodePNG(t, 13_000, 1), ErrTooLarge},
	}
	for _, tt := range tests {
		if _, err := Process(tt.raw); !errors.Is(err, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, err, tt.want)
		}
	}
}

func TestRejectsDeepPixelBombs(t *testing.T) {
	// A 16-bit PNG that is tiny on disk but would decode to ~320 MB.
	var buf bytes.Buffer
	img := image.NewNRGBA64(image.Rect(0, 0, 6324, 6324))
	if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if _, err := Process(buf.Bytes()); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("got %v, want ErrTooLarge (%d bytes on disk)", err, buf.Len())
	}
}
