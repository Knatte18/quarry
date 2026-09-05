// glyphs_test.go is this task's central enforcement mechanism, and it is deliberately not a
// golden file: a golden pins one invocation's bytes against a fixed committed value, while the
// two tests here pin that the "glyphs" verb and its documented "toc" expansion never diverge from
// each other, on either side of the CLI/facade boundary. A failure in either test means the
// glyphs preset has grown a second code path rather than remaining a pure argv rewrite — the whole
// point of the design this task delivers.

package cli

import (
	"bytes"
	"testing"

	"github.com/Knatte18/quarry/quarry"
)

// TestGlyphsIsByteIdenticalToItsExpansion is gated on loomyardRepo(t) exactly as TestAfterGoldens
// is: it needs a real repository tree with real symbols to project. It covers the cross product of
// a file target and a directory target, in JSON and under --text, using internal/logger and
// internal/logger/logger.go — the same two targets the existing golden table uses, so the fixture
// set does not grow.
//
// For each pair, Run is called twice into separate bytes.Buffer pairs: once with the glyphs argv
// and once with the expansion argv. All three of stdout, stderr and the returned exit code must be
// identical between the two runs, and the exit code must additionally be exitOK with empty stderr,
// so a pair that fails identically in both runs cannot pass as a match.
//
// The expansion's flag tokens are spelled literally below, rather than read from glyphsPreset: the
// whole purpose of this test is to catch a change to that variable that is not matched by a change
// to the documented expansion, and reading the variable here would make the test pass by
// construction regardless of what glyphsPreset says. The same tokens are spelled in three other
// places that must move together with a deliberate change here: usageText, docs/rewrite-plan.md
// section 5, and glyphsPreset's own doc comment in internal/cli/flags.go.
func TestGlyphsIsByteIdenticalToItsExpansion(t *testing.T) {
	repo := loomyardRepo(t)

	targets := []string{"internal/logger", "internal/logger/logger.go"}
	for _, target := range targets {
		for _, withText := range []bool{false, true} {
			name := target
			if withText {
				name += "-text"
			}
			t.Run(name, func(t *testing.T) {
				glyphsArgs := []string{"glyphs", "--root", repo}
				expansionArgs := []string{
					"toc", "--root", repo,
					"--view", "glyphs", "--depth", "all", "--symbols",
				}
				if withText {
					glyphsArgs = append(glyphsArgs, "--text")
					expansionArgs = append(expansionArgs, "--text")
				}
				glyphsArgs = append(glyphsArgs, target)
				expansionArgs = append(expansionArgs, target)

				var glyphsOut, glyphsErr bytes.Buffer
				glyphsCode := Run(glyphsArgs, &glyphsOut, &glyphsErr)

				var expansionOut, expansionErr bytes.Buffer
				expansionCode := Run(expansionArgs, &expansionOut, &expansionErr)

				if glyphsCode != exitOK {
					t.Fatalf("Run(%v) code = %d, stderr = %q; want %d", glyphsArgs, glyphsCode, glyphsErr.String(), exitOK)
				}
				if glyphsErr.Len() != 0 {
					t.Fatalf("Run(%v) stderr = %q; want empty", glyphsArgs, glyphsErr.String())
				}

				if glyphsCode != expansionCode {
					t.Errorf("code = %d; want %d (the expansion's own code)", glyphsCode, expansionCode)
				}
				if glyphsOut.String() != expansionOut.String() {
					t.Errorf("stdout = %q; want %q (byte-identical to the expansion)", glyphsOut.String(), expansionOut.String())
				}
				if glyphsErr.String() != expansionErr.String() {
					t.Errorf("stderr = %q; want %q (byte-identical to the expansion)", glyphsErr.String(), expansionErr.String())
				}
			})
		}
	}
}

// TestGlyphsPresetMatchesFacadeOptions needs no repository and no checkout gate: it parses the
// CLI's own preset through parseArgs and compares the result against the facade's frozen options,
// so a drift between the two sides of the CLI/facade boundary is caught with no fixture at all.
//
// Unlike TestGlyphsIsByteIdenticalToItsExpansion above, this test reads glyphsPreset rather than
// spelling its tokens literally: the property it asserts is that the CLI's own preset and the
// facade's own options agree with each other, so both sides must be read from their real sources
// rather than from a copy that could itself drift. This assertion cannot live in package quarry:
// internal/cli imports quarry, and the reverse import would be a cycle, which is why
// quarry.GlyphsOptions is exported at all — see the overview's glyphs-options-is-exported
// decision.
//
// request.view is asserted separately, as "glyphs", because it is the third field the preset
// fixes and the one with no counterpart in quarry.TOCOptions: the facade returns the projected
// GlyphsAnswer type directly from Repo.Glyphs and has no view field to select, so there is nothing
// on the facade side for view to be compared against.
func TestGlyphsPresetMatchesFacadeOptions(t *testing.T) {
	argv := make([]string, 0, 1+len(glyphsPreset)+1)
	argv = append(argv, "toc")
	argv = append(argv, glyphsPreset...)
	argv = append(argv, "placeholder-target")

	req, err := parseArgs(argv)
	if err != nil {
		t.Fatalf("parseArgs(%v) = _, %v; want nil error", argv, err)
	}

	opts := quarry.GlyphsOptions()

	if req.depth != opts.Depth {
		t.Errorf("req.depth = %d; want %d (quarry.GlyphsOptions().Depth)", req.depth, opts.Depth)
	}

	if req.symbols == nil || opts.Symbols == nil {
		t.Fatalf("symbols = %v, %v; want both non-nil", req.symbols, opts.Symbols)
	}
	if *req.symbols != *opts.Symbols {
		t.Errorf("*req.symbols = %v; want %v (*quarry.GlyphsOptions().Symbols)", *req.symbols, *opts.Symbols)
	}

	if req.view != "glyphs" {
		t.Errorf("req.view = %q; want %q", req.view, "glyphs")
	}
}
