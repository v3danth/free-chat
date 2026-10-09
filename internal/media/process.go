package media

import (
	"bytes"
	"crypto/sha256"
	"image"
	"image/color"
	_ "image/gif" // register decoders
	"image/jpeg"
	_ "image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	fullMax   = 1280
	thumbMax  = 320
	blurMax   = 16 // downscale this far, then upscale: nothing recognisable survives
	maxPixels = 40_000_000
	maxSide   = 12_000
)

// Processed is an upload re-encoded into every served variant. Re-encoding
// is the privacy step: EXIF (including GPS) and any payload smuggled after
// the image data are dropped because only decoded pixels are kept.
type Processed struct {
	SHA256        [32]byte
	Width, Height int
	Full, Thumb   []byte
	Blur          []byte
}

// Process is pure: same bytes in, same bytes out.
func Process(raw []byte) (Processed, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, ErrInvalidType
	}
	switch format {
	case "jpeg", "png", "gif", "webp":
	default:
		return Processed{}, ErrInvalidType
	}
	// Check the header before decoding: a tiny file can declare a huge
	// canvas and exhaust memory (a decompression bomb).
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxSide || cfg.Height > maxSide ||
		cfg.Width*cfg.Height > maxPixels {
		return Processed{}, ErrTooLarge
	}

	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, ErrInvalidType
	}

	full := fit(src, fullMax, draw.CatmullRom)
	thumb := fit(src, thumbMax, draw.ApproxBiLinear)
	tiny := fit(thumb, blurMax, draw.ApproxBiLinear)
	blur := image.NewRGBA(thumb.Bounds())
	draw.BiLinear.Scale(blur, blur.Bounds(), tiny, tiny.Bounds(), draw.Src, nil)

	p := Processed{SHA256: sha256.Sum256(raw), Width: full.Bounds().Dx(), Height: full.Bounds().Dy()}
	for _, v := range []struct {
		img *image.RGBA
		q   int
		out *[]byte
	}{{full, 82, &p.Full}, {thumb, 75, &p.Thumb}, {blur, 60, &p.Blur}} {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, v.img, &jpeg.Options{Quality: v.q}); err != nil {
			return Processed{}, err
		}
		*v.out = buf.Bytes()
	}
	return p, nil
}

// fit scales src so its longest side is at most limit (never upscaling),
// onto white so transparent PNG/GIF areas do not turn black in JPEG.
func fit(src image.Image, limit int, s draw.Scaler) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > limit || h > limit {
		if w >= h {
			w, h = limit, max(1, h*limit/b.Dx())
		} else {
			w, h = max(1, w*limit/b.Dy()), limit
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	s.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}
