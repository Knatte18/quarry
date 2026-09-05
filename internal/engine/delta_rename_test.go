// delta_rename_test.go covers (*Repo).Delta's two rename tiers: the exact tier (TestDelta_ExactTier)
// and the evidence tier (TestDelta_EvidenceTier). Every case is in-memory source, so this file needs
// no fixture tree and no repository.

package engine

import (
	"reflect"
	"testing"
)

// findCandidateEntry returns the RenameCandidateEntry in entries whose ID equals id, failing the
// test if none matches.
func findCandidateEntry(t *testing.T, entries []RenameCandidateEntry, id string) RenameCandidateEntry {
	t.Helper()
	for _, e := range entries {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("no rename candidate entry with id %q found in %+v", id, entries)
	return RenameCandidateEntry{}
}

// findCandidate returns the RenameCandidate in cands whose ID equals id, failing the test if none
// matches.
func findCandidate(t *testing.T, cands []RenameCandidate, id string) RenameCandidate {
	t.Helper()
	for _, c := range cands {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("no candidate with id %q found in %+v", id, cands)
	return RenameCandidate{}
}

// renamedContains reports whether pairs holds a RenamedPair whose From.ID equals fromID and To.ID
// equals toID.
func renamedContains(pairs []RenamedPair, fromID, toID string) bool {
	for _, p := range pairs {
		if p.From.ID == fromID && p.To.ID == toID {
			return true
		}
	}
	return false
}

func TestDelta_ExactTier(t *testing.T) {
	t.Run("IdenticalModuloName_RecursiveSelfCall_Asserted", func(t *testing.T) {
		before := "package p\n\nfunc Old() {\n\tOld()\n}\n"
		after := "package p\n\nfunc New() {\n\tNew()\n}\n"
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte(before), After: []byte(after),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		}})
		if !renamedContains(ans.Renamed, uid("Old"), uid("New")) {
			t.Fatalf("Renamed = %+v; want the Old->New pair asserted", ans.Renamed)
		}
		if containsID(ans.Created, uid("New")) || containsID(ans.Deleted, uid("Old")) {
			t.Errorf("Created=%+v Deleted=%+v; want both constituents absent — the pair is the only "+
				"surviving record", ans.Created, ans.Deleted)
		}
		var pair RenamedPair
		for _, p := range ans.Renamed {
			if p.From.ID == uid("Old") {
				pair = p
			}
		}
		if pair.From.Kind != KindFunction || pair.To.Kind != KindFunction {
			t.Errorf("From.Kind=%v To.Kind=%v; want both function", pair.From.Kind, pair.To.Kind)
		}
		if pair.From.File != "a.go" || pair.To.File != "a.go" {
			t.Errorf("From.File=%q To.File=%q; want both a.go", pair.From.File, pair.To.File)
		}
		if pair.From.Start == 0 || pair.To.Start == 0 {
			t.Errorf("From.Start=%d To.Start=%d; want both non-zero spans", pair.From.Start, pair.To.Start)
		}
	})

	t.Run("ExtraStatement_DemotedToCandidate", func(t *testing.T) {
		before := "package p\n\nfunc Old() {\n\tOld()\n}\n"
		after := "package p\n\nfunc New() {\n\tNew()\n\tx := 1\n\t_ = x\n}\n"
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte(before), After: []byte(after),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		}})
		if renamedContains(ans.Renamed, uid("Old"), uid("New")) {
			t.Errorf("Renamed = %+v; want the pair demoted — the after body carries an extra statement", ans.Renamed)
		}
		if !containsID(ans.Created, uid("New")) || !containsID(ans.Deleted, uid("Old")) {
			t.Errorf("Created=%+v Deleted=%+v; want both still present after a demotion", ans.Created, ans.Deleted)
		}
		entry := findCandidateEntry(t, ans.RenameCandidates, uid("Old"))
		findCandidate(t, entry.Candidates, uid("New"))
	})

	t.Run("AnonymousOperatorDiffers_DemotedNeverRenamed", func(t *testing.T) {
		before := "package p\n\nfunc Old() {\n\tx := 1 + 1\n\t_ = x\n}\n"
		after := "package p\n\nfunc New() {\n\tx := 1 - 1\n\t_ = x\n}\n"
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte(before), After: []byte(after),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		}})
		if renamedContains(ans.Renamed, uid("Old"), uid("New")) {
			t.Errorf("Renamed = %+v; want never present — the bodies differ only in an anonymous operator", ans.Renamed)
		}
		entry := findCandidateEntry(t, ans.RenameCandidates, uid("Old"))
		findCandidate(t, entry.Candidates, uid("New"))
	})

	t.Run("TwoDeletedPairExactlyWithOneCreated_AmbiguousNoRenamed", func(t *testing.T) {
		entries := []DeltaEntry{
			{Path: "a.go", Before: []byte("package p\n\nfunc OldOne() {\n\tOldOne()\n}\n"), BeforeUnit: testUnit},
			{Path: "b.go", Before: []byte("package p\n\nfunc OldTwo() {\n\tOldTwo()\n}\n"), BeforeUnit: testUnit},
			{Path: "c.go", After: []byte("package p\n\nfunc New() {\n\tNew()\n}\n"), AfterUnit: testUnit},
		}
		ans := mustDelta(t, entries)
		if len(ans.Renamed) != 0 {
			t.Errorf("Renamed = %+v; want none — New matches both deleted symbols exactly, so neither is chosen", ans.Renamed)
		}
		if !containsID(ans.Created, uid("New")) {
			t.Errorf("Created = %+v; want New present, unresolved", ans.Created)
		}
		entryOne := findCandidateEntry(t, ans.RenameCandidates, uid("OldOne"))
		findCandidate(t, entryOne.Candidates, uid("New"))
		entryTwo := findCandidateEntry(t, ans.RenameCandidates, uid("OldTwo"))
		findCandidate(t, entryTwo.Candidates, uid("New"))
	})

	t.Run("CrossFileWithinOneUnit_Exact", func(t *testing.T) {
		entries := []DeltaEntry{
			{Path: "a.go", Before: []byte("package p\n\nfunc Old() {\n\tOld()\n}\n"), BeforeUnit: testUnit},
			{Path: "b.go", After: []byte("package p\n\nfunc New() {\n\tNew()\n}\n"), AfterUnit: testUnit},
		}
		ans := mustDelta(t, entries)
		if !renamedContains(ans.Renamed, uid("Old"), uid("New")) {
			t.Fatalf("Renamed = %+v; want the cross-file pair asserted", ans.Renamed)
		}
		for _, p := range ans.Renamed {
			if p.From.ID == uid("Old") && (p.From.File != "a.go" || p.To.File != "b.go") {
				t.Errorf("From.File=%q To.File=%q; want a.go and b.go", p.From.File, p.To.File)
			}
		}
	})

	t.Run("CrossUnit_PairedOnNeitherTier", func(t *testing.T) {
		entries := []DeltaEntry{
			{Path: "a.go", Before: []byte("package p\n\nfunc Old() {\n\tOld()\n}\n"), BeforeUnit: testUnit},
			{Path: "other/b.go", After: []byte("package p\n\nfunc New() {\n\tNew()\n}\n"), AfterUnit: "other"},
		}
		ans := mustDelta(t, entries)
		if len(ans.Renamed) != 0 {
			t.Errorf("Renamed = %+v; want none — the two sides belong to different units", ans.Renamed)
		}
		if len(ans.RenameCandidates) != 0 {
			t.Errorf("RenameCandidates = %+v; want none across a unit boundary", ans.RenameCandidates)
		}
		if !containsID(ans.Created, uid2("other", "New")) || !containsID(ans.Deleted, uid("Old")) {
			t.Errorf("Created=%+v Deleted=%+v; want a plain create plus delete", ans.Created, ans.Deleted)
		}
	})

	t.Run("OtherHalfOutsideBatch_PlainDeletedNoCandidate", func(t *testing.T) {
		ans := mustDelta(t, []DeltaEntry{
			{Path: "a.go", Before: []byte("package p\n\nfunc Old() {\n\tOld()\n}\n"), BeforeUnit: testUnit},
		})
		if !containsID(ans.Deleted, uid("Old")) {
			t.Fatalf("Deleted = %+v; want Old present", ans.Deleted)
		}
		for _, e := range ans.RenameCandidates {
			if e.ID == uid("Old") {
				t.Errorf("RenameCandidates contains %q; want none — its other half was never in the batch", uid("Old"))
			}
		}
	})

	t.Run("PartiallyParsedAfterSide_DemotedToCandidate", func(t *testing.T) {
		before := "package p\n\nfunc Old() {\n\tOld()\n}\n"
		after := "package p\n\nfunc New() {\n\tNew()\n}\n\nfunc Broken(\n"
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte(before), After: []byte(after),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		}})
		if !ans.Files[0].LossyAfter {
			t.Fatalf("LossyAfter = false; want true for a deliberately broken trailing declaration")
		}
		if renamedContains(ans.Renamed, uid("Old"), uid("New")) {
			t.Errorf("Renamed = %+v; want the pair demoted — the after side parsed partially", ans.Renamed)
		}
		entry := findCandidateEntry(t, ans.RenameCandidates, uid("Old"))
		findCandidate(t, entry.Candidates, uid("New"))
	})

	t.Run("TwoUnrelatedInterfaces_NotExactRename", func(t *testing.T) {
		before := "package p\n\ntype Reader interface {\n\tRead() int\n}\n"
		after := "package p\n\ntype Closer interface {\n\tClose() error\n}\n"
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte(before), After: []byte(after),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		}})
		if renamedContains(ans.Renamed, uid("Reader"), uid("Closer")) {
			t.Errorf("Renamed = %+v; want the two unrelated interfaces never asserted as a rename", ans.Renamed)
		}
	})
}

