// delta_order_test.go covers (*Repo).Delta's total ordering and determinism: TestDelta_Ordering runs
// one batch large enough that Go's randomised map iteration would surface a non-total order, and
// asserts both that repeated calls are byte-identical and that each array obeys its own stated rule.

package engine

import (
	"encoding/json"
	"testing"
)

// indexOfSymbolID returns the index of the first symbol in syms whose ID equals id, or -1.
func indexOfSymbolID(syms []Symbol, id string) int {
	for i, s := range syms {
		if s.ID == id {
			return i
		}
	}
	return -1
}

// indexOfModifiedID returns the index of the first ModifiedSymbol in mods whose ID equals id, or -1.
func indexOfModifiedID(mods []ModifiedSymbol, id string) int {
	for i, m := range mods {
		if m.ID == id {
			return i
		}
	}
	return -1
}

// indexOfRenamedFromID returns the index of the first RenamedPair in pairs whose From.ID equals id,
// or -1.
func indexOfRenamedFromID(pairs []RenamedPair, id string) int {
	for i, p := range pairs {
		if p.From.ID == id {
			return i
		}
	}
	return -1
}

// indexOfCandidateEntryID returns the index of the first RenameCandidateEntry in entries whose ID
// equals id, or -1.
func indexOfCandidateEntryID(entries []RenameCandidateEntry, id string) int {
	for i, e := range entries {
		if e.ID == id {
			return i
		}
	}
	return -1
}

