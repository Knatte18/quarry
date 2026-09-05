// delta_golden_test.go pins the seven required delta cases — created, deleted, modified, an
// exact-tier rename, an evidence-tier rename, a mixed batch exercising several dispositions and
// both tiers at once, and a per-entry extraction failure inside an otherwise good batch — against
// their committed goldens under testdata/delta/, one ".json" and one ".txt" file per case.
//
// Every case builds its entries from string literals in this file, feeds them to the pure Delta
// method (never DeltaGit: these are in-memory byte pairs, not a git checkout), wraps the answer
// with two fixed revision strings, and renders both output views. Under this package's own
// "-update" flag (declared here, once, following the three existing precedents in this repository —
// internal/engine/loomyard_test.go, internal/cli/loomyard_test.go and
// internal/mcpserver/toc_golden_test.go — each already declares its own flag.Bool of the same name
// because each package's tests build their own binary, so a same-named flag in this package is not
// a conflict) each case rewrites its committed golden from the current run instead of comparing
// against it.
//
// These fourteen files can only be produced this way, by running "go test ./quarry/ -run
// TestDeltaGolden -update" against this file's own entries. A hand-written golden pins bytes
// nothing produced and passes forever, which is exactly the failure a golden exists to prevent.
// TestDeltaGolden's own name and that regeneration command are load-bearing on each other: a
// differently named test function would make that -update run match nothing and silently produce
// no files at all.

package quarry

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/Knatte18/quarry/internal/engine"
)

// updateGoldens is "-update", checked by compareDeltaGolden to decide whether to compare the
// current run's output against the committed golden or to rewrite it from that output. Declared
// here, once, so no other file in this package needs its own flag.Bool for the same name.
var updateGoldens = flag.Bool("update", false, "regenerate this package's delta goldens under testdata/delta from the current run of TestDeltaGolden")

// deltaGoldenFrom and deltaGoldenTo are the two fixed revision strings every golden case wraps its
// DeltaAnswer with. Their exact values carry no meaning beyond being present and distinct: no case
// in this file drives DeltaGit, so no revision here ever resolves against a real repository.
const deltaGoldenFrom = "rev1"

var deltaGoldenToValue = "rev2"

// deltaGoldenCase is one of the seven required cases: a golden file's base name, and the batch of
// entries the pure Delta method compares.
type deltaGoldenCase struct {
	// name is the golden file's base name, without extension; the case is compared against
	// testdata/delta/<name>.json and testdata/delta/<name>.txt.
	name string
	// entries is the batch fed to the pure Delta method, built entirely from string literals.
	entries []DeltaEntry
}

