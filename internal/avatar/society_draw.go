package avatar

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"math"
	"strings"
)

// A scene is a list of shapes in a 100 x 100 box, painted in order. Both
// renderers draw the same list, so the two styles are always one mark.
// As in paint.go, products that are then added are wrapped in float64(...)
// so the CPU cannot fuse them and move an edge.

type shapeItem struct {
	poly   []point
	seg    [2]point
	segW   float64
	light  float64 // > 0: shade from the metal ramp by facet lighting
	fill   string
	glow   bool
	eye    bool  // never let the pixel renderer lose it
	at     point // centre, for eyes and stamps
	stamp  []string
	vector bool // low-poly only
	pixel  bool // pixel only
}

type scene struct {
	items   []shapeItem
	ramp    [4]string
	bg      string
	shimmer bool
}

func lerpPt(a, b point, t float64) point {
	return point{a[0] + float64((b[0]-a[0])*t), a[1] + float64((b[1]-a[1])*t)}
}

func mirrorPt(p point) point { return point{100 - p[0], p[1]} }

func regularPoly(cx, cy, r float64, n int, rot float64) []point {
	pts := make([]point, n)
	for i := range n {
		a := rot + float64(i)*2*math.Pi/float64(n)
		pts[i] = point{cx + float64(r*math.Cos(a)), cy + float64(r*math.Sin(a))}
	}
	return pts
}

func rotateAround(pts []point, c point, a float64) []point {
	cs, sn := math.Cos(a), math.Sin(a)
	out := make([]point, len(pts))
	for i, p := range pts {
		dx, dy := p[0]-c[0], p[1]-c[1]
		out[i] = point{c[0] + float64(dx*cs) - float64(dy*sn), c[1] + float64(dx*sn) + float64(dy*cs)}
	}
	return out
}

var lightDir = func() [3]float64 {
	v := [3]float64{-0.5, -0.65, 0.7}
	l := math.Hypot(math.Hypot(v[0], v[1]), v[2])
	return [3]float64{v[0] / l, v[1] / l, v[2] / l}
}()

// brightness lights a facet by its slope: 0.12 (away) to 1 (facing).
func brightness(a, b, c [3]float64) float64 {
	u := [3]float64{b[0] - a[0], b[1] - a[1], b[2] - a[2]}
	v := [3]float64{c[0] - a[0], c[1] - a[1], c[2] - a[2]}
	n := [3]float64{
		float64(u[1]*v[2]) - float64(u[2]*v[1]),
		float64(u[2]*v[0]) - float64(u[0]*v[2]),
		float64(u[0]*v[1]) - float64(u[1]*v[0]),
	}
	if n[2] < 0 {
		n = [3]float64{-n[0], -n[1], -n[2]}
	}
	l := math.Hypot(math.Hypot(n[0], n[1]), n[2])
	if l == 0 {
		l = 1
	}
	dot := (float64(n[0]*lightDir[0]) + float64(n[1]*lightDir[1]) + float64(n[2]*lightDir[2])) / l
	return 0.12 + float64(0.88*math.Max(0, math.Min(1, dot)))
}

