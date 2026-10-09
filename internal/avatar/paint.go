package avatar

import (
	"math"
	"slices"
)

// Size is the face's width and height in pixels.
const Size = 24

const ink = "#1A1520"

// grid holds one colour per pixel; "" is empty (background).
type grid [Size * Size]string

func (g *grid) set(x, y int, c string) {
	if x >= 0 && x < Size && y >= 0 && y < Size {
		g[y*Size+x] = c
	}
}

func (g *grid) get(x, y int) string {
	if x >= 0 && x < Size && y >= 0 && y < Size {
		return g[y*Size+x]
	}
	return ""
}

// setSym sets a pixel and its mirror across the vertical centre line.
func (g *grid) setSym(x, y int, c string) {
	g.set(x, y, c)
	g.set(Size-1-x, y, c)
}

func (g *grid) row(xs []int, y int, c string) {
	for _, x := range xs {
		g.set(x, y, c)
	}
}

// Products that are then added are wrapped in float64(...): Go may fuse
// a*b+c into one instruction (FMA) on some CPUs, which rounds once instead
// of twice and can move a pixel. The conversion forces JavaScript's rounding.

// shape tests a point in continuous coordinates; pixel (x, y) is sampled
// at its centre (x+0.5, y+0.5) and the face's centre line is x = 12.
type shape func(x, y float64) bool

func (g *grid) paint(s shape, c string) {
	for y := range Size {
		for x := range Size {
			if s(float64(x)+0.5, float64(y)+0.5) {
				g[y*Size+x] = c
			}
		}
	}
}

func ell(cx, cy, rx, ry, p float64) shape {
	return func(x, y float64) bool {
		return math.Pow(math.Abs((x-cx)/rx), p)+math.Pow(math.Abs((y-cy)/ry), p) <= 1
	}
}

func circle(cx, cy, rx, ry float64) shape { return ell(cx, cy, rx, ry, 2) }

// tri is a triangle from an apex (ax, ay) down to a base at y = by of half
// width bw; tilt shifts the base sideways relative to the apex.
func tri(ax, ay, by, bw, tilt float64) shape {
	return func(x, y float64) bool {
		if y < ay || y > by {
			return false
		}
		t := (y - ay) / (by - ay)
		return math.Abs(x-(ax+float64(t*tilt))) <= float64(t*bw)+0.35
	}
}

func mirror(s shape) shape {
	return func(x, y float64) bool { return s(x, y) || s(Size-x, y) }
}

// ---------------------------------------------------------------------------
// Eyes
// ---------------------------------------------------------------------------

// eyePixel is one pixel of an eye: out counts away from the face centre,
// so the right eye is the exact mirror of the left.
type eyePixel struct {
	out, down int
	c         string
}

type eye struct {
	px   []eyePixel
	w, h int
	side int // pupil width of a side-eye; 0 otherwise
}

func eyeMask(kind string, s int, p Palette) eye {
	const white = "#FFFFFF"
	var px []eyePixel
	box := func(w, h int, c string, cut bool) {
		for y := range h {
			for x := range w {
				corner := (x == 0 || x == w-1) && (y == 0 || y == h-1)
				if !(cut && corner) {
					px = append(px, eyePixel{x, y, c})
				}
			}
		}
	}
	switch kind {
	case "dot":
		box(s, s, ink, false)
		if s >= 2 {
			px = append(px, eyePixel{s - 1, 0, white})
		}
		return eye{px: px, w: s, h: s}
	case "round":
		w := s + 1
		box(w, w, ink, w >= 4)
		if w >= 3 {
			px = append(px, eyePixel{w - 2, 1, white})
		} else {
			px = append(px, eyePixel{w - 1, 0, white})
		}
		return eye{px: px, w: w, h: w}
	case "tall":
		box(s, s+1, ink, false)
		if s >= 2 {
			px = append(px, eyePixel{s - 1, 0, white})
		}
		return eye{px: px, w: s, h: s + 1}
	case "sleepy":
		w, row := s+1, s/2
		for x := range w {
			px = append(px, eyePixel{x, row, ink})
		}
		if w >= 3 {
			px = append(px, eyePixel{w - 1, row + 1, ink})
		}
		return eye{px: px, w: w, h: row + 2}
	case "happy":
		w := max(3, s+1)
		for x := 1; x < w-1; x++ {
			px = append(px, eyePixel{x, 0, ink})
		}
		px = append(px, eyePixel{0, 1, ink}, eyePixel{w - 1, 1, ink})
		return eye{px: px, w: w, h: 2}
	case "sparkle":
		w := s + 1
		box(w, w, ink, w >= 4)
		px = append(px, eyePixel{w - 2, 1, white}, eyePixel{1, w - 2, white})
		if w >= 4 {
			px = append(px, eyePixel{w - 3, 1, white})
		}
		return eye{px: px, w: w, h: w}
	case "slit":
		w := s + 1
		box(w, w, p.Iris, w >= 4)
		for y := range w {
			px = append(px, eyePixel{1, y, ink})
		}
		return eye{px: px, w: w, h: w}
	case "big":
		w := s + 2
		box(w, w, ink, true)
		px = append(px, eyePixel{w - 2, 1, white})
		if w >= 4 {
			px = append(px, eyePixel{w - 3, 1, white}, eyePixel{1, w - 2, white})
		}
		return eye{px: px, w: w, h: w}
	case "side":
		w, h := s+2, s+1
		box(w, h, white, false)
		return eye{px: px, w: w, h: h, side: s}
	}
	return eye{w: 1, h: 1}
}

