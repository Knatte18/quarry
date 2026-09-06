// unitpath.go declares UnitPath, the path direction: the API form of the §3 round-trip property
// that a Go glyph's unit half is itself the repository-relative path.

package glyph

// UnitPath returns the repository-relative path the glyph's unit half denotes, when the glyph's
// alphabet spells units as paths. It answers from the parsed struct alone: no filesystem access, no
// extraction, and no string surgery on the glyph's printed form. ok is false when the alphabet does
// not spell units as paths, and for the zero Glyph, whose Lang is not a valid language.
//
// Every alphabet decides this for itself, and a new alphabet must decide it rather than inherit an
// answer: Go's unit is a repository-relative path — the package directory for a package unit, the
// file for a file unit — so Go returns it unchanged with ok true (docs/glyph.md §3). A Python
// dotted module path would decide true only after its own conversion; a C# namespace is not a path
// at all and would decide false. Go is the only Language this package implements today.
//
// The path is returned exactly as the unit is spelled: repository-relative, forward slashes, no
// filepath.Join and no OS-specific separators. Callers own disk semantics.
func (g Glyph) UnitPath() (path string, ok bool) {
	switch g.Lang {
	case Go:
		return g.Unit, true
	default:
		return "", false
	}
}