func buildScene(m Mark) scene {
	a := m.Animal
	ramp := m.Metal.ramp
	sc := scene{ramp: ramp, bg: m.Bg, shimmer: m.Rank == "grandmaster"}
	add := func(it shapeItem) { sc.items = append(sc.items, it) }
	f := m.Frame

	// Frame: rings painted outside-in.
	ring := func(r float64, fill string) { add(shapeItem{poly: regularPoly(50, 50, r, f.sides, f.rot), fill: fill}) }
	if m.Rank == "grandmaster" {
		for i := range 16 {
			ang := float64(i)*math.Pi/8 + f.rot
			w := 3.0
			if i%2 == 1 {
				w = 1.6
			}
			add(shapeItem{seg: [2]point{{50 + float64(46*math.Cos(ang)), 50 + float64(46*math.Sin(ang))}, {50 + float64(50*math.Cos(ang)), 50 + float64(50*math.Sin(ang))}}, segW: w, fill: ramp[3]})
		}
	}
	ring(47, ramp[2])
	ring(43.5, m.Bg)
	if m.Rank != "initiate" {
		ring(41.5, ramp[1])
		ring(39, m.Bg)
	}
	if m.Rank == "keeper" || m.Rank == "grandmaster" {
		for i := range 4 {
			ang := float64(i)*math.Pi/2 - math.Pi/2
			c := point{50 + float64(45.2*math.Cos(ang)), 50 + float64(45.2*math.Sin(ang))}
			add(shapeItem{poly: rotateAround(regularPoly(c[0], c[1], 3.2, 4, 0), c, math.Pi/4), fill: ramp[3]})
		}
	}

	// Face points with this name's proportions.
	p := map[string]point{}
	for _, k := range meshOrder {
		v := a.pts[k]
		p[k] = point{50 + float64((v[0]-50)*m.SX), v[1]}
	}
	p["ET"] = lerpPt(p["EI"], p["ET"], m.Ears)
	c1y := p["C1"][1]
	for _, k := range []string{"SN", "JW", "C2", "C3"} {
		p[k] = point{p[k][0], c1y + float64((p[k][1]-c1y)*m.Snout)}
	}
	antlers := make([][2]point, len(a.antlers))
	for i, seg := range a.antlers {
		for j, q := range seg {
			antlers[i][j] = point{50 + float64((q[0]-50)*m.SX), 30 + float64((q[1]-30)*m.Ears)}
		}
	}

	// Fit the whole head inside the frame.
	x0, x1, y0, y1 := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	grow := func(q point) {
		x0, x1, y0, y1 = math.Min(x0, q[0]), math.Max(x1, q[0]), math.Min(y0, q[1]), math.Max(y1, q[1])
	}
	for _, k := range meshOrder {
		grow(p[k])
		grow(mirrorPt(p[k]))
	}
	for _, seg := range antlers {
		for _, q := range seg {
			grow(q)
			grow(mirrorPt(q))
		}
	}
	s := math.Min(f.fit/(x1-x0), f.fit/(y1-y0))
	place := func(q point) point {
		return point{50 + float64((q[0]-(x0+x1)/2)*s), 51 + float64((q[1]-(y0+y1)/2)*s)}
	}

	for _, seg := range antlers {
		add(shapeItem{seg: [2]point{place(seg[0]), place(seg[1])}, segW: 3.4 * s, fill: ramp[1]})
		add(shapeItem{seg: [2]point{place(mirrorPt(seg[0])), place(mirrorPt(seg[1]))}, segW: 3.4 * s, fill: ramp[1]})
	}

	// Facets, lit by their own slope.
	for _, fc := range maskFacets {
		var left, right [3][3]float64
		for i, k := range fc {
			z := maskDepth[k] * 30
			left[i] = [3]float64{p[k][0], p[k][1], z}
			mp := mirrorPt(p[k])
			right[i] = [3]float64{mp[0], mp[1], z}
		}
		for _, tri := range [][3][3]float64{left, right} {
			poly := []point{place(point{tri[0][0], tri[0][1]}), place(point{tri[1][0], tri[1][1]}), place(point{tri[2][0], tri[2][1]})}
			add(shapeItem{poly: poly, light: brightness(tri[0], tri[1], tri[2])})
		}
	}
	if !a.noInner {
		base := lerpPt(p["EO"], p["EI"], 0.5)
		inner := []point{lerpPt(p["ET"], base, 0.3), lerpPt(p["EO"], p["EI"], 0.28), lerpPt(p["EO"], p["EI"], 0.72)}
		var l, r []point
		for _, q := range inner {
			l = append(l, place(q))
			r = append(r, place(mirrorPt(q)))
		}
		add(shapeItem{poly: l, fill: ramp[0]})
		add(shapeItem{poly: r, fill: ramp[0]})
	}

	// Eyes.
	glow := m.Glow[1]
	eyeColor := map[string]string{"hollow": "#07060A", "slit": glow, "glow": glow, "void": m.Bg, "closed": ramp[0]}[m.Eyes]
	ec := point{p["EY"][0] + 4.5, p["EY"][1] + 1.5}
	const ea = 6.2
	eh := map[string]float64{"hollow": 3, "slit": 1.3, "glow": 2.8, "void": 3.4}[m.Eyes]
	lit := m.Eyes == "glow" || m.Eyes == "slit"
	for _, side := range []float64{1, -1} {
		c := ec
		if side == -1 {
			c = mirrorPt(ec)
		}
		if m.Eyes == "closed" {
			pts := []point{place(point{c[0] - ea, c[1] - 0.6}), place(point{c[0], c[1] + 1.4}), place(point{c[0] + ea, c[1] - 0.6})}
			add(shapeItem{seg: [2]point{pts[0], pts[1]}, segW: 1.5 * s, fill: eyeColor})
			add(shapeItem{seg: [2]point{pts[1], pts[2]}, segW: 1.5 * s, fill: eyeColor})
			continue
		}
		almond := []point{
			{c[0] - ea, c[1]}, {c[0] - float64(ea*0.3), c[1] - eh}, {c[0] + float64(ea*0.5), c[1] - float64(eh*0.8)},
			{c[0] + ea, c[1]}, {c[0] + float64(ea*0.3), c[1] + float64(eh*0.7)}, {c[0] - float64(ea*0.4), c[1] + float64(eh*0.6)},
		}
		if side == -1 {
			for i, q := range almond {
				almond[i] = point{2*c[0] - q[0], q[1]}
			}
		}
		almond = rotateAround(almond, c, a.tilt*side)
		if lit { // a dark socket so the light reads on any metal
			socket := make([]point, len(almond))
			for i, q := range almond {
				socket[i] = place(point{c[0] + float64((q[0]-c[0])*1.25), c[1] + float64((q[1]-c[1])*1.6)})
			}
			add(shapeItem{poly: socket, fill: "#07060A", vector: true})
		}
		placed := make([]point, len(almond))
		for i, q := range almond {
			placed[i] = place(q)
		}
		add(shapeItem{poly: placed, fill: eyeColor, glow: lit, eye: true, at: place(c)})
	}

	// The mark on the brow.
	F := place(lerpPt(p["C0"], p["C1"], 0.32))
	k := 4.2 * s
	stamp := func(rows ...string) { add(shapeItem{stamp: rows, at: F, fill: glow, pixel: true}) }
	switch m.Sigil {
	case "triangle":
		tr := regularPoly(F[0], F[1]+float64(k*0.15), k, 3, -math.Pi/2)
		for i := range 3 {
			add(shapeItem{seg: [2]point{tr[i], tr[(i+1)%3]}, segW: 0.9 * s, fill: glow, vector: true})
		}
		stamp(".#.", "#.#", "###")
	case "crescent":
		var pts []point
		for i := 0; i <= 12; i++ {
			th := (-20 + float64(i)*220/12) * math.Pi / 180
			pts = append(pts, point{F[0] + float64(k*math.Cos(th)), F[1] + float64(k*math.Sin(th))})
		}
		for i := 12; i >= 0; i-- {
			th := (-5 + float64(i)*190/12) * math.Pi / 180
			pts = append(pts, point{F[0] + float64(k*0.8*math.Cos(th)), F[1] - float64(k*0.35) + float64(k*0.8*math.Sin(th))})
		}
		add(shapeItem{poly: pts, fill: glow, vector: true})
		stamp("#.#", "#.#", ".#.")
	case "star":
		var pts []point
		for i := range 8 {
			r := k
			if i%2 == 1 {
				r = k * 0.32
			}
			th := float64(i)*math.Pi/4 - math.Pi/2
			pts = append(pts, point{F[0] + float64(r*math.Cos(th)), F[1] + float64(r*math.Sin(th))})
		}
		add(shapeItem{poly: pts, fill: glow, vector: true})
		stamp(".#.", "###", ".#.")
	case "eye", "third eye":
		add(shapeItem{poly: []point{{F[0], F[1] - k}, {F[0] + float64(k*0.55), F[1]}, {F[0], F[1] + k}, {F[0] - float64(k*0.55), F[1]}}, fill: glow, glow: true, vector: true})
		add(shapeItem{poly: regularPoly(F[0], F[1], k*0.28, 12, 0), fill: "#07060A", vector: true})
		stamp(".#.", "###", ".#.")
	case "ring":
		add(shapeItem{poly: regularPoly(F[0], F[1], k*0.85, 24, 0), fill: glow, vector: true})
		add(shapeItem{poly: regularPoly(F[0], F[1], k*0.5, 24, 0), fill: ramp[1], vector: true})
		stamp("###", "#.#", "###")
	case "cross":
		add(shapeItem{seg: [2]point{{F[0] - k, F[1]}, {F[0] + k, F[1]}}, segW: 1.1 * s, fill: glow, vector: true})
		add(shapeItem{seg: [2]point{{F[0], F[1] - k}, {F[0], F[1] + k}}, segW: 1.1 * s, fill: glow, vector: true})
		stamp("#.#", ".#.", "#.#")
	}
	return sc
}

