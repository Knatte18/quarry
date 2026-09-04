// root.go sits directly in testdata/units/, the fixture glyph_test.go opens as its own repository
// root to exercise the empty-unit case: queried from that root, this file's own unit is "" — the
// repository root's unit, per unitFor's own doc comment — and "" is never spellable. Queried from
// the quarry root instead, this file's unit is the ordinary, spellable
// "internal/engine/testdata/units", so it carries symbols in the whole-repository round trip same
// as any other committed fixture.

package units

// Placeholder is listed with a header but never with symbols when this directory is queried as its
// own repository root, since "" is not a spellable unit.
func Placeholder() {}
