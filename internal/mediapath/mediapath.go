// Package mediapath is the single definition of where image variants live on
// disk and at which URLs they are served. It has no dependencies so any
// package can build media URLs without importing the media service.
package mediapath

import (
	"path"
	"strings"
)

const URLPrefix = "/media/"

type Variant string

const (
	Full  Variant = "full"  // longest side 1280px
	Thumb Variant = "thumb" // longest side 320px
)

var Variants = []Variant{Full, Thumb}

// AnimatedPrefix starts the key of an animated GIF. Keys are otherwise
// lowercase base32 (a-z, 2-7), so "0" never begins a still image's key and
// the key alone says which file type to serve: no column, no lookup.
const AnimatedPrefix = "0"

func Animated(key string) bool { return strings.HasPrefix(key, AnimatedPrefix) }

// File is the path of a variant relative to the storage root.
func File(v Variant, key string) string {
	ext := ".jpg"
	if Animated(key) {
		ext = ".gif"
	}
	return path.Join(string(v), key+ext)
}

func URL(v Variant, key string) string { return URLPrefix + File(v, key) }
