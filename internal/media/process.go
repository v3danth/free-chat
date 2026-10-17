package media

import (
	"bytes"
	"crypto/sha256"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	_ "image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	fullMax   = 1280
	thumbMax  = 320
	maxPixels = 40_000_000
	maxSide   = 12_000
	// maxDecoded caps the memory one decode may take. Pixels alone are not
	// enough: a 40 MP 16-bit PNG needs 320 MB but compresses to a few hundred KB.
	maxDecoded = 160 << 20

	// Animated GIFs are kept as GIFs, never resized (that would mean
	// re-quantizing every frame). Past these limits one becomes a still.
	animMaxSide   = 1280
	animMaxFrames = 600
	animMaxPixels = 64_000_000 // summed over frames; one byte each when decoded
	animMaxBytes  = 12 << 20
)

// Processed is an upload re-encoded into every served variant. Re-encoding
// is the privacy step: EXIF (including GPS) and any payload smuggled after
// the image data are dropped because only decoded pixels are kept.
type Processed struct {
	SHA256        [32]byte
	Width, Height int
	Full, Thumb   []byte
	// Animated: Full and Thumb are the same GIF, stored under an animated key.
	Animated bool
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
		cfg.Width*cfg.Height > maxPixels || cfg.Width*cfg.Height*bytesPerPixel(cfg.ColorModel) > maxDecoded {
		return Processed{}, ErrTooLarge
	}

	if format == "gif" && cfg.Width <= animMaxSide && cfg.Height <= animMaxSide {
		if p, ok := processAnimated(raw); ok {
			return p, nil
		}
	}

	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, ErrInvalidType
	}

	full := fit(src, fullMax, draw.CatmullRom)
	thumb := fit(src, thumbMax, draw.ApproxBiLinear)

	p := Processed{SHA256: sha256.Sum256(raw), Width: full.Bounds().Dx(), Height: full.Bounds().Dy()}
	for _, v := range []struct {
		img *image.RGBA
		q   int
		out *[]byte
	}{{full, 82, &p.Full}, {thumb, 75, &p.Thumb}} {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, v.img, &jpeg.Options{Quality: v.q}); err != nil {
			return Processed{}, err
		}
		*v.out = buf.Bytes()
	}
	return p, nil
}

// processAnimated re-encodes a GIF of two or more frames. Re-encoding keeps
// only frames, palettes, delays and disposal: comments, application blocks
// and anything appended after the trailer are dropped. ok is false for a
// single frame or a GIF over the limits, which the caller turns into a still.
func processAnimated(raw []byte) (Processed, bool) {
	frames, pixels, ok := gifFrames(raw)
	if !ok || frames < 2 || frames > animMaxFrames || pixels > animMaxPixels {
		return Processed{}, false
	}
	g, err := gif.DecodeAll(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, false
	}
	var buf bytes.Buffer
	buf.Grow(len(raw))
	if err := gif.EncodeAll(&buf, g); err != nil || buf.Len() > animMaxBytes {
		return Processed{}, false
	}
	out := buf.Bytes()
	return Processed{
		SHA256: sha256.Sum256(raw), Width: g.Config.Width, Height: g.Config.Height,
		Full: out, Thumb: out, Animated: true,
	}, true
}

// gifFrames walks a GIF's blocks without decompressing anything, counting
// frames and their summed area, so a decompression bomb is refused before
// gif.DecodeAll allocates a byte for it. ok is false for a malformed file.
func gifFrames(raw []byte) (frames, pixels int, ok bool) {
	if len(raw) < 13 {
		return 0, 0, false
	}
	pos := 13
	if raw[10]&0x80 != 0 {
		pos += 3 << (raw[10]&7 + 1)
	}
	u16 := func(at int) int { return int(raw[at]) | int(raw[at+1])<<8 }
	// skipSubBlocks steps over a chain of length-prefixed blocks.
	skipSubBlocks := func() bool {
		for pos < len(raw) {
			n := int(raw[pos])
			pos += 1 + n
			if n == 0 {
				return true
			}
		}
		return false
	}
	for pos < len(raw) {
		switch raw[pos] {
		case 0x3B: // trailer
			return frames, pixels, true
		case 0x21: // extension: label, then sub-blocks
			pos += 2
			if !skipSubBlocks() {
				return 0, 0, false
			}
		case 0x2C: // image descriptor
			if pos+10 > len(raw) {
				return 0, 0, false
			}
			frames++
			pixels += u16(pos+5) * u16(pos+7)
			packed := raw[pos+9]
			pos += 10
			if packed&0x80 != 0 {
				pos += 3 << (packed&7 + 1)
			}
			pos++ // LZW minimum code size
			if !skipSubBlocks() {
				return 0, 0, false
			}
		default:
			return 0, 0, false
		}
	}
	// Many GIFs in the wild lack the trailer; the frames read so far stand.
	return frames, pixels, frames > 0
}

// bytesPerPixel is what the decoder will allocate per pixel for a colour
// model, rounded up (JPEG's YCbCr is counted as full 4:4:4).
func bytesPerPixel(m color.Model) int {
	switch m {
	case color.RGBA64Model, color.NRGBA64Model:
		return 8
	case color.GrayModel, color.AlphaModel:
		return 1
	case color.Gray16Model, color.Alpha16Model:
		return 2
	case color.YCbCrModel:
		return 3
	}
	if _, ok := m.(color.Palette); ok {
		return 1
	}
	return 4
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