// TestDelta_Ordering asserts every array of a DeltaAnswer is in its stated total order and that
// repeated calls over the identical batch produce a byte-identical marshalled answer. The batch is
// large enough — nine entries, several dozen symbols — that Go's own randomised map iteration would
// surface a non-total order across a handful of runs, and it includes one grouped const spec
// ("Z, Y, X = 3, 2, 1") declaring several names sharing one Start and one End, which is exactly the
// case the id-and-kind tie-break exists for: a file-then-start rule alone cannot separate them.
func TestDelta_Ordering(t *testing.T) {
	entries := []DeltaEntry{
		{Path: "a2.go", After: []byte("package p\n\nfunc Beta() {}\n"), AfterUnit: testUnit},
		{
			Path:      "b.go",
			After:     []byte("package p\n\nconst (\n\tZ, Y, X = 3, 2, 1\n)\n\nfunc Alpha() {}\n"),
			AfterUnit: testUnit,
		},
		{Path: "b2.go", Before: []byte("package p\n\nfunc Epsilon() {}\n"), BeforeUnit: testUnit},
		{
			Path:       "c.go",
			Before:     []byte("package p\n\nfunc Gamma() {}\n\nfunc Delta() {}\n"),
			BeforeUnit: testUnit,
		},
		{
			Path:       "f1.go",
			Before:     []byte("package p\n\n// Old doc.\nfunc Combo() {}\n"),
			BeforeUnit: testUnit,
		},
		{
			Path:      "f2.go",
			After:     []byte("package p\n\n// New doc.\nfunc Combo() {}\n"),
			AfterUnit: testUnit,
		},
		{
			Path:       "g.go",
			Before:     []byte("package p\n\n// Old doc.\nfunc Mod2() {}\n\nfunc Mod1(a int) {}\n"),
			After:      []byte("package p\n\n// New doc.\nfunc Mod2() {}\n\nfunc Mod1(a, b int) {}\n"),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		},
		{
			Path: "h.go",
			Before: []byte("package p\n\nfunc OldB() {\n\tOldB()\n}\n\n" +
				"func OldA() {\n\tOldA()\n\tx := 1\n\t_ = x\n}\n"),
			After: []byte("package p\n\nfunc NewB() {\n\tNewB()\n}\n\n" +
				"func NewA() {\n\tNewA()\n\tx := 1\n\t_ = x\n}\n"),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		},
		{
			Path:       "i.go",
			Before:     []byte("package p\n\nconst CB = 1\n\nconst CA = 1\n"),
			After:      []byte("package p\n\nconst DB = 1\n\nconst DA = 1\n"),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		},
	}

	const repeats = 25
	var first []byte
	var last DeltaAnswer
	for i := 0; i < repeats; i++ {
		ans := mustDelta(t, entries)
		last = ans
		marshalled, err := json.Marshal(ans)
		if err != nil {
			t.Fatalf("json.Marshal(...) returned error: %v", err)
		}
		if i == 0 {
			first = marshalled
			continue
		}
		if string(marshalled) != string(first) {
			t.Fatalf("run %d's marshalled answer differs from run 0's; want byte-identical across repeats", i)
		}
	}

	// files: the input batch's own order, never sorted.
	if len(last.Files) != len(entries) {
		t.Fatalf("len(Files) = %d; want %d", len(last.Files), len(entries))
	}
	for i, e := range entries {
		if last.Files[i].Path != e.Path {
			t.Errorf("Files[%d].Path = %q; want %q — files keeps the input batch's own order", i, last.Files[i].Path, e.Path)
		}
	}

	// created: file ascending, then Start ascending, then id ascending, then kind ascending. The
	// grouped const spec Z, Y, X shares one Start and is separated only by the id tie-break.
	posBeta := indexOfSymbolID(last.Created, uid("Beta"))
	posX := indexOfSymbolID(last.Created, uid("X"))
	posY := indexOfSymbolID(last.Created, uid("Y"))
	posZ := indexOfSymbolID(last.Created, uid("Z"))
	posAlpha := indexOfSymbolID(last.Created, uid("Alpha"))
	posDB := indexOfSymbolID(last.Created, uid("DB"))
	posDA := indexOfSymbolID(last.Created, uid("DA"))
	for _, p := range []int{posBeta, posX, posY, posZ, posAlpha, posDB, posDA} {
		if p < 0 {
			t.Fatalf("Created = %+v; missing an expected symbol", last.Created)
		}
	}
	if !(posBeta < posX && posX < posY && posY < posZ && posZ < posAlpha && posAlpha < posDB && posDB < posDA) {
		t.Errorf("Created order = [Beta:%d X:%d Y:%d Z:%d Alpha:%d DB:%d DA:%d]; want that exact "+
			"ascending order — file, then start (X/Y/Z share one span from their grouped spec, split "+
			"only by id), then id, then kind",
			posBeta, posX, posY, posZ, posAlpha, posDB, posDA)
	}

	// deleted: the same rule as created.
	posEpsilon := indexOfSymbolID(last.Deleted, uid("Epsilon"))
	posGamma := indexOfSymbolID(last.Deleted, uid("Gamma"))
	posDelta := indexOfSymbolID(last.Deleted, uid("Delta"))
	posCB := indexOfSymbolID(last.Deleted, uid("CB"))
	posCA := indexOfSymbolID(last.Deleted, uid("CA"))
	for _, p := range []int{posEpsilon, posGamma, posDelta, posCB, posCA} {
		if p < 0 {
			t.Fatalf("Deleted = %+v; missing an expected symbol", last.Deleted)
		}
	}
	if !(posEpsilon < posGamma && posGamma < posDelta && posDelta < posCB && posCB < posCA) {
		t.Errorf("Deleted order = [Epsilon:%d Gamma:%d Delta:%d CB:%d CA:%d]; want that exact ascending order",
			posEpsilon, posGamma, posDelta, posCB, posCA)
	}

	// modified: id ascending, then kind ascending — the table key itself.
	posCombo := indexOfModifiedID(last.Modified, uid("Combo"))
	posMod1 := indexOfModifiedID(last.Modified, uid("Mod1"))
	posMod2 := indexOfModifiedID(last.Modified, uid("Mod2"))
	if posCombo < 0 || posMod1 < 0 || posMod2 < 0 {
		t.Fatalf("Modified = %+v; missing an expected entry", last.Modified)
	}
	if !(posCombo < posMod1 && posMod1 < posMod2) {
		t.Errorf("Modified order = [Combo:%d Mod1:%d Mod2:%d]; want ascending id order", posCombo, posMod1, posMod2)
	}

	comboEntry := findModified(t, last.Modified, uid("Combo"))
	if !changedEquals(comboEntry, ChangedDoc, ChangedFile) {
		t.Errorf("Combo's Changed = %v; want [doc file] in the closed vocabulary's own declared order, "+
			"not the order the two dimensions might be discovered in", comboEntry.Changed)
	}
	mod1Entry := findModified(t, last.Modified, uid("Mod1"))
	if !changedEquals(mod1Entry, ChangedSignature) {
		t.Errorf("Mod1's Changed = %v; want [signature]", mod1Entry.Changed)
	}
	mod2Entry := findModified(t, last.Modified, uid("Mod2"))
	if !changedEquals(mod2Entry, ChangedDoc) {
		t.Errorf("Mod2's Changed = %v; want [doc]", mod2Entry.Changed)
	}

	// renamed: From.id ascending, then To.id ascending.
	posOldA := indexOfRenamedFromID(last.Renamed, uid("OldA"))
	posOldB := indexOfRenamedFromID(last.Renamed, uid("OldB"))
	if posOldA < 0 || posOldB < 0 {
		t.Fatalf("Renamed = %+v; want both the OldA and OldB pairs asserted", last.Renamed)
	}
	if !(posOldA < posOldB) {
		t.Errorf("Renamed order = [OldA:%d OldB:%d]; want OldA before OldB (from.id ascending)", posOldA, posOldB)
	}

	// rename_candidates: the deleted symbol's id ascending, then its kind ascending.
	posCAEntry := indexOfCandidateEntryID(last.RenameCandidates, uid("CA"))
	posCBEntry := indexOfCandidateEntryID(last.RenameCandidates, uid("CB"))
	if posCAEntry < 0 || posCBEntry < 0 {
		t.Fatalf("RenameCandidates = %+v; want entries for both CA and CB", last.RenameCandidates)
	}
	if !(posCAEntry < posCBEntry) {
		t.Errorf("RenameCandidates order = [CA:%d CB:%d]; want CA before CB (deleted id ascending)", posCAEntry, posCBEntry)
	}
}
