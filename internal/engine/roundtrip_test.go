// roundtrip_test.go is this task's headline criterion made a test: every declaration Repo.TOC lists
// has a glyph, and Repo.SpansOf's own primitive, symbolsOfUnit, returns exactly the span the walk
// listed for it — zero misses, zero extras. TestRoundTrip_QuarryItself runs this over this
// repository's own tree and needs no environment, so it always runs; TestRoundTrip_Loomyard runs
// the same assertion, factored into assertSymbolRoundTrip so neither case copies the other, over a
// whole Loomyard checkout, gated by loomyard_test.go's environment helper and skipped under
// -short.
//
// harvestWalkSymbols is the harvest half of assertSymbolRoundTrip, extracted so
// TestRoundTrip_LoomyardNaming (naming_roundtrip_test.go) can consume the one walk collector without
// also paying for assertSymbolRoundTrip's per-unit span lookup, which answers a question that test
// does not ask. collectWalkSymbols carries each symbol's signature, kind and enclosing file's lossy
// flag alongside its id and unit, for that same consumer.

package engine

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Knatte18/quarry/glyph"
)

// spanTuple is the (File, Start, SigEnd, End) span identity the round trip compares: what a walk
// answer's FileEntry.Symbols carries for one declaration against what symbolsOfUnit returns for the
// same glyph. File is always the repository-relative, forward-slash path — composed for the walk
// side, since a symbol inside a toc answer leaves its own File field empty on purpose (it already
// sits inside its file entry), and read directly from the lookup side, which fills File itself.
type spanTuple struct {
	File   string
	Start  int
	SigEnd int
	End    int
}

// roundTripSymbol is one glyph's span as the walk listed it, with the glyph unit carried alongside
// so the caller can group by unit before looking anything up. signature and kind are the walked
// symbol's own Signature and Kind, and lossy is the enclosing FileEntry's own Lossy flag — carried
// for TestRoundTrip_LoomyardNaming (naming_roundtrip_test.go), which predicts an id and kind from
// signature and unit alone and uses lossy to assert that no symbol harvested from a partially parsed
// file is present, rather than letting one silently fail its zero-misses check. The two existing
// round trips ignore all three fields.
type roundTripSymbol struct {
	id        string
	unit      string
	tuple     spanTuple
	signature string
	kind      Kind
	lossy     bool
}

// collectWalkSymbols recursively appends every symbol in d's tree to out, composing each one's File
// as the enclosing DirAnswer.Dir joined with its FileEntry.Name via joinRel — the root's "."
// contributing no prefix, exactly as joinRel already does for the walk's own internal use. A
// FileEntry whose Symbols is nil (not requested, or the file's unit is unspellable) contributes
// nothing, matching what toc itself reports for it.
func collectWalkSymbols(d DirAnswer, out *[]roundTripSymbol) {
	for _, fe := range d.Files {
		if fe.Symbols == nil {
			continue
		}
		file := joinRel(d.Dir, fe.Name)
		for _, sym := range *fe.Symbols {
			*out = append(*out, roundTripSymbol{
				id:   sym.ID,
				unit: sym.Glyph.Unit,
				tuple: spanTuple{
					File:   file,
					Start:  sym.Start,
					SigEnd: sym.SigEnd,
					End:    sym.End,
				},
				signature: sym.Signature,
				kind:      sym.Kind,
				lossy:     fe.Lossy,
			})
		}
	}
	for _, child := range d.Dirs {
		collectWalkSymbols(child, out)
	}
}

// harvestWalkSymbols is the harvest half of assertSymbolRoundTrip: it calls TOC on r's root with
// DepthAll and symbols on, fatal on error, runs collectWalkSymbols over the result, and fatals when
// the walk collected zero symbols. It exists so a caller that wants the one walk collector's output
// without assertSymbolRoundTrip's own per-unit span lookup — TestRoundTrip_LoomyardNaming
// (naming_roundtrip_test.go) is the one such caller — has a single place to get it, keeping this
// package to exactly one walk collector rather than two that could drift apart.
func harvestWalkSymbols(t *testing.T, r *Repo) []roundTripSymbol {
	t.Helper()

	root, err := r.TOC(".", TOCOptions{Depth: DepthAll, Symbols: boolPtr(true)})
	if err != nil {
		t.Fatalf(`TOC(".", {Depth: DepthAll, Symbols: true}) failed: %v`, err)
	}

	var walked []roundTripSymbol
	collectWalkSymbols(root, &walked)
	if len(walked) == 0 {
		t.Fatal("collected zero symbols from the walk; the round trip has nothing to check")
	}
	return walked
}

// tupleSetDiff reports the multiset difference between want and got: missing holds every tuple want
// has that got lacks, extra holds every tuple got has that want lacks. Set equality rather than a
// one-to-one positional match is what the round trip needs — several func init() declarations in
// one package are one glyph with several spans, and so are build-tag duplicates, so a one-to-one
// check would be unsatisfiable by construction even for a perfectly correct engine. A plain map
// dedup would be wrong in the other direction, silently collapsing two genuinely distinct
// declarations that happen to share a span; counting occurrences keeps both directions honest.
func tupleSetDiff(want, got []spanTuple) (missing, extra []spanTuple) {
	counts := make(map[spanTuple]int, len(want))
	for _, tp := range want {
		counts[tp]++
	}
	for _, tp := range got {
		if counts[tp] > 0 {
			counts[tp]--
		} else {
			extra = append(extra, tp)
		}
	}
	for tp, n := range counts {
		for i := 0; i < n; i++ {
			missing = append(missing, tp)
		}
	}
	return missing, extra
}

