//go:build linux

// linux.go is one half of the build-tag duplicate pair. Its sibling other.go declares the same
// three names under the negated constraint, so every name here matches twice. Nothing under
// testdata/ is ever built, so the pair is data, not code.

package tags

// Dup is declared in both files of this package, as a function in each.
func Dup() string {
	return "linux"
}

// DupType is declared in both files of this package, as a type in each.
type DupType struct {
	// Which names the constraint this declaration sits under.
	Which string
}

// Mixed is a type here and a function in other.go, so one glyph names two different kinds.
type Mixed struct{}
