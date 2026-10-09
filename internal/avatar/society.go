package avatar

import (
	"fmt"
	"math"
	"slices"
)

// V2: society marks. Every name is sworn into a secret society: an animal
// mask, a metal, a mark on the brow, a rank and a member number. One scene
// (society_scene.go) is drawn two ways: low-poly SVG or 32 x 32 pixel PNG.

const VersionV2 = "v2"

type point [2]float64

// The shared mask mesh: ear tip, ear outer and inner base, cheek, eye,
// snout side and jaw on the left; forehead, bridge, nose and chin on the
// centre line. Only coordinates differ, so every animal shares one style.
type maskAnimal struct {
	Key, Label string
	tilt       float64 // eye tilt in radians, left eye
	pts        map[string]point
	antlers    [][2]point
	noInner    bool // no inner-ear facet (birds)
}

var maskAnimals = []maskAnimal{
	{Key: "fox", Label: "Fox", tilt: -0.3, pts: map[string]point{"ET": {20, 4}, "EO": {12, 38}, "EI": {36, 26}, "CK": {4, 56}, "EY": {30, 46}, "SN": {42, 70}, "JW": {24, 68}, "C0": {50, 28}, "C1": {50, 52}, "C2": {50, 78}, "C3": {50, 84}}},
	{Key: "wolf", Label: "Wolf", tilt: -0.18, pts: map[string]point{"ET": {24, 2}, "EO": {14, 34}, "EI": {36, 26}, "CK": {6, 58}, "EY": {32, 44}, "SN": {40, 74}, "JW": {20, 74}, "C0": {50, 26}, "C1": {50, 50}, "C2": {50, 82}, "C3": {50, 90}}},
	{Key: "cat", Label: "Cat", tilt: -0.25, pts: map[string]point{"ET": {20, 10}, "EO": {12, 40}, "EI": {34, 30}, "CK": {8, 62}, "EY": {30, 50}, "SN": {42, 66}, "JW": {24, 74}, "C0": {50, 32}, "C1": {50, 56}, "C2": {50, 67}, "C3": {50, 78}}},
	{Key: "bear", Label: "Bear", tilt: 0, pts: map[string]point{"ET": {18, 12}, "EO": {8, 28}, "EI": {30, 22}, "CK": {6, 56}, "EY": {32, 46}, "SN": {38, 66}, "JW": {16, 76}, "C0": {50, 26}, "C1": {50, 52}, "C2": {50, 68}, "C3": {50, 84}}},
	{Key: "owl", Label: "Owl", tilt: 0.1, pts: map[string]point{"ET": {22, 8}, "EO": {10, 34}, "EI": {36, 30}, "CK": {6, 60}, "EY": {28, 46}, "SN": {44, 62}, "JW": {22, 80}, "C0": {50, 34}, "C1": {50, 52}, "C2": {50, 74}, "C3": {50, 88}}},
	{Key: "stag", Label: "Stag", tilt: 0, pts: map[string]point{"ET": {0, 34}, "EO": {24, 28}, "EI": {26, 40}, "CK": {16, 58}, "EY": {32, 44}, "SN": {42, 84}, "JW": {30, 78}, "C0": {50, 30}, "C1": {50, 54}, "C2": {50, 92}, "C3": {50, 96}},
		antlers: [][2]point{{{38, 30}, {32, 14}}, {{32, 14}, {24, 0}}, {{33, 17}, {44, 6}}, {{28, 7}, {16, 4}}}},
	{Key: "hare", Label: "Hare", tilt: 0, pts: map[string]point{"ET": {30, 0}, "EO": {24, 36}, "EI": {40, 32}, "CK": {12, 62}, "EY": {28, 50}, "SN": {42, 70}, "JW": {26, 76}, "C0": {50, 34}, "C1": {50, 56}, "C2": {50, 69}, "C3": {50, 80}}},
	{Key: "raven", Label: "Raven", tilt: -0.1, pts: map[string]point{"ET": {30, 22}, "EO": {22, 32}, "EI": {36, 30}, "CK": {10, 56}, "EY": {32, 44}, "SN": {46, 66}, "JW": {28, 72}, "C0": {50, 30}, "C1": {50, 50}, "C2": {50, 94}, "C3": {50, 74}}, noInner: true},
}