// assertSymbolRoundTrip runs this task's headline criterion against r: it calls TOC on the
// repository root with DepthAll and symbols on, collecting every listed symbol. It then groups the
// listed glyphs by unit and calls symbolsOfUnit once per unit — never once per glyph — handing it a
// fresh ignoreSet built the same way SpansOf builds its own: newIgnoreSet(root) plus one
// extend("."), with symbolsOfUnit owning every step below the root. Grouping by unit is required,
// not an optimisation: a per-glyph lookup re-parses the whole unit directory for every glyph in it,
// nothing is cached (see the "nothing is cached" Shared Decision), and a whole-repository check done
// that way would cost one parse pass per glyph rather than one per unit — a difference a small
// repository's own size keeps affordable but which a Loomyard-scale run would land inside minutes
// and outside go test's default timeout.
//
// Per glyph — that is, per distinct id within a unit — it then asserts that the set of
// (File, Start, SigEnd, End) tuples symbolsOfUnit returned for that id equals the set toc listed for
// it: zero misses, zero extras. It returns every symbol the walk collected, so a caller that needs
// to check something further about each one (TestRoundTrip_Loomyard's glyph.Parse round trip) does
// not have to re-walk the tree to get it.
func assertSymbolRoundTrip(t *testing.T, r *Repo) []roundTripSymbol {
	t.Helper()

	walked := harvestWalkSymbols(t, r)

	byUnit := make(map[string][]roundTripSymbol)
	for _, sym := range walked {
		byUnit[sym.unit] = append(byUnit[sym.unit], sym)
	}

	for unit, walkSyms := range byUnit {
		ig := newIgnoreSet(r.root)
		if _, err := ig.extend("."); err != nil {
			t.Fatalf("read .gitignore for %q: %v", ".", err)
		}
		lookupSyms, err := r.symbolsOfUnit(unit, ig)
		if err != nil {
			t.Fatalf("symbolsOfUnit(%q) failed: %v", unit, err)
		}

		wantByID := make(map[string][]spanTuple)
		for _, sym := range walkSyms {
			wantByID[sym.id] = append(wantByID[sym.id], sym.tuple)
		}
		gotByID := make(map[string][]spanTuple)
		for _, sym := range lookupSyms {
			gotByID[sym.ID] = append(gotByID[sym.ID], spanTuple{
				File: sym.File, Start: sym.Start, SigEnd: sym.SigEnd, End: sym.End,
			})
		}

		ids := make(map[string]bool, len(wantByID)+len(gotByID))
		for id := range wantByID {
			ids[id] = true
		}
		for id := range gotByID {
			ids[id] = true
		}
		for id := range ids {
			missing, extra := tupleSetDiff(wantByID[id], gotByID[id])
			if len(missing) > 0 || len(extra) > 0 {
				t.Errorf("glyph %q (unit %q): symbolsOfUnit spans mismatch; missing=%v extra=%v", id, unit, missing, extra)
			}
		}
	}

	return walked
}

// TestRoundTrip_QuarryItself is this task's headline criterion, run over this repository, with no
// environment needed so it always runs on every machine and in CI. It resolves the module root from
// runtime.Caller(0) and opens it before handing off to assertSymbolRoundTrip.
func TestRoundTrip_QuarryItself(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine this test file's own source location")
	}
	// This file sits at internal/engine/roundtrip_test.go, so the module root is two directories up.
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	r, err := Open(moduleRoot)
	if err != nil {
		t.Fatalf("Open(%q) failed: %v", moduleRoot, err)
	}

	assertSymbolRoundTrip(t, r)
}

// TestRoundTrip_Loomyard runs the same assertion over a whole Loomyard checkout, gated by
// loomyardRepo and skipped under -short — a whole-repository, no-cache walk-then-lookup pass is
// exactly the kind of test -short exists to let a developer skip.
//
// It additionally asserts, for every symbol the walk collected, that the id round-trips through
// glyph.Parse and back through String unchanged. This is what "every declaration toc lists has a
// glyph" means operationally: it is the one assertion in this suite that would catch an id the
// grammar itself would reject, which a span comparison alone cannot, since a rejected id would never
// reach SpansOf's own caller in the first place.
func TestRoundTrip_Loomyard(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping the Loomyard round trip in -short mode")
	}
	repoRoot := loomyardRepo(t)
	r := openRepo(t, repoRoot)

	walked := assertSymbolRoundTrip(t, r)

	for _, sym := range walked {
		parsed, err := glyph.Parse(glyph.Go, sym.id)
		if err != nil {
			t.Errorf("glyph.Parse(Go, %q) failed: %v", sym.id, err)
			continue
		}
		if got := parsed.String(); got != sym.id {
			t.Errorf("glyph.Parse(Go, %q).String() = %q; want %q", sym.id, got, sym.id)
		}
	}
}