// ---------------------------------------------------------------------------
// Low-poly: SVG.
// ---------------------------------------------------------------------------

// rampColor blends along the metal's four tones by brightness.
func rampColor(ramp [4]string, b float64) string {
	x := math.Max(0, math.Min(2.999, b*3))
	i := int(math.Floor(x))
	f := x - float64(i)
	c0, c1 := parseHex(ramp[i]), parseHex(ramp[i+1])
	mix := func(a, b uint8) int { return jsRound(float64(a) + float64((float64(b)-float64(a))*f)) }
	return fmt.Sprintf("rgb(%d,%d,%d)", mix(c0.R, c1.R), mix(c0.G, c1.G), mix(c0.B, c1.B))
}

// MarkSVG draws the low-poly style. The SVG holds only numbers and colours
// this package produced, never user text. animate adds the Grandmaster
// sheen; it is off for avatars shown in lists, and always hidden for
// people who ask their system for reduced motion.
func MarkSVG(m Mark, animate bool) []byte {
	sc := buildScene(m)
	var b strings.Builder
	b.WriteString(`<svg viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg">`)
	b.WriteString(`<defs><filter id="f" x="-50%" y="-50%" width="200%" height="200%"><feGaussianBlur stdDeviation="0.9" result="b"/><feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge></filter>`)
	b.WriteString(`<linearGradient id="s" x1="0" x2="1"><stop offset="0" stop-color="#fff" stop-opacity="0"/><stop offset="0.5" stop-color="#FFF3C8" stop-opacity="0.45"/><stop offset="1" stop-color="#fff" stop-opacity="0"/></linearGradient></defs>`)
	fmt.Fprintf(&b, `<rect width="100" height="100" fill="%s"/>`, sc.bg)
	for _, it := range sc.items {
		if it.pixel {
			continue
		}
		fill := it.fill
		if it.light > 0 {
			fill = rampColor(sc.ramp, it.light)
		}
		glow := ""
		if it.glow {
			glow = ` filter="url(#f)"`
		}
		if it.poly != nil {
			b.WriteString(`<polygon points="`)
			for i, q := range it.poly {
				if i > 0 {
					b.WriteByte(' ')
				}
				fmt.Fprintf(&b, "%.2f,%.2f", q[0], q[1])
			}
			fmt.Fprintf(&b, `" fill="%s" stroke="%s" stroke-width="0.35" stroke-linejoin="round"%s/>`, fill, fill, glow)
		} else {
			fmt.Fprintf(&b, `<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="%s" stroke-width="%.2f" stroke-linecap="round"/>`,
				it.seg[0][0], it.seg[0][1], it.seg[1][0], it.seg[1][1], fill, it.segW)
		}
	}
	if sc.shimmer && animate {
		b.WriteString(`<style>@media (prefers-reduced-motion: reduce) { .sheen { display: none; } }</style>`)
		b.WriteString(`<rect class="sheen" x="-100" y="0" width="60" height="100" fill="url(#s)" style="mix-blend-mode:screen"><animate attributeName="x" from="-100" to="160" dur="3.2s" repeatCount="indefinite"/></rect>`)
	}
	b.WriteString(`</svg>`)
	return []byte(b.String())
}

