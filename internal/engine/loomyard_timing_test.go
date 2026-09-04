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

// BenchmarkResolveTwentyGlyphs is kept beside TestResolve_TwentyGlyphsUnder150ms so a regression is
// measurable, not merely detectable — go test does not run benchmarks, so a criterion asserted only
// here would never be checked by the verify gate. It calls the same loomyardRepo gate, passing a
// *testing.B, which is the reason card 19 widened that helper's parameter to testing.TB. It opens the
// checkout once outside the timed loop with Open directly — not the openRepo helper, which takes a
// *testing.T a benchmark does not have — and fails on Open's error with b.Fatalf. It then calls
// b.ResetTimer and calls Resolve over loomyardTwentyGlyphs once per iteration, failing the benchmark
// on any error. It reuses the package-level glyph list declared above rather than restating it.
func BenchmarkResolveTwentyGlyphs(b *testing.B) {
	repoRoot := loomyardRepo(b)
	r, err := Open(repoRoot)
	if err != nil {
		b.Fatalf("Open(%q) failed: %v", repoRoot, err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := r.Resolve(loomyardTwentyGlyphs); err != nil {
			b.Fatalf("Resolve(loomyardTwentyGlyphs) failed: %v", err)
		}
	}
}

// TestExpand_LoomyardMembersAcrossFiles checks testing.Short() and skips first, then calls
// loomyardRepo — the same gate order TestResolve_TwentyGlyphsUnder150ms uses and for the same reason
// — and opens the checkout with openRepo. It expands internal/fabricengine#Topology: a type declared
// in topology.go, whose own methods are declared across add.go, checkout.go, cleanup.go, and list.go
// at the pinned commit. That last property is the one no committed fixture proves as convincingly as
// this real repository does — a fixture built to have members in two files proves the filter runs,
// where a real repository proves it finds what a reader would have had to open several files to see.
func TestExpand_LoomyardMembersAcrossFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping the Loomyard expand spot check in -short mode")
	}
	repoRoot := loomyardRepo(t)
	r := openRepo(t, repoRoot)

	got, err := r.Expand("internal/fabricengine#Topology")
	if err != nil {
		t.Fatalf(`Expand("internal/fabricengine#Topology") failed: %v`, err)
	}
	if got.Status != StatusFound {
		t.Fatalf("Expand(...).Status = %q; want %q (full answer: %+v)", got.Status, StatusFound, got)
	}
	if got.Head == nil {
		t.Fatal("Expand(...).Head is nil; want the Topology type's own symbol entry")
	}
	if len(got.Members) == 0 {
		t.Fatal("Expand(...).Members is empty; want Topology's methods")
	}

	files := make(map[string]bool)
	for _, m := range got.Members {
		files[m.File] = true
	}
	if len(files) < 2 {
		t.Errorf("Expand(...).Members span %d distinct file(s): %v; want at least 2", len(files), files)
	}
}
