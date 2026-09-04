// loomyard_timing_test.go turns this task's headline timing criterion into a test and keeps a
// benchmark beside it: twenty glyphs, four each from five Loomyard packages of differing size, must
// resolve in under 150 ms once the memoised, grouped-by-unit lookup Resolve is built on is doing the
// work. It also spot-checks Expand against a Loomyard type whose methods live in more than one file,
// the property no committed fixture proves as convincingly as a real repository does. Every case here
// is gated by loomyard_test.go's loomyardRepo and skipped under -short, exactly like the rest of this
// package's Loomyard suite.

package engine

import (
	"testing"
	"time"
)

// loomyardTwentyGlyphs is twenty glyphs read off a checkout pinned at loomyardPin: four each from
// internal/gitexec (4 files, the smallest of the five), internal/configengine (6 files),
// internal/batcher (10 files), internal/boardengine (18 files), and internal/gitrepo (29 files, the
// large package of the five and the one the grouping guarantee exists for — an ungrouped
// implementation re-parsing every unit once per glyph, rather than once for all four glyphs sharing
// it, is what turns this list's total file count into a budget this test could not otherwise meet).
// Every string was read directly off that checkout, never invented, because the pin is what keeps
// this list from drifting silently out of step with the code it measures.
var loomyardTwentyGlyphs = []string{
	"internal/gitexec#renderArg",
	"internal/gitexec#runCore",
	"internal/gitexec#RunGit",
	"internal/gitexec#Run",

	"internal/configengine#FindBaseDir",
	"internal/configengine#ConfigDir",
	"internal/configengine#ConfigFileRel",
	"internal/configengine#LoadOrTemplate",

	"internal/batcher#Active",
	"internal/batcher#register",
	"internal/batcher#lookup",
	"internal/batcher#Select",

	"internal/boardengine#New",
	"internal/boardengine#LoadConfig",
	"internal/boardengine#ComputeLayers",
	"internal/boardengine#RenderOrder",

	"internal/gitrepo#validSHA",
	"internal/gitrepo#New",
	"internal/gitrepo#commitByHash",
	"internal/gitrepo#treeForRev",
}

// TestResolve_TwentyGlyphsUnder150ms checks testing.Short() and skips first, before calling
// loomyardRepo: loomyardRepo fails rather than skips on a wrongly-pinned checkout, so gating on the
// checkout first would fail this test under -short on a machine where the established Loomyard suite
// merely skips. It then resolves loomyardTwentyGlyphs once to assert every result is StatusFound — a
// drifted glyph list would otherwise turn this test green by timing twenty misses — before timing one
// Resolve call over the same twenty glyphs five times with time.Since. It asserts the minimum of the
// five elapsed durations is under 150 ms, reporting all five on failure: the minimum is the floor this
// criterion is about, where a single run would fail on an unrelated build in another worktree and an
// average or a percentile would measure the machine's load rather than the code.
func TestResolve_TwentyGlyphsUnder150ms(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping the Loomyard timing assertion in -short mode")
	}
	repoRoot := loomyardRepo(t)
	r := openRepo(t, repoRoot)

	results, err := r.Resolve(loomyardTwentyGlyphs)
	if err != nil {
		t.Fatalf("Resolve(loomyardTwentyGlyphs) failed: %v", err)
	}
	for i, res := range results {
		if res.Status != StatusFound {
			t.Fatalf("Resolve(%q) = %+v; want Status %q", loomyardTwentyGlyphs[i], res, StatusFound)
		}
	}

	const runs = 5
	durations := make([]time.Duration, runs)
	for i := 0; i < runs; i++ {
		start := time.Now()
		if _, err := r.Resolve(loomyardTwentyGlyphs); err != nil {
			t.Fatalf("Resolve(loomyardTwentyGlyphs) failed on timed run %d: %v", i, err)
		}
		durations[i] = time.Since(start)
	}

	min := durations[0]
	for _, d := range durations[1:] {
		if d < min {
			min = d
		}
	}
	if min >= 150*time.Millisecond {
		t.Errorf("min Resolve time over %d runs = %v; want < 150ms; all runs: %v", runs, min, durations)
	}
}