// meshOrder is the prototype's key order; the bounding box does not depend
// on it, but keeping it makes the port easy to compare.
var meshOrder = []string{"ET", "EO", "EI", "CK", "EY", "SN", "JW", "C0", "C1", "C2", "C3"}

var maskDepth = map[string]float64{"C0": 0.55, "C1": 0.75, "C2": 1, "C3": 0.55, "ET": 0.4, "EO": 0.2, "EI": 0.5, "CK": 0.1, "EY": 0.5, "SN": 0.75, "JW": 0.3}

var maskFacets = [][3]string{{"ET", "EO", "EI"}, {"EI", "C0", "EY"}, {"EI", "EY", "EO"}, {"EO", "EY", "CK"}, {"CK", "EY", "JW"}, {"EY", "SN", "JW"},
	{"C0", "C1", "EY"}, {"EY", "C1", "SN"}, {"C1", "C2", "SN"}, {"JW", "SN", "C3"}, {"SN", "C2", "C3"}}

type metal struct {
	Name string
	ramp [4]string // shadow, mid, light, highlight
}

var metals = []metal{
	{"Gold", [4]string{"#5A3F14", "#A67C2E", "#E3B95A", "#FFE7A3"}},
	{"Silver", [4]string{"#3C4250", "#7D8798", "#C3CCD8", "#F1F5FA"}},
	{"Bronze", [4]string{"#4A2A1A", "#8E5434", "#C98A5A", "#F0C29A"}},
	{"Jade", [4]string{"#173E33", "#2F7A62", "#6FBF9A", "#C8F0DC"}},
	{"Obsidian", [4]string{"#1B1830", "#36305A", "#6A61A0", "#B4ACE6"}},
	{"Blood", [4]string{"#3A0F12", "#7A1F24", "#C2453F", "#F2A08F"}},
	{"Bone", [4]string{"#4A4236", "#8C8270", "#CFC6B0", "#F6F1E4"}},
}

var (
	markBackgrounds = []string{"#0B0A0F", "#0D1420", "#170D10", "#0C1712", "#13101D"}
	markGlows       = [][2]string{{"Ember", "#FF5A4E"}, {"Ghost", "#5CF2E0"}, {"Violet", "#B48CFF"}, {"Lantern", "#FFD36E"}, {"Venom", "#9CFF7A"}}
	markEyes        = []string{"hollow", "slit", "glow", "void", "closed"}
	markSigils      = []string{"triangle", "crescent", "star", "eye", "ring", "cross", "none"}
	markTitles      = []string{"Silent", "Hollow", "Pale", "Gilded", "Sleepless", "Veiled", "Midnight", "Last", "Quiet", "Ninth", "Seventh", "Lantern", "Ashen", "Wandering", "Hidden", "Velvet"}
)

type markFrame struct {
	Name  string
	sides int
	rot   float64
	fit   float64 // how large the head may be inside it
}

var markFrames = []markFrame{{"Circle", 40, 0, 58}, {"Hexagon", 6, math.Pi / 6, 56}, {"Octagon", 8, math.Pi / 8, 58}, {"Diamond", 4, 0, 48}}

// markRanks: the first threshold above the rank draw wins.
var markRanks = []struct {
	name  string
	below float64
}{{"grandmaster", 0.02}, {"keeper", 0.08}, {"adept", 0.25}, {"initiate", 1}}

// Mark is every decision behind one society mark; For2 is pure.
type Mark struct {
	Name, Rank, Title, Number string
	Seed                      uint64
	Animal                    maskAnimal
	Metal                     metal
	Bg                        string
	Frame                     markFrame
	Eyes, Sigil               string
	Glow                      [2]string
	SX, Ears, Snout           float64
}

