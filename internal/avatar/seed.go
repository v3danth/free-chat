// Package avatar draws a pixel-art animal face from a name. The same name
// always gives the same face: the name is normalized, hashed, and the hash
// seeds a fixed list of random numbers that the traits are picked from.
//
// The arithmetic matches the browser prototype bit for bit (32-bit integer
// math on UTF-16 code units), so a face never depends on where it is drawn.
package avatar

import (
	"math"
	"strings"
	"unicode/utf16"

	"golang.org/x/text/unicode/norm"
)

// Version is part of every avatar URL. Bump it only when existing faces
// are meant to change, so caches holding the old images are bypassed.
const Version = "v1"

// slots is how many random numbers a name yields. Each trait owns one slot;
// new traits take new slots at the end so existing faces never change.
const slots = 24

const (
	slotTier = iota
	slotSpecies
	slotPalette
	slotHeadW
	slotHeadH
	slotRound
	slotEar
	slotEyeShape
	slotEyeSize
	slotEyeGap
	slotEyeY
	slotMuzzleW
	slotMuzzleH
	slotMouth
	slotBlush
	slotMarks
	slotAcc
	slotAccColor
	slotBg
	slotBg2
	slotSpots
)

// Normalize folds look-alike characters, case and spacing, so "Night_Owl "
// and "night_owl" are the same seed.
func Normalize(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(norm.NFKC.String(name)), " "))
}

// hash is cyrb53 over UTF-16 code units, as in JavaScript.
func hash(s string) uint64 {
	h1, h2 := uint32(0xdeadbeef), uint32(0x41c6ce57)
	for _, ch := range utf16.Encode([]rune(s)) {
		h1 = (h1 ^ uint32(ch)) * 2654435761
		h2 = (h2 ^ uint32(ch)) * 1597334677
	}
	h1 = (h1^(h1>>16))*2246822507 ^ (h2^(h2>>13))*3266489909
	h2 = (h2^(h2>>16))*2246822507 ^ (h1^(h1>>13))*3266489909
	return uint64(h2&2097151)<<32 | uint64(h1)
}

// mulberry32 is a tiny seeded generator of numbers in [0, 1).
func mulberry32(a uint32) func() float64 {
	return func() float64 {
		a += 0x6D2B79F5
		t := (a ^ a>>15) * (1 | a)
		t = (t + (t^t>>7)*(61|t)) ^ t
		return float64(t^t>>14) / 4294967296
	}
}

// draws is the fixed list of numbers a name yields.
func draws(name string) (seed uint64, d [slots]float64) {
	key := Normalize(name)
	if key == "" {
		key = "stranger"
	}
	seed = hash(key)
	next := mulberry32(uint32(seed))
	for i := range d {
		d[i] = next()
	}
	return seed, d
}

type choice struct {
	item   string
	weight float64
}

func pick[T any](v float64, items []T) T {
	return items[min(len(items)-1, int(math.Floor(v*float64(len(items)))))]
}

func pickInt(v float64, r [2]int) int {
	return r[0] + min(r[1]-r[0], int(math.Floor(v*float64(r[1]-r[0]+1))))
}

func weighted(v float64, choices []choice) string {
	total := 0.0
	for _, c := range choices {
		total += c.weight
	}
	x := v * total
	for _, c := range choices {
		if x -= c.weight; x < 0 {
			return c.item
		}
	}
	return choices[len(choices)-1].item
}

// jsRound is JavaScript's Math.round: halves round up, not away from zero.
func jsRound(x float64) int { return int(math.Floor(x + 0.5)) }