// deltaGoldenCases is the seven required cases, in the order the task lists them.
var deltaGoldenCases = []deltaGoldenCase{
	{
		name: "created",
		entries: []DeltaEntry{
			{Path: "pkg/new.go", After: []byte("package pkg\n\nfunc New() {}\n"), AfterUnit: "pkg"},
		},
	},
	{
		name: "deleted",
		entries: []DeltaEntry{
			{Path: "pkg/old.go", Before: []byte("package pkg\n\nfunc Old() {}\n"), BeforeUnit: "pkg"},
		},
	},
	{
		name: "modified",
		entries: []DeltaEntry{
			{
				Path:       "pkg/a.go",
				Before:     []byte("package pkg\n\nfunc F() {\n\tx := 1\n\t_ = x\n}\n"),
				After:      []byte("package pkg\n\nfunc F() {\n\tx := 2\n\t_ = x\n}\n"),
				BeforeUnit: "pkg",
				AfterUnit:  "pkg",
			},
		},
	},
	{
		// The exact-tier pair spans two files in one unit, each recursively self-calling under its
		// own name, so the body and signature streams agree modulo the renamed identifier and
		// nothing else — the unique-match-on-both-sides condition the exact tier requires.
		name: "rename-exact",
		entries: []DeltaEntry{
			{Path: "pkg/a.go", Before: []byte("package pkg\n\nfunc Old() {\n\tOld()\n}\n"), BeforeUnit: "pkg"},
			{Path: "pkg/b.go", After: []byte("package pkg\n\nfunc New() {\n\tNew()\n}\n"), AfterUnit: "pkg"},
		},
	},
	{
		// The evidence-tier pair's after side carries one extra statement its before side lacks, so
		// the body streams no longer agree modulo the renamed identifier and the pair is demoted from
		// the exact tier to a reported candidate rather than an asserted rename.
		name: "rename-evidence",
		entries: []DeltaEntry{
			{
				Path:       "pkg/a.go",
				Before:     []byte("package pkg\n\nfunc Old() {\n\tOld()\n}\n"),
				After:      []byte("package pkg\n\nfunc New() {\n\tNew()\n\tx := 1\n\t_ = x\n}\n"),
				BeforeUnit: "pkg",
				AfterUnit:  "pkg",
			},
		},
	},
	{
		// The mixed case exercises several dispositions and both rename tiers at once. added.go and
		// removed.go deliberately differ in Kind (function versus var), which fails the structural
		// condition every rename tier's first three conditions share, keeping this pair a plain
		// create and a plain delete rather than a spurious third rename candidate; the other two
		// pairs are each isolated in their own unit for the same reason.
		name: "mixed",
		entries: []DeltaEntry{
			{Path: "pkg/added.go", After: []byte("package pkg\n\nfunc Added() {}\n"), AfterUnit: "pkg"},
			{Path: "pkg/removed.go", Before: []byte("package pkg\n\nvar Removed = 1\n"), BeforeUnit: "pkg"},
			{
				Path:       "pkg/modified.go",
				Before:     []byte("package pkg\n\nfunc Mod() {\n\tx := 1\n\t_ = x\n}\n"),
				After:      []byte("package pkg\n\nfunc Mod() {\n\tx := 2\n\t_ = x\n}\n"),
				BeforeUnit: "pkg",
				AfterUnit:  "pkg",
			},
			{
				// Shifted's only change between the two sides is the line its declaration starts on;
				// it must be absent from every list in this golden -- created, deleted and modified
				// alike -- which is what pins the ordering rule's line-shift exclusion as bytes.
				Path:       "pkg/lineshift.go",
				Before:     []byte("package pkg\n\nfunc Shifted() {\n\treturn\n}\n"),
				After:      []byte("package pkg\n\n\nfunc Shifted() {\n\treturn\n}\n"),
				BeforeUnit: "pkg",
				AfterUnit:  "pkg",
			},
			{Path: "renameexact/a.go", Before: []byte("package renameexact\n\nfunc OldExact() {\n\tOldExact()\n}\n"), BeforeUnit: "renameexact"},
			{Path: "renameexact/b.go", After: []byte("package renameexact\n\nfunc NewExact() {\n\tNewExact()\n}\n"), AfterUnit: "renameexact"},
			{
				// The after side's trailing "func Broken(" never closes, so this entry's parse is
				// partial and LossyAfter is set -- the one file echo in this golden carrying a lossy
				// flag -- and the rename this entry would otherwise be an exact-tier candidate for is
				// demoted to the evidence tier by that same partial parse.
				Path:       "renameevidence/evidence.go",
				Before:     []byte("package renameevidence\n\nfunc OldEvidence() {\n\tOldEvidence()\n}\n"),
				After:      []byte("package renameevidence\n\nfunc NewEvidence() {\n\tNewEvidence()\n}\n\nfunc Broken(\n"),
				BeforeUnit: "renameevidence",
				AfterUnit:  "renameevidence",
			},
		},
	},
	{
		// entry-error exercises a per-entry extraction failure inside an otherwise good batch: the
		// invalid-UTF-8 entry fails extraction on its own, and the second entry's symbol still
		// contributes, exactly as the task requires -- no other entry's failure ever fails the whole
		// batch.
		name: "entry-error",
		entries: []DeltaEntry{
			{Path: "pkg/bad.go", After: []byte{0xff, 0xfe}, AfterUnit: "pkg"},
			{Path: "pkg/good.go", After: []byte("package pkg\n\nfunc Good() {}\n"), AfterUnit: "pkg"},
		},
	},
}

// mustDeltaGolden calls the pure Delta method on entries, failing the test immediately on a non-nil
// error, which Delta's own contract says only a failure of the call as a whole -- never a single
// entry's own extraction failure -- can produce. The receiver is a zero-value *engine.Repo, exactly
// as the engine's own delta tests use: Delta reads nothing outside its arguments.
func mustDeltaGolden(t *testing.T, entries []DeltaEntry) DeltaAnswer {
	t.Helper()
	r := &Repo{engine: &engine.Repo{}}
	ans, err := r.Delta(entries)
	if err != nil {
		t.Fatalf("Delta(...) returned error: %v", err)
	}
	return ans
}

// TestDeltaGolden runs every case in deltaGoldenCases through the pure Delta method, renders both
// output views, and compares each against its committed golden -- or rewrites both under
// "-update". See this file's own header comment for why this test's name and the regeneration
// command "go test ./quarry/ -run TestDeltaGolden -update" are load-bearing on each other.
func TestDeltaGolden(t *testing.T) {
	for _, tc := range deltaGoldenCases {
		t.Run(tc.name, func(t *testing.T) {
			deltaAns := mustDeltaGolden(t, tc.entries)
			gitAns := GitDeltaAnswer{From: deltaGoldenFrom, To: &deltaGoldenToValue, DeltaAnswer: deltaAns}

			jsonGot, err := RenderDeltaJSON(gitAns)
			if err != nil {
				t.Fatalf("RenderDeltaJSON() error = %v", err)
			}
			compareDeltaGolden(t, tc.name+".json", jsonGot)

			textGot := RenderDeltaText(gitAns)
			compareDeltaGolden(t, tc.name+".txt", []byte(textGot))
		})
	}
}

// compareDeltaGolden compares got byte for byte against the committed
// testdata/delta/name, or — under "-update" — rewrites that golden from got.
func compareDeltaGolden(t *testing.T, name string, got []byte) {
	t.Helper()

	path := filepath.Join("testdata", "delta", name)
	if *updateGoldens {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write golden %q: %v", path, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %q: %v", path, err)
	}
	if !bytes.Equal(want, got) {
		t.Errorf("golden %q mismatch (-want +got):\n--- want ---\n%s\n--- got ---\n%s", name, want, got)
	}
}
