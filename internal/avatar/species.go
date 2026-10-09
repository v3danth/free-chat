package avatar

// Palette is one colour scheme an animal can come in.
type Palette struct {
	Name                                             string
	Fur, Shade, Light, Inner, Nose, Ear, Patch, Iris string
}

type extra struct{ inner, nose, ear, patch, iris string }

func pal(name, fur, shade, light string, x extra) Palette {
	or := func(v, def string) string {
		if v == "" {
			return def
		}
		return v
	}
	return Palette{
		Name: name, Fur: fur, Shade: shade, Light: light,
		Inner: or(x.inner, shade), Nose: or(x.nose, "#2A2025"), Ear: or(x.ear, shade),
		Patch: or(x.patch, ink), Iris: or(x.iris, "#9BD45A"),
	}
}

// Species is the set of rules one animal follows. Every range is inclusive.
type Species struct {
	Key, Label, Tier string
	weight           float64
	HeadW, HeadH     [2]int
	Round            [2]float64 // 2 is an ellipse; higher is boxier
	Ears             string
	EarSize          [2]int
	EyeShapes        []string
	EyeSize, EyeGap  [2]int
	EyeY             [2]int
	Muzzle           *[2][2]int // width range, height range; nil for none
	Nose             string
	Mouths           []string
	Blush            float64 // chance of blush
	marks            []choice
	Palettes         []Palette
	Shiny            Palette
	noAcc            []string
	accBias          []choice
}

func muzzle(w, h [2]int) *[2][2]int { return &[2][2]int{w, h} }