// ---------------------------------------------------------------------------
// Pixel: 32 x 32, the metal quantized to four tones.
// ---------------------------------------------------------------------------

const markPixels = 32

func inPoly(x, y float64, poly []point) bool {
	inside := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		xi, yi, xj, yj := poly[i][0], poly[i][1], poly[j][0], poly[j][1]
		if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
			inside = !inside
		}
	}
	return inside
}

func segDist(px, py float64, seg [2]point) float64 {
	ax, ay, bx, by := seg[0][0], seg[0][1], seg[1][0], seg[1][1]
	dx, dy := bx-ax, by-ay
	den := float64(dx*dx) + float64(dy*dy)
	if den == 0 {
		den = 1
	}
	t := math.Max(0, math.Min(1, (float64((px-ax)*dx)+float64((py-ay)*dy))/den))
	return math.Hypot(px-(ax+float64(t*dx)), py-(ay+float64(t*dy)))
}

// markGrid paints the pixel style as 32 x 32 "#RRGGBB" colours.
func markGrid(m Mark) [markPixels * markPixels]string {
	sc := buildScene(m)
	const unit = 100.0 / markPixels
	var g [markPixels * markPixels]string
	for i := range g {
		g[i] = sc.bg
	}
	set := func(x, y int, c string) {
		if x >= 0 && x < markPixels && y >= 0 && y < markPixels {
			g[y*markPixels+x] = c
		}
	}
	for _, it := range sc.items {
		if it.vector {
			continue
		}
		fill := it.fill
		if it.light > 0 {
			fill = sc.ramp[min(3, int(math.Floor(it.light*4)))]
		}
		if it.stamp != nil {
			cx, cy := int(math.Floor(it.at[0]/unit)), int(math.Floor(it.at[1]/unit))
			for dy, row := range it.stamp {
				for dx, ch := range row {
					if ch == '#' {
						set(cx-1+dx, cy-1+dy, fill)
					}
				}
			}
			continue
		}
		hit := false
		for y := range markPixels {
			for x := range markPixels {
				px, py := (float64(x)+0.5)*unit, (float64(y)+0.5)*unit
				var on bool
				if it.poly != nil {
					on = inPoly(px, py, it.poly)
				} else {
					on = segDist(px, py, it.seg) <= math.Max(it.segW/2, unit*0.5)
				}
				if on {
					g[y*markPixels+x] = fill
					hit = true
				}
			}
		}
		if it.eye && !hit {
			set(int(math.Floor(it.at[0]/unit)), int(math.Floor(it.at[1]/unit)), fill)
		}
	}
	if sc.shimmer {
		for _, q := range [][2]int{{3, 3}, {28, 4}, {4, 28}, {27, 27}} {
			set(q[0], q[1], "#FFF6C9")
		}
	}
	return g
}

// MarkPNG draws the pixel style, each pixel scale x scale.
func MarkPNG(m Mark, scale int) ([]byte, error) {
	g := markGrid(m)
	img := image.NewNRGBA(image.Rect(0, 0, markPixels*scale, markPixels*scale))
	for y := range markPixels {
		for x := range markPixels {
			c := parseHex(g[y*markPixels+x])
			for dy := range scale {
				for dx := range scale {
					img.SetNRGBA(x*scale+dx, y*scale+dy, c)
				}
			}
		}
	}
	var buf bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
