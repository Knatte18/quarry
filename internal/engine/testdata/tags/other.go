//go:build !linux

// other.go is the negated half of the build-tag duplicate pair described in linux.go.

package tags

// Dup is declared in both files of this package, as a function in each.
func Dup() string {
	return "other"
}

// DupType is declared in both files of this package, as a type in each.
type DupType struct {
	// Which names the constraint this declaration sits under.
	Which string
}

// Mixed is a function here and a type in linux.go, so one glyph names two different kinds.
func Mixed() {}
