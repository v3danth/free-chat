package avatar

import (
	"slices"
	"strconv"
)

// Traits is every decision behind one face. It is a pure function of the
// name: For never touches storage, the clock or global state.
type Traits struct {
	Name     string // normalized
	Seed     uint64
	Tier     string
	Species  Species
	Palette  Palette
	HeadW    int
	HeadH    int
	Round    float64
	EarSize  int
	EyeShape string
	EyeSize  int
	EyeGap   int
	EyeY     int
	Muzzle   *[2]int // width, height
	Mouth    string
	Blush    bool
	Marks    string
	Acc      string
	AccColor string
	Bg, Bg2  background
	spots    float64
}

func For(name string) Traits {
	seed, d := draws(name)
	t := Traits{Name: Normalize(name), Seed: seed, spots: d[slotSpots]}

	for _, tier := range tiers {
		if d[slotTier] < tier.below {
			t.Tier = tier.name
			break
		}
	}
	byWeight := make([]choice, len(species))
	for i, sp := range species {
		byWeight[i] = choice{sp.Key, sp.weight}
	}
	key := weighted(d[slotSpecies], byWeight)
	sp := species[slices.IndexFunc(species, func(s Species) bool { return s.Key == key })]
	t.Species = sp

	t.Palette = pick(d[slotPalette], sp.Palettes)
	if t.Tier == "legendary" {
		t.Palette = sp.Shiny
	}
	t.HeadW = pickInt(d[slotHeadW], sp.HeadW)
	t.HeadH = pickInt(d[slotHeadH], sp.HeadH)
	t.Round, _ = strconv.ParseFloat(strconv.FormatFloat(sp.Round[0]+float64(d[slotRound]*(sp.Round[1]-sp.Round[0])), 'f', 2, 64), 64)
	t.EarSize = pickInt(d[slotEar], sp.EarSize)

	t.EyeShape = pick(d[slotEyeShape], sp.EyeShapes)
	if t.Tier == "epic" && slices.Contains(sp.EyeShapes, "sparkle") {
		t.EyeShape = "sparkle"
	}
	limit := eyeLimits[t.EyeShape]
	lo := max(limit[0], sp.EyeSize[0])
	t.EyeSize = pickInt(d[slotEyeSize], [2]int{lo, max(lo, min(limit[1], sp.EyeSize[1]))})
	t.EyeGap = pickInt(d[slotEyeGap], sp.EyeGap)
	t.EyeY = pickInt(d[slotEyeY], sp.EyeY)

	if sp.Muzzle != nil {
		t.Muzzle = &[2]int{pickInt(d[slotMuzzleW], sp.Muzzle[0]), pickInt(d[slotMuzzleH], sp.Muzzle[1])}
	}
	t.Mouth = pick(d[slotMouth], sp.Mouths)
	t.Blush = d[slotBlush] < sp.Blush
	t.Marks = weighted(d[slotMarks], sp.marks)

	var table []choice
	for _, a := range accessories {
		if !slices.Contains(sp.noAcc, a.item) {
			table = append(table, a)
		}
	}
	table = append(table, sp.accBias...)
	if sp.Key == "frog" {
		table = append(table, choice{"leaf", 10})
	}
	t.Acc = weighted(d[slotAcc], table)
	if t.Tier == "epic" || t.Tier == "legendary" {
		t.Acc = "halo"
		if d[slotAccColor] < 0.5 {
			t.Acc = "crown"
		}
	}
	t.AccColor = pick(d[slotAccColor], accColors)

	t.Bg = pick(d[slotBg], backgrounds)
	others := slices.DeleteFunc(slices.Clone(backgrounds), func(b background) bool { return b == t.Bg })
	t.Bg2 = pick(d[slotBg2], others)
	return t
}

// Summary is the public description of a face, for the info endpoint.
type Summary struct {
	Name     string  `json:"name"`
	Tier     string  `json:"tier"`
	Species  string  `json:"species"`
	Animal   string  `json:"animal"`
	Colour   string  `json:"colour"`
	Face     [2]int  `json:"face"`
	Round    float64 `json:"roundness"`
	Ears     string  `json:"ears"`
	EarSize  int     `json:"ear_size"`
	EyeShape string  `json:"eye_shape"`
	EyeSize  int     `json:"eye_size"`
	EyeGap   int     `json:"eye_gap"`
	Muzzle   *[2]int `json:"muzzle,omitempty"`
	Mouth    string  `json:"mouth"`
	Marks    string  `json:"markings"`
	Blush    bool    `json:"blush"`
	Extra    string  `json:"extra"`
	URL      string  `json:"url"`
}

func (t Traits) Summary() Summary {
	return Summary{
		Name: t.Name, Tier: t.Tier, Species: t.Species.Key, Animal: t.Species.Label, Colour: t.Palette.Name,
		Face: [2]int{t.HeadW, t.HeadH}, Round: t.Round, Ears: t.Species.Ears, EarSize: t.EarSize,
		EyeShape: t.EyeShape, EyeSize: t.EyeSize, EyeGap: t.EyeGap, Muzzle: t.Muzzle, Mouth: t.Mouth,
		Marks: t.Marks, Blush: t.Blush, Extra: t.Acc, URL: URL(t.Name),
	}
}

// Rule is one animal's rules, as shown on the faces page.
type Rule struct {
	Species   string     `json:"species"`
	Animal    string     `json:"animal"`
	Tier      string     `json:"tier"`
	Chance    float64    `json:"chance"` // of drawing this animal, 0..1
	FaceW     [2]int     `json:"face_w"`
	FaceH     [2]int     `json:"face_h"`
	Roundness [2]float64 `json:"roundness"`
	Ears      string     `json:"ears"`
	EarSize   [2]int     `json:"ear_size"`
	EyeShapes []string   `json:"eye_shapes"`
	EyeSize   [2]int     `json:"eye_size"`
	EyeGap    [2]int     `json:"eye_gap"`
	Muzzle    *[2][2]int `json:"muzzle,omitempty"`
	Mouths    []string   `json:"mouths"`
	Colours   []string   `json:"colours"`
	Shiny     string     `json:"shiny"`
	NoExtras  []string   `json:"no_extras,omitempty"`
	Example   string     `json:"example"` // a name that draws this animal
}

// Rules lists every species' rules, read straight from the generator's data.
func Rules() []Rule {
	total := 0.0
	for _, sp := range species {
		total += sp.weight
	}
	rules := make([]Rule, len(species))
	for i, sp := range species {
		colours := make([]string, len(sp.Palettes))
		for j, p := range sp.Palettes {
			colours[j] = p.Name
		}
		rules[i] = Rule{
			Species: sp.Key, Animal: sp.Label, Tier: sp.Tier, Chance: sp.weight / total,
			FaceW: sp.HeadW, FaceH: sp.HeadH, Roundness: sp.Round, Ears: sp.Ears, EarSize: sp.EarSize,
			EyeShapes: slices.Compact(slices.Sorted(slices.Values(sp.EyeShapes))), EyeSize: sp.EyeSize, EyeGap: sp.EyeGap,
			Muzzle: sp.Muzzle, Mouths: slices.Compact(slices.Sorted(slices.Values(sp.Mouths))),
			Colours: colours, Shiny: sp.Shiny.Name, NoExtras: sp.noAcc, Example: exampleOf(sp.Key),
		}
	}
	return rules
}

// exampleOf finds the first "<species>_<n>" name that draws that species.
func exampleOf(key string) string {
	for i := range 10000 {
		if name := key + "_" + strconv.Itoa(i); For(name).Species.Key == key {
			return name
		}
	}
	return ""
}
