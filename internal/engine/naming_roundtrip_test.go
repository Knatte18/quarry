// naming_roundtrip_test.go is the prediction-equals-extraction round trip: for every declaration
// harvested from the pinned Loomyard checkout, Name — the glyph maker — must predict, from the
// declaration's own unit and signature alone, the exact id and kind the walk already extracted for
// it. TestRoundTrip_LoomyardNaming is gated by loomyardRepo and skipped under -short like every
// other whole-repository pass in this package; it grows no environment gate of its own, since
// loomyardRepo already owns the skip-versus-fail asymmetry and the pin check.
//
// Runtime budget: the maker adds one parse per harvested symbol, two for a head that takes the
// completion retry, but each parses a three-line synthetic file rather than a real source file, so
// the cost is linear in the symbol count with a small constant against the existing helper's
// per-unit whole-file passes (see walk.go's own cost note). If this test's run time nonetheless
// lands near go test's default timeout, the mitigation is to raise the timeout for this one test —
// sampling the harvest is not an option, since it would silently weaken the zero-misses criterion
// this test exists to assert.

package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// namingCounts is the small local shape TestRoundTrip_LoomyardNaming pins against a counts golden,
// through compareGolden's widened any parameter: the harvest total and the two partitioned counts,
// as a vacuity guard against a round trip that would otherwise pass by asserting nothing.
type namingCounts struct {
	Harvest    int `json:"harvest"`
	InContract int `json:"in_contract"`
	Excluded   int `json:"excluded"`
}

// namingSpecKey is the partition key for the multi-name-spec exclusion: unit, file, start, end and
// signature. A spec's several names — "const a, b = 1, 2" — produce symbols agreeing on all five, so
// two or more harvested symbols sharing this key is exactly "these came from one spec". The file is
// part of the key deliberately: build-tag twins in one unit can share a signature and a line span
// across two different files, and excluding those would assert an error the maker will never
// produce, since each twin's fragment names exactly one symbol in its own file and answers normally.
type namingSpecKey struct {
	unit      string
	file      string
	start     int
	end       int
	signature string
}

// namingSpecKeyOf builds sym's namingSpecKey from its walked unit, tuple and signature.
func namingSpecKeyOf(sym roundTripSymbol) namingSpecKey {
	return namingSpecKey{
		unit:      sym.unit,
		file:      sym.tuple.File,
		start:     sym.tuple.Start,
		end:       sym.tuple.End,
		signature: sym.signature,
	}
}

// namingIsInterfaceMethod reports whether sym is an interface method element rather than a method
// declaration: its kind is KindMethod and its signature does not begin with the "func" keyword. A
// populated interface *type* is not excluded by this rule — its signature is head-only, cut at the
// body, so it goes through the maker's completion retry and answers normally.
func namingIsInterfaceMethod(sym roundTripSymbol) bool {
	return sym.kind == KindMethod && !strings.HasPrefix(sym.signature, "func")
}