// species is ordered: the order is part of the seed format.
var species = []Species{
	{Key: "cat", Label: "Cat", Tier: "common", weight: 3,
		HeadW: [2]int{15, 19}, HeadH: [2]int{13, 16}, Round: [2]float64{2, 2.5}, Ears: "pointy", EarSize: [2]int{4, 6},
		EyeShapes: []string{"round", "slit", "happy", "sleepy", "sparkle", "dot", "side"}, EyeSize: [2]int{1, 3}, EyeGap: [2]int{2, 4}, EyeY: [2]int{-2, 0},
		Nose: "tri", Mouths: []string{"w", "w", "smile", "tongue", "fang", "o"}, Blush: 0.5, marks: []choice{{"none", 3}, {"stripes", 2}},
		Palettes: []Palette{
			pal("Ginger", "#F0A35E", "#C97A3D", "#FBE3C4", extra{inner: "#F7B7B0", nose: "#E07A8A", iris: "#9BD45A"}),
			pal("Grey", "#9AA0AE", "#727889", "#E3E5EA", extra{inner: "#F2B4BE", nose: "#E58B9B", iris: "#F2C94C"}),
			pal("Midnight", "#423E4C", "#2E2B36", "#5C5868", extra{inner: "#C98A97", nose: "#D87C8E", iris: "#F2D04C"}),
			pal("Snow", "#F4EFE6", "#D5CEC2", "#FFFFFF", extra{inner: "#F6B9C2", nose: "#EE8FA0", iris: "#7EC8E3"}),
		},
		Shiny: pal("Shiny lilac", "#B9A6FF", "#8E79E0", "#EDE6FF", extra{inner: "#FFC1E3", nose: "#FF8FC8", iris: "#7DF9FF"})},
	{Key: "dog", Label: "Dog", Tier: "common", weight: 3,
		HeadW: [2]int{15, 18}, HeadH: [2]int{13, 16}, Round: [2]float64{2, 2.4}, Ears: "floppy", EarSize: [2]int{4, 6},
		EyeShapes: []string{"round", "dot", "happy", "sparkle", "tall", "sleepy"}, EyeSize: [2]int{1, 3}, EyeGap: [2]int{2, 4}, EyeY: [2]int{-2, -1},
		Muzzle: muzzle([2]int{6, 8}, [2]int{3, 4}), Nose: "dot", Mouths: []string{"smile", "tongue", "tongue", "open", "line"}, Blush: 0.25,
		marks: []choice{{"none", 3}, {"eyepatch", 1}},
		Palettes: []Palette{
			pal("Golden", "#E2B86F", "#B88E4E", "#F8E7C5", extra{ear: "#B07F42"}),
			pal("Cocoa", "#9A6440", "#744A2E", "#D9B48E", extra{ear: "#5E3A22"}),
			pal("Shiba", "#E39A4C", "#B87635", "#FBEEDC", extra{ear: "#B87635"}),
			pal("Cream", "#F2EDE4", "#CFC7BA", "#FFFFFF", extra{ear: "#8C7B6A"}),
		},
		Shiny: pal("Shiny mint", "#7FE0C4", "#4FB89C", "#D8FFF3", extra{ear: "#3E9C82"})},
	{Key: "bunny", Label: "Bunny", Tier: "common", weight: 3,
		HeadW: [2]int{14, 17}, HeadH: [2]int{12, 14}, Round: [2]float64{2, 2.2}, Ears: "long", EarSize: [2]int{5, 8},
		EyeShapes: []string{"round", "dot", "happy", "sparkle", "sleepy", "big"}, EyeSize: [2]int{1, 3}, EyeGap: [2]int{2, 4}, EyeY: [2]int{-1, 0},
		Nose: "tri", Mouths: []string{"w", "w", "o", "smile"}, Blush: 0.7, marks: []choice{{"none", 1}},
		Palettes: []Palette{
			pal("Cotton", "#F6F2EC", "#D9D2C8", "#FFFFFF", extra{inner: "#F7B6C4", nose: "#EE8FA0"}),
			pal("Mocha", "#B08A6E", "#8C6A52", "#E8D6C6", extra{inner: "#F2B4BE", nose: "#E07A8A"}),
			pal("Ash", "#A9A6B2", "#86838F", "#E6E4EA", extra{inner: "#F2B4BE", nose: "#E58B9B"}),
		},
		Shiny: pal("Shiny bubblegum", "#FFB3D9", "#F08CC0", "#FFE3F2", extra{inner: "#FFFFFF", nose: "#E0559A"}),
		noAcc: []string{"cap", "beanie", "headphones"}},
	{Key: "bear", Label: "Bear", Tier: "common", weight: 3,
		HeadW: [2]int{16, 20}, HeadH: [2]int{13, 16}, Round: [2]float64{2, 2.3}, Ears: "round", EarSize: [2]int{2, 3},
		EyeShapes: []string{"dot", "round", "happy", "sleepy", "sparkle"}, EyeSize: [2]int{1, 2}, EyeGap: [2]int{2, 4}, EyeY: [2]int{-2, -1},
		Muzzle: muzzle([2]int{6, 7}, [2]int{3, 4}), Nose: "dot", Mouths: []string{"smile", "line", "w", "open"}, Blush: 0.4, marks: []choice{{"none", 1}},
		Palettes: []Palette{
			pal("Brown", "#9C6B45", "#7A5134", "#D8B28E", extra{inner: "#D8B28E"}),
			pal("Honey", "#D9A15B", "#B07B3C", "#F4D9AE", extra{inner: "#F4D9AE"}),
			pal("Black", "#3E3A44", "#2C2932", "#9C8C7E", extra{inner: "#7A6E66", nose: "#16131A"}),
		},
		Shiny: pal("Shiny sky", "#7FD4FF", "#4FAEE0", "#E2F6FF", extra{inner: "#E2F6FF"})},
	{Key: "frog", Label: "Frog", Tier: "common", weight: 3,
		HeadW: [2]int{17, 20}, HeadH: [2]int{11, 13}, Round: [2]float64{2.2, 2.6}, Ears: "bumps", EarSize: [2]int{0, 0},
		EyeShapes: []string{"round", "side", "sleepy", "happy", "big"}, EyeSize: [2]int{1, 2}, EyeGap: [2]int{2, 3}, EyeY: [2]int{0, 0},
		Nose: "none", Mouths: []string{"wide", "wide", "smile", "tongue"}, Blush: 0.6, marks: []choice{{"none", 2}, {"spots", 1}},
		Palettes: []Palette{
			pal("Leaf", "#7DCB6B", "#57A04B", "#D8F2B7", extra{}),
			pal("Pond", "#5FBFA4", "#3F9A80", "#CFF2E6", extra{}),
			pal("Lime", "#B5DB5A", "#8DB43C", "#EEF8C8", extra{}),
		},
		Shiny: pal("Shiny pink", "#FF9FCB", "#E676AA", "#FFE0EE", extra{}),
		noAcc: []string{"headphones", "beanie"}},
	{Key: "mouse", Label: "Mouse", Tier: "common", weight: 3,
		HeadW: [2]int{13, 16}, HeadH: [2]int{12, 14}, Round: [2]float64{2, 2.2}, Ears: "round", EarSize: [2]int{3, 4},
		EyeShapes: []string{"dot", "round", "big", "sparkle"}, EyeSize: [2]int{1, 2}, EyeGap: [2]int{2, 3}, EyeY: [2]int{-1, 0},
		Muzzle: muzzle([2]int{4, 5}, [2]int{2, 3}), Nose: "tri", Mouths: []string{"w", "o", "smile"}, Blush: 0.6, marks: []choice{{"none", 1}},
		Palettes: []Palette{
			pal("Field", "#B49A86", "#8F7765", "#EFE3D8", extra{inner: "#F4AFC0", nose: "#E58B9B"}),
			pal("Silver", "#B4B4BE", "#8E8E9A", "#ECECF1", extra{inner: "#F4AFC0", nose: "#E58B9B"}),
		},
		Shiny: pal("Shiny butter", "#FFE36E", "#E8C344", "#FFF6C9", extra{inner: "#FF9EBB", nose: "#E0559A"})},
	{Key: "fox", Label: "Fox", Tier: "rare", weight: 2,
		HeadW: [2]int{15, 18}, HeadH: [2]int{12, 15}, Round: [2]float64{2, 2.4}, Ears: "pointy", EarSize: [2]int{5, 7},
		EyeShapes: []string{"slit", "happy", "sleepy", "side", "sparkle"}, EyeSize: [2]int{1, 2}, EyeGap: [2]int{2, 4}, EyeY: [2]int{-2, -1},
		Nose: "dot", Mouths: []string{"w", "smile", "fang"}, Blush: 0.3, marks: []choice{{"cheeks", 1}},
		Palettes: []Palette{
			pal("Red fox", "#E8783A", "#B9572A", "#FFF4E8", extra{inner: "#3A2A2A", iris: "#F2B33D"}),
			pal("Arctic", "#E9EEF2", "#C3CCD6", "#FFFFFF", extra{inner: "#9AA5B5", iris: "#6FB7E9"}),
		},
		Shiny: pal("Shiny ocean", "#5FA8FF", "#3D7FD6", "#E6F2FF", extra{inner: "#1E3A66", iris: "#FFE36E"})},
	{Key: "panda", Label: "Panda", Tier: "rare", weight: 2,
		HeadW: [2]int{16, 19}, HeadH: [2]int{13, 15}, Round: [2]float64{2, 2.3}, Ears: "round", EarSize: [2]int{2, 3},
		EyeShapes: []string{"dot", "round", "sparkle", "sleepy"}, EyeSize: [2]int{1, 2}, EyeGap: [2]int{2, 3}, EyeY: [2]int{-1, 0},
		Muzzle: muzzle([2]int{5, 6}, [2]int{3, 3}), Nose: "dot", Mouths: []string{"smile", "w", "line"}, Blush: 0.5, marks: []choice{{"patches", 1}},
		Palettes: []Palette{
			pal("Classic", "#F7F4EF", "#D9D3CA", "#FFFFFF", extra{patch: "#2C2933", ear: "#2C2933", inner: "#2C2933", nose: "#2C2933"}),
		},
		Shiny: pal("Shiny berry", "#FFF1D6", "#E6D3B0", "#FFFFFF", extra{patch: "#D9467A", ear: "#D9467A", inner: "#D9467A", nose: "#8E1F48"})},
	{Key: "raccoon", Label: "Raccoon", Tier: "rare", weight: 2,
		HeadW: [2]int{16, 19}, HeadH: [2]int{12, 15}, Round: [2]float64{2, 2.4}, Ears: "pointy", EarSize: [2]int{3, 4},
		EyeShapes: []string{"dot", "round", "side", "sparkle"}, EyeSize: [2]int{1, 2}, EyeGap: [2]int{2, 4}, EyeY: [2]int{-2, -1},
		Muzzle: muzzle([2]int{5, 6}, [2]int{3, 3}), Nose: "dot", Mouths: []string{"fang", "smile", "w"}, Blush: 0.2, marks: []choice{{"mask", 1}},
		Palettes: []Palette{
			pal("Trash panda", "#A3A2AA", "#7C7B85", "#E8E6EA", extra{patch: "#3A3842", inner: "#3A3842"}),
		},
		Shiny: pal("Shiny dusk", "#C9B8FF", "#9D89E0", "#F0EBFF", extra{patch: "#4A2E7A", inner: "#4A2E7A"})},
	{Key: "hamster", Label: "Hamster", Tier: "rare", weight: 2,
		HeadW: [2]int{16, 20}, HeadH: [2]int{13, 16}, Round: [2]float64{2, 2.2}, Ears: "round", EarSize: [2]int{2, 2},
		EyeShapes: []string{"big", "dot", "sparkle", "happy"}, EyeSize: [2]int{1, 2}, EyeGap: [2]int{2, 4}, EyeY: [2]int{-2, -1},
		Nose: "tri", Mouths: []string{"w", "o", "smile"}, Blush: 0.8, marks: []choice{{"puffs", 1}},
		Palettes: []Palette{
			pal("Syrian", "#E9B07A", "#C48A55", "#FFF5EA", extra{inner: "#F4A9A8", nose: "#E08A96"}),
			pal("Dove", "#C8C2BC", "#A39C96", "#FBF8F5", extra{inner: "#F4A9A8", nose: "#E08A96"}),
		},
		Shiny: pal("Shiny gold", "#FFD84A", "#E0B322", "#FFF6CC", extra{inner: "#FF9EBB", nose: "#E0559A"})},
	{Key: "owl", Label: "Owl", Tier: "rare", weight: 2,
		HeadW: [2]int{15, 18}, HeadH: [2]int{13, 16}, Round: [2]float64{2.2, 2.6}, Ears: "tufts", EarSize: [2]int{3, 4},
		EyeShapes: []string{"round", "big", "sleepy", "side"}, EyeSize: [2]int{2, 3}, EyeGap: [2]int{1, 2}, EyeY: [2]int{-1, 0},
		Nose: "beak", Mouths: []string{"none"}, Blush: 0.2, marks: []choice{{"discs", 1}},
		Palettes: []Palette{
			pal("Barn", "#B08A62", "#8A6847", "#F1E4D0", extra{nose: "#F2A93B"}),
			pal("Snowy", "#EEEAE2", "#C9C2B6", "#FFFFFF", extra{nose: "#3A3530"}),
		},
		Shiny: pal("Shiny ember", "#FF8A5B", "#D9643A", "#FFE2D2", extra{nose: "#5A2A14"})},
	{Key: "pig", Label: "Pig", Tier: "rare", weight: 2,
		HeadW: [2]int{16, 19}, HeadH: [2]int{13, 15}, Round: [2]float64{2, 2.3}, Ears: "pig", EarSize: [2]int{3, 4},
		EyeShapes: []string{"dot", "happy", "sleepy", "round"}, EyeSize: [2]int{1, 2}, EyeGap: [2]int{3, 4}, EyeY: [2]int{-2, -1},
		Nose: "snout", Mouths: []string{"smile", "line", "open"}, Blush: 0.7, marks: []choice{{"none", 1}},
		Palettes: []Palette{pal("Pink", "#F6B6B8", "#E08D93", "#FFD5D7", extra{nose: "#F08A95"})},
		Shiny:    pal("Shiny matcha", "#B8F0C0", "#86D093", "#E4FFE8", extra{nose: "#5DB06D"})},
	{Key: "capybara", Label: "Capybara", Tier: "epic", weight: 1,
		HeadW: [2]int{16, 19}, HeadH: [2]int{13, 16}, Round: [2]float64{2.6, 3.2}, Ears: "round", EarSize: [2]int{1, 2},
		EyeShapes: []string{"sleepy", "sleepy", "dot", "happy"}, EyeSize: [2]int{1, 2}, EyeGap: [2]int{3, 5}, EyeY: [2]int{-3, -2},
		Muzzle: muzzle([2]int{9, 12}, [2]int{4, 5}), Nose: "wide", Mouths: []string{"line", "smile"}, Blush: 0.3, marks: []choice{{"none", 1}},
		Palettes: []Palette{
			pal("Wild", "#A97C55", "#83603F", "#C49A72", extra{inner: "#6E4F34", nose: "#3B2A20"}),
			pal("Dusk", "#8C6A4E", "#6C5039", "#A98564", extra{inner: "#5A4130", nose: "#2E2018"}),
		},
		Shiny:   pal("Shiny gold", "#F4C542", "#CC9E1F", "#FFE08A", extra{inner: "#A87A10", nose: "#5A3E06"}),
		accBias: []choice{{"orange", 30}}},
	{Key: "axolotl", Label: "Axolotl", Tier: "epic", weight: 1,
		HeadW: [2]int{16, 19}, HeadH: [2]int{12, 14}, Round: [2]float64{2.2, 2.6}, Ears: "gills", EarSize: [2]int{3, 4},
		EyeShapes: []string{"dot", "big", "happy", "sparkle"}, EyeSize: [2]int{1, 2}, EyeGap: [2]int{3, 5}, EyeY: [2]int{-1, 0},
		Nose: "none", Mouths: []string{"wide", "smile", "w"}, Blush: 0.9, marks: []choice{{"none", 2}, {"spots", 1}},
		Palettes: []Palette{
			pal("Pink", "#F7B5CD", "#E38CAE", "#FFDDEA", extra{inner: "#E2507F"}),
			pal("Leucistic", "#F5EFF2", "#D9CED4", "#FFFFFF", extra{inner: "#F0709A"}),
			pal("Wild", "#7A7F8C", "#5E6270", "#A9AEBB", extra{inner: "#C77DA0"}),
		},
		Shiny: pal("Shiny cosmic", "#9AD8FF", "#6FB3E6", "#E2F5FF", extra{inner: "#5A6BFF"}),
		noAcc: []string{"headphones"}},
}