func For2(name string) Mark {
	key := Normalize(name)
	if key == "" {
		key = "stranger"
	}
	seed := hash("v2:" + key)
	next := mulberry32(uint32(seed))
	var d [16]float64
	for i := range d {
		d[i] = next()
	}
	m := Mark{Name: key, Seed: seed}
	for _, r := range markRanks {
		if d[0] < r.below {
			m.Rank = r.name
			break
		}
	}
	m.Animal = pick(d[1], maskAnimals)
	m.Metal = pick(d[2], metals[1:]) // gold is reserved for grandmasters
	m.Bg = pick(d[3], markBackgrounds)
	m.Frame = pick(d[4], markFrames)
	m.Eyes = pick(d[5], markEyes)
	m.Sigil = pick(d[6], markSigils)
	m.Glow = pick(d[7], markGlows)
	switch m.Rank {
	case "keeper":
		m.Eyes = "glow"
	case "grandmaster":
		m.Metal, m.Bg, m.Eyes, m.Sigil = metals[0], "#0B0A0F", "glow", "third eye"
	}
	span := func(v, lo, hi float64) float64 { return lo + float64(v*(hi-lo)) }
	m.SX, m.Ears, m.Snout = span(d[8], 0.92, 1.08), span(d[9], 0.85, 1.2), span(d[10], 0.9, 1.12)
	m.Title = "The " + pick(d[11], markTitles) + " " + m.Animal.Label
	m.Number = fmt.Sprintf("%04d", 1+int(math.Floor(d[12]*9999)))
	return m
}

// MarkSummary is the public description of a mark, for the info endpoint.
type MarkSummary struct {
	Name   string  `json:"name"`
	Rank   string  `json:"rank"`
	Title  string  `json:"title"`
	Number string  `json:"number"`
	Animal string  `json:"animal"`
	Metal  string  `json:"metal"`
	Eyes   string  `json:"eyes"`
	Glow   string  `json:"glow,omitempty"`
	Mark   string  `json:"mark"`
	Frame  string  `json:"frame"`
	Width  float64 `json:"face_width"`
	Ears   float64 `json:"ears"`
	Snout  float64 `json:"snout"`
	URL    string  `json:"url"`
}

func (m Mark) Summary() MarkSummary {
	s := MarkSummary{
		Name: m.Name, Rank: m.Rank, Title: m.Title, Number: m.Number, Animal: m.Animal.Label,
		Metal: m.Metal.Name, Eyes: m.Eyes, Mark: m.Sigil, Frame: m.Frame.Name,
		Width: round2(m.SX), Ears: round2(m.Ears), Snout: round2(m.Snout), URL: URLV2(m.Name),
	}
	if m.Eyes == "glow" || m.Eyes == "slit" {
		s.Glow = m.Glow[0]
	}
	return s
}

func round2(x float64) float64 { return math.Round(x*100) / 100 }

// MarkParts lists everything a mark can be made of, for the faces page.
type MarkParts struct {
	Animals []string           `json:"animals"`
	Metals  []string           `json:"metals"`
	Eyes    []string           `json:"eyes"`
	Glows   []string           `json:"glows"`
	Marks   []string           `json:"marks"`
	Frames  []string           `json:"frames"`
	Ranks   map[string]float64 `json:"ranks"` // rank -> chance, 0..1
}

func Parts() MarkParts {
	p := MarkParts{Eyes: slices.Clone(markEyes), Marks: slices.Clone(markSigils), Ranks: map[string]float64{}}
	for _, a := range maskAnimals {
		p.Animals = append(p.Animals, a.Label)
	}
	for _, m := range metals {
		p.Metals = append(p.Metals, m.Name)
	}
	for _, g := range markGlows {
		p.Glows = append(p.Glows, g[0])
	}
	for _, f := range markFrames {
		p.Frames = append(p.Frames, f.Name)
	}
	prev := 0.0
	for _, r := range markRanks {
		p.Ranks[r.name] = round2(r.below - prev)
		prev = r.below
	}
	return p
}