// ---------------------------------------------------------------------------
// Painting
// ---------------------------------------------------------------------------

// paintFace draws the animal on an empty grid. blink swaps open eyes for
// closed ones, for the second frame of an idle animation.
func paintFace(t Traits, blink bool) grid {
	var g grid
	sp, p := t.Species, t.Palette
	rx, ry := float64(t.HeadW)/2, float64(t.HeadH)/2
	cy := Size - 2 - ry
	headTop := cy - ry
	headBottom := int(math.Floor(cy+ry)) - 1
	head := ell(12, cy, rx, ry, t.Round)
	inHead := func(x, y int) bool { return head(float64(x)+0.5, float64(y)+0.5) }

	// Eye geometry first: frogs grow their eyes on bumps, owls ring them.
	kind := t.EyeShape
	if blink && !slices.Contains([]string{"sleepy", "happy", "side"}, kind) {
		kind = "sleepy"
	}
	e := eyeMask(kind, t.EyeSize, p)
	maxGap := int(math.Floor(rx - float64(e.w) - 1))
	gap := max(1, min(t.EyeGap, maxGap))
	innerL, innerR := 12-gap-1, 12+gap
	ew, eh := float64(e.w), float64(e.h)
	ey := jsRound(cy + float64(t.EyeY) - eh/2)
	if sp.Ears == "bumps" {
		ey = jsRound(headTop + 1.2 - eh/2)
	}
	eyeCx := float64(innerL) + 0.5 - (ew-1)/2
	eyeCy := float64(ey) + eh/2

	// 1. Ears behind the head.
	es := float64(t.EarSize)
	switch sp.Ears {
	case "pointy":
		ax, ay, by, bw := 12-float64(rx*0.6)-0.5, math.Max(0.5, headTop-es+1), headTop+3, float64(es*0.45)+1
		g.paint(mirror(tri(ax, ay, by, bw, 1)), p.Fur)
		if bw > 2 {
			g.paint(mirror(tri(ax+0.2, ay+1.8, by, bw-1.3, 1)), p.Inner)
		}
	case "round":
		ear := p.Ear
		if p.Ear == p.Shade {
			ear = p.Fur
		}
		g.paint(mirror(circle(12-float64(rx*0.7), headTop+1.2, es+0.3, es+0.3)), ear)
		if t.EarSize >= 2 {
			g.paint(mirror(circle(12-float64(rx*0.7), headTop+1.4, es-0.8, es-0.8)), p.Inner)
		}
	case "long":
		cx := 12 - 2.6
		g.paint(mirror(circle(cx, headTop-es/2+1.5, 1.9, es/2+1.6)), p.Fur)
		g.paint(mirror(circle(cx, headTop-es/2+1.8, 0.75, es/2)), p.Inner)
	case "tufts":
		g.paint(mirror(tri(12-float64(rx*0.7)-1, math.Max(0.5, headTop-es+1.5), headTop+2, float64(es*0.4)+0.7, 1.6)), p.Shade)
	case "pig":
		g.paint(mirror(tri(12-float64(rx*0.55)-1, math.Max(0.5, headTop-es+1.5), headTop+2.5, float64(es*0.5)+0.8, 1.2)), p.Fur)
		g.paint(mirror(tri(12-float64(rx*0.55)-0.8, math.Max(0.5, headTop-es+3), headTop+2.5, float64(es*0.5)-0.4, 1.2)), p.Shade)
	case "gills":
		for k := range 3 {
			bx, by, dy := jsRound(12-rx)+1, jsRound(cy-2.5+float64(float64(k)*1.8)), []float64{-0.7, -0.15, 0.45}[k]
			n := t.EarSize
			if k == 1 {
				n++
			}
			for i := 1; i <= n; i++ {
				g.setSym(bx-i, jsRound(float64(by)+float64(float64(i)*dy)), p.Inner)
			}
		}
	}

	// 2. Head, with a darker rim along the bottom.
	g.paint(head, p.Fur)
	g.paint(func(x, y float64) bool { return head(x, y) && !head(x, y+1.4) }, p.Shade)
	if sp.Ears == "bumps" {
		g.paint(mirror(circle(eyeCx, headTop+1.2, ew/2+1.4, eh/2+1.3)), p.Fur)
	}

	// 3. Floppy ears hang over the sides.
	if sp.Ears == "floppy" {
		g.paint(mirror(circle(12-rx+0.8, cy-float64(ry*0.35)+es/2-1, 2.1, es/2+0.7)), p.Ear)
	}

	// 4. Markings.
	switch t.Marks {
	case "cheeks":
		g.paint(func(x, y float64) bool {
			return head(x, y) && y > cy+0.5 && math.Abs(x-12) > float64((float64(headBottom)-y)*0.6)-1.2
		}, p.Light)
	case "stripes":
		top := int(math.Ceil(headTop))
		for _, s := range [][2]int{{11, 2}, {12, 2}, {9, 1}, {14, 1}} {
			for i := range s[1] {
				if inHead(s[0], top+1+i) {
					g.set(s[0], top+1+i, p.Shade)
				}
			}
		}
	case "eyepatch":
		g.paint(circle(eyeCx, eyeCy, ew/2+1.4, eh/2+1.4), p.Ear)
	case "patches":
		g.paint(mirror(circle(eyeCx-0.4, eyeCy+0.6, ew/2+1.2, eh/2+1.5)), p.Patch)
	case "mask":
		g.paint(func(x, y float64) bool {
			return head(x, y) && math.Abs(y-eyeCy) < eh/2+1.1 && math.Abs(x-12) > 0.6
		}, p.Patch)
	case "discs":
		g.paint(mirror(circle(eyeCx, eyeCy, ew/2+1.5, eh/2+1.5)), p.Light)
	case "puffs":
		g.paint(mirror(circle(12-rx+2.6, cy+float64(ry*0.35), 2.8, 2.4)), p.Light)
	case "spots":
		r := mulberry32(uint32(math.Floor(t.spots * 1e9)))
		for range 5 {
			x := int(math.Floor(12 - rx + 1 + float64(r()*float64(t.HeadW-2))))
			y := int(math.Floor(headTop + 1 + float64(r()*(float64(ey)-headTop-1))))
			if inHead(x, y) && math.Abs(float64(y)-eyeCy) > eh/2+0.5 {
				g.set(x, y, p.Shade)
			}
		}
	}

	// 5. Muzzle under the nose.
	ny := min(ey+e.h+1, headBottom-3)
	if t.Muzzle != nil {
		mw, mh := float64(t.Muzzle[0]), float64(t.Muzzle[1])
		g.paint(circle(12, float64(ny)+mh/2-0.3, mw/2, mh/2), p.Light)
	}

	// 6. Eyes, unless shades cover them.
	if t.Acc != "shades" {
		for _, px := range e.px {
			g.set(innerL-px.out, ey+px.down, px.c)
			g.set(innerR+px.out, ey+px.down, px.c)
		}
		// A side-eye's pupils both look to the viewer's right.
		for y := range e.h {
			for i := range e.side {
				g.set(innerL-i, ey+y, ink)
				g.set(innerR+e.w-1-i, ey+y, ink)
			}
		}
	}

	// 7. Blush under the eyes.
	if t.Blush {
		by := ey + e.h
		if t.EyeShape == "happy" || t.EyeShape == "sleepy" {
			by++
		}
		for dx := range 2 {
			if x := innerL - e.w + 1 - dx; inHead(x, by) {
				g.setSym(x, by, "#FF8FA3")
			}
		}
	}

	// 8. Nose.
	switch sp.Nose {
	case "tri":
		g.setSym(11, ny, p.Nose)
		g.setSym(11, ny+1, p.Nose)
	case "dot":
		g.setSym(11, ny, p.Nose)
		g.setSym(10, ny, p.Nose)
		g.setSym(11, ny+1, p.Nose)
	case "wide":
		g.setSym(9, ny+1, p.Nose)
		g.setSym(10, ny+1, p.Nose)
	case "beak":
		g.setSym(11, ny-1, p.Nose)
		g.setSym(11, ny, p.Nose)
	case "snout":
		g.paint(circle(12, float64(ny)+1, 3.1, 1.9), p.Nose)
		g.setSym(10, ny+1, p.Shade)
	}

	// 9. Mouth.
	my := ny + 2
	if sp.Nose == "none" {
		my = ny
	}
	my = min(my, headBottom-1)
	switch t.Mouth {
	case "w":
		g.row([]int{9, 14}, my, ink)
		g.row([]int{10, 13}, my+1, ink)
		g.row([]int{11, 12}, my, ink)
	case "smile":
		g.row([]int{10, 13}, my, ink)
		g.row([]int{11, 12}, my+1, ink)
	case "line":
		g.row([]int{10, 11, 12, 13}, my, ink)
	case "o":
		g.row([]int{11, 12}, my, ink)
		g.row([]int{11, 12}, my+1, ink)
	case "open":
		g.row([]int{10, 11, 12, 13}, my, ink)
		g.row([]int{10, 13}, my+1, ink)
		g.row([]int{11, 12}, my+1, "#D9445E")
		g.row([]int{11, 12}, my+2, ink)
	case "tongue":
		g.row([]int{10, 13}, my, ink)
		g.row([]int{11, 12}, my+1, ink)
		g.row([]int{12}, my+2, "#FF7A9A")
		g.row([]int{13}, my+1, "#FF7A9A")
	case "fang":
		g.row([]int{10, 13}, my, ink)
		g.row([]int{11, 12}, my+1, ink)
		g.row([]int{10}, my+1, "#FFFFFF")
	case "wide":
		g.row([]int{8, 15}, my, ink)
		g.row([]int{9, 10, 11, 12, 13, 14}, my+1, ink)
	}

	// 10. Accessory.
	paintAccessory(&g, t, accessoryFrame{
		top: int(math.Ceil(headTop)), cy: cy, rx: rx, ry: ry, headBottom: headBottom,
		innerL: innerL, innerR: innerR, ey: ey, eyeW: e.w, eyeH: e.h,
	})

	// 11. Outline: every empty pixel touching the animal turns ink.
	filled := g
	for y := range Size {
		for x := range Size {
			if filled[y*Size+x] != "" {
				continue
			}
			if filled.get(x-1, y) != "" || filled.get(x+1, y) != "" || filled.get(x, y-1) != "" || filled.get(x, y+1) != "" {
				g.set(x, y, ink)
			}
		}
	}

	// 12. Legendary sparkles in empty corners.
	if t.Tier == "legendary" {
		star := [][2]int{{0, 0}, {0, -1}, {0, 1}, {-1, 0}, {1, 0}}
		for _, at := range [][2]int{{2, 3}, {21, 2}, {2, 20}, {21, 20}} {
			free := true
			for _, d := range star {
				free = free && g.get(at[0]+d[0], at[1]+d[1]) == ""
			}
			if !free {
				continue
			}
			for i, d := range star {
				c := "#FFF6C9"
				if i == 0 {
					c = "#FFFFFF"
				}
				g.set(at[0]+d[0], at[1]+d[1], c)
			}
		}
	}
	return g
}

