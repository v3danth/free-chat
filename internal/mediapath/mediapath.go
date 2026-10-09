// Package mediapath is the single definition of where image variants live on
// disk and at which URLs they are served. It has no dependencies so any
// package can build media URLs without importing the media service.
package mediapath

import "path"

const URLPrefix = "/media/"

type Variant string

const (
	Full  Variant = "full"  // longest side 1280px
	Thumb Variant = "thumb" // longest side 320px
)

var Variants = []Variant{Full, Thumb}

// File is the path of a variant relative to the storage root.
func File(v Variant, key string) string { return path.Join(string(v), key+".jpg") }

func URL(v Variant, key string) string { return URLPrefix + File(v, key) }
