// self.go declares Self, the compose constructor for the self form: it appends "#" to a path and
// delegates the whole result to Parse, so every unit rule Parse enforces applies here unchanged.

package glyph

// Self builds the self-form Glyph for path in lang, validating by delegation rather than by a
// second copy of the unit alphabet: it appends "#" to path and returns Parse(lang, path+"#")
// directly. A "#" inside path therefore surfaces as ReasonMultipleSeparators, an empty path as
// ReasonUnitEmpty, and every other unit rule — dot segments, empty segments, bad runes, unsupported
// language — comes along unchanged and can never drift from Parse's. This is the one concatenation
// in the whole system; no consumer performs it. lang is never defaulted to Go: Language's zero
// value is deliberately not a valid language, so a forgotten argument fails at the first call, and
// hardcoding Go would reintroduce exactly that silent default.
func Self(lang Language, path string) (Glyph, error) {
	return Parse(lang, path+"#")
}
