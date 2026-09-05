// usage_test.go pins usageText's glyphs-verb and --view additions, following the pattern
// name_test.go's TestUsageText_NameVerb already establishes for a verb's usage line, its flag row,
// and the ASCII-only rule.

package cli

import (
	"strings"
	"testing"
)

// TestUsageText_GlyphsVerb asserts usageText carries the glyphs usage line, a --view <name> flag
// row, and remains ASCII only.
func TestUsageText_GlyphsVerb(t *testing.T) {
	if !strings.Contains(usageText, "quarry glyphs <target>") {
		t.Errorf("usageText = %q; want the glyphs usage line", usageText)
	}
	if !strings.Contains(usageText, "--view <name>") {
		t.Errorf("usageText = %q; want a --view <name> flag row", usageText)
	}
	for i := 0; i < len(usageText); i++ {
		if usageText[i] >= 0x80 {
			t.Fatalf("usageText contains a non-ASCII byte at index %d: %q", i, usageText[i])
		}
	}
}

// TestUsageText_GlyphsPresetMatchesVariable asserts that usageText contains
// strings.Join(glyphsPreset, " "), read from the variable, so the help text's spelling of the
// expansion and the real preset cannot drift apart.
//
// Reading the variable is correct here for the same reason it is correct in
// TestGlyphsPresetMatchesFacadeOptions (card 13) and wrong in
// TestGlyphsIsByteIdenticalToItsExpansion (card 12): this test asserts that two spellings of the
// same preset agree with each other, not that one of them matches a fixed, independently
// documented value. The discussion's preset-expansion decision names usageText as one of three
// places the preset's tokens are spelled — the other two are docs/rewrite-plan.md section 5 and
// glyphsPreset's own doc comment — and without this assertion nothing mechanically ties the help
// text to the real preset, so a batch verify could stay green while --help documents an expansion
// the CLI no longer runs.
func TestUsageText_GlyphsPresetMatchesVariable(t *testing.T) {
	want := strings.Join(glyphsPreset, " ")
	if !strings.Contains(usageText, want) {
		t.Errorf("usageText = %q; want it to contain %q, the real glyphsPreset spelling", usageText, want)
	}
}