// TestRoundTrip_LoomyardNaming asserts that Name predicts, from unit and signature alone, the exact
// id and kind the walk already extracted for every in-contract symbol in the pinned checkout, and
// returns a per-entry error and no id for every symbol the two declared non-goals exclude — the
// multi-name-spec exclusion and the interface-method exclusion.
func TestRoundTrip_LoomyardNaming(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping the Loomyard naming round trip in -short mode")
	}
	repoRoot := loomyardRepo(t)
	r := openRepo(t, repoRoot)

	harvest := harvestWalkSymbols(t, r)

	// A symbol harvested from a file the walk marked lossy is neither in-contract nor excluded: the
	// checkout is pinned at a commit whose sources compile, so a lossy file there is itself the
	// finding, and recording it as one here is better than letting it quietly land in-contract and
	// fail the zero-misses check below for a reason that has nothing to do with the maker.
	var lossyFiles []string
	seenLossy := make(map[string]bool)
	for _, sym := range harvest {
		if sym.lossy && !seenLossy[sym.tuple.File] {
			seenLossy[sym.tuple.File] = true
			lossyFiles = append(lossyFiles, sym.tuple.File)
		}
	}
	if len(lossyFiles) > 0 {
		t.Fatalf("harvest carries symbols from %d lossy file(s), which is a checkout premise violation, not a maker finding: %v", len(lossyFiles), lossyFiles)
	}

	specGroups := make(map[namingSpecKey][]roundTripSymbol, len(harvest))
	for _, sym := range harvest {
		specGroups[namingSpecKeyOf(sym)] = append(specGroups[namingSpecKeyOf(sym)], sym)
	}

	var inContract, excluded []roundTripSymbol
	for _, sym := range harvest {
		if len(specGroups[namingSpecKeyOf(sym)]) > 1 || namingIsInterfaceMethod(sym) {
			excluded = append(excluded, sym)
			continue
		}
		inContract = append(inContract, sym)
	}

	// Structural floor: a partition bug that sweeps everything into one side fails immediately here,
	// without waiting on the pinned counts below.
	if len(inContract) == 0 {
		t.Fatal("partition put zero symbols in-contract; the round trip has nothing to check")
	}
	if len(excluded) >= len(harvest) {
		t.Fatalf("partition excluded all %d harvested symbol(s); the round trip has nothing left to check", len(harvest))
	}

	// Batched, not one call per symbol: the facade takes a slice, and the whole point of the batch
	// shape is that a caller hands it a whole harvest at once.
	inContractDecls := make([]Declaration, len(inContract))
	for i, sym := range inContract {
		inContractDecls[i] = Declaration{Unit: sym.unit, Decl: sym.signature}
	}
	inContractResults := Name(inContractDecls)
	for i, sym := range inContract {
		got := inContractResults[i]
		if got.Error != "" {
			t.Errorf("Name(unit=%q, decl=%q) failed: reason=%q error=%q; want id=%q kind=%q",
				sym.unit, sym.signature, got.Reason, got.Error, sym.id, sym.kind)
			continue
		}
		if got.ID != sym.id {
			t.Errorf("Name(unit=%q, decl=%q).ID = %q; want %q", sym.unit, sym.signature, got.ID, sym.id)
		}
		if got.Kind != sym.kind {
			t.Errorf("Name(unit=%q, decl=%q).Kind = %q; want %q", sym.unit, sym.signature, got.Kind, sym.kind)
		}
	}

	excludedDecls := make([]Declaration, len(excluded))
	for i, sym := range excluded {
		excludedDecls[i] = Declaration{Unit: sym.unit, Decl: sym.signature}
	}
	excludedResults := Name(excludedDecls)
	for i, sym := range excluded {
		got := excludedResults[i]
		if got.Error == "" || got.ID != "" {
			t.Errorf("Name(unit=%q, decl=%q) = %+v; want a per-entry error and no id, since this symbol is excluded from the maker's contract",
				sym.unit, sym.signature, got)
		}
	}

	// The counts golden is not committed by this plan: this machine has no Loomyard checkout, so
	// every Loomyard-gated test here skips and the file cannot be produced here. compareGolden
	// fatals on the bare read error itself and card 15 changes only its parameter type, so this test
	// owns the missing-golden message. Gating on the update flag keeps the regeneration run itself
	// from tripping this very check.
	goldenPath := filepath.Join("testdata", "loomyard", "naming-counts.json")
	if !*updateGoldens {
		if _, err := os.Stat(goldenPath); err != nil {
			t.Fatalf("counts golden %q is missing; regenerate it on a machine with the pinned checkout: "+
				"LADDER_LOOMYARD_REPO=<checkout pinned at %s> go test ./internal/engine/ -run TestRoundTrip_LoomyardNaming -update",
				goldenPath, loomyardPin)
		}
	}

	compareGolden(t, "naming-counts.json", namingCounts{
		Harvest:    len(harvest),
		InContract: len(inContract),
		Excluded:   len(excluded),
	})
}