// eyeLimits bounds the size of each eye shape, whatever the species allows.
var eyeLimits = map[string][2]int{
	"dot": {1, 3}, "round": {1, 3}, "tall": {1, 2}, "sleepy": {1, 3}, "happy": {1, 3},
	"sparkle": {2, 3}, "slit": {2, 3}, "big": {1, 2}, "side": {1, 2},
}

type background struct{ Name, Color string }

var backgrounds = []background{
	{"Blush", "#FFD6E0"}, {"Sky", "#CDEBFF"}, {"Mint", "#D4F2CC"}, {"Butter", "#FFF1C2"},
	{"Lilac", "#E6DBFF"}, {"Peach", "#FFDCC2"}, {"Seafoam", "#C9F2EC"}, {"Cloud", "#ECECF0"},
}

var accColors = []string{"#E5484D", "#3E7BFA", "#2BB673", "#F5A524", "#8E4EC6", "#F06BA8"}

var accessories = []choice{
	{"none", 40}, {"cap", 8}, {"beanie", 8}, {"headphones", 7}, {"flower", 8}, {"bow", 6}, {"bandaid", 6}, {"shades", 5},
}

// tiers maps the tier draw to a rarity: the first threshold above it wins.
var tiers = []struct {
	name  string
	below float64
}{{"legendary", 0.02}, {"epic", 0.08}, {"rare", 0.25}, {"common", 1}}