type accessoryFrame struct {
	top, headBottom    int
	cy, rx, ry         float64
	innerL, innerR, ey int
	eyeW, eyeH         int
}

func paintAccessory(g *grid, t Traits, f accessoryFrame) {
	const dark, gold = "#2B2B36", "#F6C445"
	c, top := t.AccColor, f.top
	firstRow := func() int {
		for y := range Size {
			for x := range Size {
				if g.get(x, y) != "" {
					return y
				}
			}
		}
		return Size
	}
	acc := t.Acc
	if acc == "halo" && firstRow() < 3 {
		acc = "crown" // no room above the ears
	}
	switch acc {
	case "crown":
		g.row([]int{8, 11, 12, 15}, top-3, gold)
		g.row([]int{8, 9, 11, 12, 14, 15}, top-2, gold)
		g.row([]int{8, 9, 10, 11, 12, 13, 14, 15}, top-1, gold)
		g.row([]int{8, 9, 10, 11, 12, 13, 14, 15}, top, "#C99A1E")
		g.row([]int{11, 12}, top-1, "#E5484D")
	case "halo":
		y := firstRow() - 3
		g.row([]int{9, 10, 11, 12, 13, 14}, y, gold)
		g.row([]int{8, 15}, y+1, gold)
		g.row([]int{9, 10, 11, 12, 13, 14}, y+2, gold)
	case "cap":
		dome := circle(12, float64(top)+3, f.rx*0.8, 4)
		g.paint(func(x, y float64) bool { return dome(x, y) && y < float64(top)+2.5 }, c)
		for x := 12; float64(x) < math.Min(Size-1, 12+float64(f.rx*0.8)+3); x++ {
			g.set(x, top+2, dark)
		}
		g.set(11, top-1, dark)
		g.set(12, top-1, dark)
	case "beanie":
		hat := circle(12, f.cy, f.rx+0.6, f.ry+0.6)
		g.paint(func(x, y float64) bool { return hat(x, y) && y < float64(top)+3 }, c)
		g.paint(func(x, y float64) bool { return hat(x, y) && y > float64(top)+2 && y < float64(top)+3.5 }, dark)
		g.row([]int{11, 12}, top-2, "#FFFFFF")
		g.row([]int{11, 12}, top-1, "#FFFFFF")
	case "headphones":
		outer, inner := circle(12, f.cy, f.rx+1.6, f.ry+1.6), circle(12, f.cy, f.rx+0.6, f.ry+0.6)
		g.paint(func(x, y float64) bool { return y < f.cy-1 && outer(x, y) && !inner(x, y) }, dark)
		g.paint(mirror(func(x, y float64) bool {
			return x > 12-f.rx-1.6 && x < 12-f.rx+1.2 && y > f.cy-2 && y < f.cy+2.5
		}), c)
	case "flower":
		fx, fy := jsRound(12-float64(f.rx*0.65))-1, top+1
		for _, d := range [][2]int{{0, -1}, {-1, 0}, {1, 0}, {0, 1}} {
			g.set(fx+d[0], fy+d[1], c)
		}
		g.set(fx, fy, gold)
	case "bow":
		bx, by := jsRound(12+float64(f.rx*0.55)), top+1
		for _, d := range [][2]int{{-2, -1}, {-2, 0}, {-2, 1}, {-1, 0}, {0, 0}, {1, 0}, {2, -1}, {2, 0}, {2, 1}} {
			g.set(bx+d[0], by+d[1], c)
		}
	case "bandaid":
		bx, by := f.innerR+1, min(f.ey+f.eyeH+1, f.headBottom-2)
		g.row([]int{bx, bx + 1, bx + 2}, by, "#E9C39B")
		g.row([]int{bx, bx + 1, bx + 2}, by+1, "#E9C39B")
		g.set(bx+1, by, "#C99A72")
	case "shades":
		const lens = "#141218"
		y0 := f.ey + max(0, f.eyeH/2-1)
		for x := f.innerL - f.eyeW; x <= f.innerR+f.eyeW; x++ {
			g.set(x, y0, lens)
			g.set(x, y0+1, lens)
		}
		for x := f.innerL - f.eyeW + 1; x <= f.innerL; x++ {
			g.set(x, y0+2, lens)
		}
		for x := f.innerR; x <= f.innerR+f.eyeW-1; x++ {
			g.set(x, y0+2, lens)
		}
		g.set(f.innerL-f.eyeW+1, y0, "#FFFFFF")
		g.set(f.innerR, y0, "#FFFFFF")
	case "orange":
		fruit := circle(12, float64(top)-1.2, 2.7, 2.2)
		g.paint(fruit, "#FF9F1C")
		lit := circle(12, float64(top)-1.6, 2.7, 2.2)
		g.paint(func(x, y float64) bool { return fruit(x, y) && !lit(x, y) }, "#E07B00")
		g.set(12, top-4, "#4CAF50")
		g.set(13, top-4, "#4CAF50")
	case "leaf":
		g.set(9, top, "#3E9B4F")
		g.set(10, top-1, "#3E9B4F")
		g.set(9, top-1, "#5CC46C")
	}
}

// background returns the backdrop colour of pixel (x, y): flat for common
// faces, split on the diagonal for rarer ones.
func (t Traits) background(x, y int) string {
	b1, b2 := t.Bg.Color, t.Bg2.Color
	if t.Tier == "legendary" {
		b1, b2 = "#E6DBFF", "#C9F2EC"
	}
	if t.Tier != "common" && x+y >= Size {
		return b2
	}
	return b1
}