// uid2 builds the expected package-level glyph id for name under an arbitrary unit, for the one
// cross-unit test case that needs a second unit string.
func uid2(unit, name string) string {
	return unit + "#" + name
}

func TestDelta_EvidenceTier(t *testing.T) {
	t.Run("BodylessKinds_ReportedNeverAsserted", func(t *testing.T) {
		tests := []struct {
			name      string
			before    string
			after     string
			deletedID string
			createdID string
		}{
			{
				name:      "Const",
				before:    "package p\n\nconst X = 1\n",
				after:     "package p\n\nconst Y = 1\n",
				deletedID: uid("X"),
				createdID: uid("Y"),
			},
			{
				name:      "Var",
				before:    "package p\n\nvar X = 1\n",
				after:     "package p\n\nvar Y = 1\n",
				deletedID: uid("X"),
				createdID: uid("Y"),
			},
			{
				name:      "TypeAlias",
				before:    "package p\n\ntype X = int\n",
				after:     "package p\n\ntype Y = int\n",
				deletedID: uid("X"),
				createdID: uid("Y"),
			},
			{
				name:      "InterfaceMethod",
				before:    "package p\n\ntype I interface {\n\tX() int\n}\n",
				after:     "package p\n\ntype I interface {\n\tY() int\n}\n",
				deletedID: umid("I", "X"),
				createdID: umid("I", "Y"),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ans := mustDelta(t, []DeltaEntry{{
					Path: "a.go", Before: []byte(tt.before), After: []byte(tt.after),
					BeforeUnit: testUnit, AfterUnit: testUnit,
				}})
				if renamedContains(ans.Renamed, tt.deletedID, tt.createdID) {
					t.Errorf("Renamed = %+v; want absent — a bodyless kind is never asserted", ans.Renamed)
				}
				entry := findCandidateEntry(t, ans.RenameCandidates, tt.deletedID)
				cand := findCandidate(t, entry.Candidates, tt.createdID)
				if !cand.Signals.SignatureIdenticalModuloName {
					t.Errorf("SignatureIdenticalModuloName = false; want true")
				}
				if cand.Signals.BodyTokenSimilarity != 1.0 {
					t.Errorf("BodyTokenSimilarity = %v; want 1.0 over two empty streams", cand.Signals.BodyTokenSimilarity)
				}
				if !containsID(ans.Deleted, tt.deletedID) {
					t.Errorf("Deleted = %+v; want %q to remain", ans.Deleted, tt.deletedID)
				}
				if !containsID(ans.Created, tt.createdID) {
					t.Errorf("Created = %+v; want %q to remain", ans.Created, tt.createdID)
				}
			})
		}
	})

	t.Run("ReceiverSharesPrefixWithOldName_TrueSignatureSignal", func(t *testing.T) {
		before := "package p\n\ntype Runner struct{}\n\nfunc (r *Runner) Run() error {\n\treturn nil\n}\n"
		after := "package p\n\ntype Runner struct{}\n\nfunc (r *Runner) Execute() error {\n\tvar err error\n\treturn err\n}\n"
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte(before), After: []byte(after),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		}})
		entry := findCandidateEntry(t, ans.RenameCandidates, umid("Runner", "Run"))
		cand := findCandidate(t, entry.Candidates, umid("Runner", "Execute"))
		if !cand.Signals.SignatureIdenticalModuloName {
			t.Error("SignatureIdenticalModuloName = false; want true — a textual substitution would " +
				"have corrupted the \"Runner\" receiver, but the token-based rule does not")
		}
	})

	t.Run("NoCompositeScoreField", func(t *testing.T) {
		fields := reflect.VisibleFields(reflect.TypeOf(RenameSignals{}))
		want := map[string]bool{
			"SignatureIdenticalModuloName": true,
			"BodyTokenSimilarity":          true,
			"BodyTokensBefore":             true,
			"BodyTokensAfter":              true,
			"DocIdentical":                 true,
		}
		if len(fields) != len(want) {
			t.Fatalf("RenameSignals has %d fields; want exactly %d — no composite score field", len(fields), len(want))
		}
		for _, f := range fields {
			if !want[f.Name] {
				t.Errorf("unexpected RenameSignals field %q; a composite score field must never be added", f.Name)
			}
		}
	})

	t.Run("MultiOccurrenceKey_NeverACandidate", func(t *testing.T) {
		entries := []DeltaEntry{
			{Path: "a.go", Before: []byte("package p\n\nfunc init() {}\n\nfunc init() {}\n"), BeforeUnit: testUnit},
			{Path: "b.go", After: []byte("package p\n\nfunc Foo() {}\n"), AfterUnit: testUnit},
		}
		ans := mustDelta(t, entries)
		count := 0
		for _, s := range ans.Deleted {
			if s.ID == uid("init") {
				count++
			}
		}
		if count != 2 {
			t.Fatalf("Deleted holds %d \"init\" occurrences; want 2", count)
		}
		for _, e := range ans.RenameCandidates {
			if e.ID == uid("init") {
				t.Errorf("RenameCandidates contains %q; want none — a multi-occurrence key is never a candidate", uid("init"))
			}
		}
	})
}
