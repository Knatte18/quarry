// delta_test.go covers (*Repo).Delta's per-entry dispositions and its created, deleted and modified
// arrays. Every case is built from string literals inline in this file: the pure core takes byte
// slices, so no fixture tree and no repository is needed anywhere in this file.

package engine

import (
	"testing"

	"github.com/Knatte18/quarry/glyph"
)

// deltaRepo returns a *Repo suitable for calling Delta on. Delta reads nothing outside its own
// arguments, so a zero-value Repo (no root at all) is a legitimate receiver for every test in this
// file and the next two.
func deltaRepo() *Repo {
	return &Repo{}
}

// mustDelta calls Delta and fails the test immediately on a non-nil error, which Delta's own
// contract says only a failure of the call as a whole — never a single entry's own extraction
// failure — can produce.
func mustDelta(t *testing.T, entries []DeltaEntry) DeltaAnswer {
	t.Helper()
	ans, err := deltaRepo().Delta(entries)
	if err != nil {
		t.Fatalf("Delta(...) returned error: %v", err)
	}
	return ans
}

// uid builds the expected package-level glyph id for name under testUnit.
func uid(name string) string {
	return testUnit + "#" + name
}

// umid builds the expected method or interface-method glyph id for name owned by owner under
// testUnit.
func umid(owner, name string) string {
	g := glyph.Glyph{Lang: glyph.Go, Unit: testUnit, Owner: []string{owner}, Name: name}
	return g.String()
}

// containsID reports whether syms holds a symbol whose ID equals id.
func containsID(syms []Symbol, id string) bool {
	for _, s := range syms {
		if s.ID == id {
			return true
		}
	}
	return false
}

// findModified returns the ModifiedSymbol in mods whose ID equals id, failing the test if none
// matches.
func findModified(t *testing.T, mods []ModifiedSymbol, id string) ModifiedSymbol {
	t.Helper()
	for _, m := range mods {
		if m.ID == id {
			return m
		}
	}
	t.Fatalf("no modified entry with id %q found in %+v", id, mods)
	return ModifiedSymbol{}
}

// hasChanged reports whether m's Changed array names dim.
func hasChanged(m ModifiedSymbol, dim ChangedDimension) bool {
	for _, c := range m.Changed {
		if c == dim {
			return true
		}
	}
	return false
}

// changedEquals reports whether m's Changed array is exactly want, in order.
func changedEquals(m ModifiedSymbol, want ...ChangedDimension) bool {
	if len(m.Changed) != len(want) {
		return false
	}
	for i := range want {
		if m.Changed[i] != want[i] {
			return false
		}
	}
	return true
}

func TestDelta_Dispositions(t *testing.T) {
	t.Run("Added", func(t *testing.T) {
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", After: []byte("package p\n\nfunc F() {}\n"), AfterUnit: testUnit,
		}})
		if len(ans.Files) != 1 || ans.Files[0].Disposition != DispositionAdded {
			t.Fatalf("Files = %+v; want one added entry", ans.Files)
		}
		if !containsID(ans.Created, uid("F")) {
			t.Errorf("Created = %+v; want %q", ans.Created, uid("F"))
		}
	})

	t.Run("Removed", func(t *testing.T) {
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte("package p\n\nfunc F() {}\n"), BeforeUnit: testUnit,
		}})
		if len(ans.Files) != 1 || ans.Files[0].Disposition != DispositionRemoved {
			t.Fatalf("Files = %+v; want one removed entry", ans.Files)
		}
		if !containsID(ans.Deleted, uid("F")) {
			t.Errorf("Deleted = %+v; want %q", ans.Deleted, uid("F"))
		}
	})

	t.Run("Changed", func(t *testing.T) {
		src := "package p\n\nfunc F() {}\n"
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte(src), After: []byte(src),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		}})
		if len(ans.Files) != 1 || ans.Files[0].Disposition != DispositionChanged {
			t.Fatalf("Files = %+v; want one changed entry", ans.Files)
		}
		if len(ans.Created) != 0 || len(ans.Deleted) != 0 || len(ans.Modified) != 0 {
			t.Errorf("identical content reported a difference: created=%v deleted=%v modified=%v",
				ans.Created, ans.Deleted, ans.Modified)
		}
	})

	t.Run("UnsupportedExtension", func(t *testing.T) {
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.txt", After: []byte("hello"),
		}})
		f := ans.Files[0]
		if f.Disposition != DispositionUnsupported || f.Error != "" {
			t.Errorf("Files[0] = %+v; want unsupported with no error", f)
		}
		if len(ans.Created) != 0 {
			t.Errorf("Created = %v; want none for an unsupported extension", ans.Created)
		}
	})

	t.Run("PreSetRefusalSkipsExtraction", func(t *testing.T) {
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Refusal: "unmerged path",
		}})
		f := ans.Files[0]
		if f.Disposition != DispositionError || f.Error != "unmerged path" {
			t.Errorf("Files[0] = %+v; want disposition error with message %q", f, "unmerged path")
		}
	})

	t.Run("InvalidUTF8_OtherEntryStillContributes", func(t *testing.T) {
		ans := mustDelta(t, []DeltaEntry{
			{Path: "bad.go", After: []byte{0xff, 0xfe}, AfterUnit: testUnit},
			{Path: "good.go", After: []byte("package p\n\nfunc G() {}\n"), AfterUnit: testUnit},
		})
		if ans.Files[0].Disposition != DispositionError || ans.Files[0].Error == "" {
			t.Errorf("Files[0] = %+v; want disposition error with a message", ans.Files[0])
		}
		if ans.Files[1].Disposition != DispositionAdded {
			t.Errorf("Files[1] = %+v; want disposition added", ans.Files[1])
		}
		if !containsID(ans.Created, uid("G")) {
			t.Errorf("Created = %+v; want %q despite the other entry's failure", ans.Created, uid("G"))
		}
	})

	t.Run("EmptyBatch", func(t *testing.T) {
		ans := mustDelta(t, nil)
		if len(ans.Files) != 0 || len(ans.Created) != 0 || len(ans.Deleted) != 0 ||
			len(ans.Modified) != 0 || len(ans.Renamed) != 0 || len(ans.RenameCandidates) != 0 {
			t.Errorf("Delta(nil) = %+v; want an entirely empty answer", ans)
		}
	})

	t.Run("BrokenAfterSideStillLossyAndSymbolic", func(t *testing.T) {
		before := "package p\n\nfunc Other() {}\n"
		after := "package p\n\nfunc Broken(\n\nfunc Recovered() {}\n"
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte(before), After: []byte(after),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		}})
		f := ans.Files[0]
		if !f.LossyAfter || f.LossyBefore {
			t.Errorf("LossyBefore=%v LossyAfter=%v; want only LossyAfter set", f.LossyBefore, f.LossyAfter)
		}
		if len(ans.Created) == 0 {
			t.Errorf("Created = %+v; want the surviving symbol(s) the partial parse still recovered", ans.Created)
		}
		if !containsID(ans.Deleted, uid("Other")) {
			t.Errorf("Deleted = %+v; want %q", ans.Deleted, uid("Other"))
		}
	})
}

func TestDelta_CreatedDeletedModified(t *testing.T) {
	t.Run("CreatedOnly", func(t *testing.T) {
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", After: []byte("package p\n\nfunc F() {}\n"), AfterUnit: testUnit,
		}})
		if !containsID(ans.Created, uid("F")) || len(ans.Deleted) != 0 || len(ans.Modified) != 0 {
			t.Errorf("Created=%+v Deleted=%+v Modified=%+v; want F created only", ans.Created, ans.Deleted, ans.Modified)
		}
	})

	t.Run("DeletedOnly", func(t *testing.T) {
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte("package p\n\nfunc F() {}\n"), BeforeUnit: testUnit,
		}})
		if !containsID(ans.Deleted, uid("F")) || len(ans.Created) != 0 || len(ans.Modified) != 0 {
			t.Errorf("Created=%+v Deleted=%+v Modified=%+v; want F deleted only", ans.Created, ans.Deleted, ans.Modified)
		}
	})

	t.Run("ModifiedOnly", func(t *testing.T) {
		before := "package p\n\nfunc F() {\n\tx := 1\n\t_ = x\n}\n"
		after := "package p\n\nfunc F() {\n\tx := 2\n\t_ = x\n}\n"
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte(before), After: []byte(after),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		}})
		if len(ans.Created) != 0 || len(ans.Deleted) != 0 {
			t.Errorf("Created=%+v Deleted=%+v; want neither for a modified-only case", ans.Created, ans.Deleted)
		}
		m := findModified(t, ans.Modified, uid("F"))
		if !changedEquals(m, ChangedBody) {
			t.Errorf("Changed = %v; want [body]", m.Changed)
		}
		if len(m.Before) != 1 || len(m.After) != 1 {
			t.Errorf("Before=%+v After=%+v; want length 1 each for an ordinary symbol", m.Before, m.After)
		}
	})

	t.Run("LineShiftOnly_AbsentEverywhere", func(t *testing.T) {
		before := "package p\n\nfunc F() {\n\treturn\n}\n"
		after := "package p\n\n\nfunc F() {\n\treturn\n}\n"
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte(before), After: []byte(after),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		}})
		if containsID(ans.Created, uid("F")) || containsID(ans.Deleted, uid("F")) {
			t.Errorf("Created=%+v Deleted=%+v; want F absent — it only moved lines", ans.Created, ans.Deleted)
		}
		for _, m := range ans.Modified {
			if m.ID == uid("F") {
				t.Errorf("Modified contains %q; want it absent — a line-only shift is not a change", uid("F"))
			}
		}
	})

	t.Run("ChangedDimensions", func(t *testing.T) {
		t.Run("BodyOnly", func(t *testing.T) {
			before := "package p\n\nfunc F(a int) {\n\tx := 1\n\t_ = x\n}\n"
			after := "package p\n\nfunc F(a int) {\n\tx := 2\n\t_ = x\n}\n"
			ans := mustDelta(t, []DeltaEntry{{
				Path: "a.go", Before: []byte(before), After: []byte(after),
				BeforeUnit: testUnit, AfterUnit: testUnit,
			}})
			m := findModified(t, ans.Modified, uid("F"))
			if !changedEquals(m, ChangedBody) {
				t.Errorf("Changed = %v; want [body]", m.Changed)
			}
		})

		t.Run("SignatureOnly", func(t *testing.T) {
			before := "package p\n\nfunc F(a int) {}\n"
			after := "package p\n\nfunc F(a, b int) {}\n"
			ans := mustDelta(t, []DeltaEntry{{
				Path: "a.go", Before: []byte(before), After: []byte(after),
				BeforeUnit: testUnit, AfterUnit: testUnit,
			}})
			m := findModified(t, ans.Modified, uid("F"))
			if !changedEquals(m, ChangedSignature) {
				t.Errorf("Changed = %v; want [signature]", m.Changed)
			}
		})

		t.Run("DocOnly", func(t *testing.T) {
			before := "package p\n\n// Old doc.\nfunc F() {}\n"
			after := "package p\n\n// New doc.\nfunc F() {}\n"
			ans := mustDelta(t, []DeltaEntry{{
				Path: "a.go", Before: []byte(before), After: []byte(after),
				BeforeUnit: testUnit, AfterUnit: testUnit,
			}})
			m := findModified(t, ans.Modified, uid("F"))
			if !changedEquals(m, ChangedDoc) {
				t.Errorf("Changed = %v; want [doc]", m.Changed)
			}
		})

		t.Run("FileOnly_MovedWithinOneUnit", func(t *testing.T) {
			src := "package p\n\nfunc F() {}\n"
			ans := mustDelta(t, []DeltaEntry{
				{Path: "a.go", Before: []byte(src), BeforeUnit: testUnit},
				{Path: "b.go", After: []byte(src), AfterUnit: testUnit},
			})
			m := findModified(t, ans.Modified, uid("F"))
			if !changedEquals(m, ChangedFile) {
				t.Errorf("Changed = %v; want [file]", m.Changed)
			}
			if len(m.Before) != 1 || m.Before[0].File != "a.go" {
				t.Errorf("Before = %+v; want one location in a.go", m.Before)
			}
			if len(m.After) != 1 || m.After[0].File != "b.go" {
				t.Errorf("After = %+v; want one symbol in b.go", m.After)
			}
		})

		t.Run("Combination_BodyAndSignature", func(t *testing.T) {
			before := "package p\n\nfunc F(a int) {\n\tx := 1\n\t_ = x\n}\n"
			after := "package p\n\nfunc F(a, b int) {\n\tx := 2\n\t_ = x\n}\n"
			ans := mustDelta(t, []DeltaEntry{{
				Path: "a.go", Before: []byte(before), After: []byte(after),
				BeforeUnit: testUnit, AfterUnit: testUnit,
			}})
			m := findModified(t, ans.Modified, uid("F"))
			if !changedEquals(m, ChangedBody, ChangedSignature) {
				t.Errorf("Changed = %v; want [body signature] in that declared order", m.Changed)
			}
		})
	})

	t.Run("ReformattedBody_WhitespaceOnly_NotModified", func(t *testing.T) {
		before := "package p\n\nfunc F() {\n\tx := 1\n\t_ = x\n}\n"
		after := "package p\n\nfunc F() {\n    x := 1\n    _ = x\n}\n"
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte(before), After: []byte(after),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		}})
		for _, m := range ans.Modified {
			if m.ID == uid("F") {
				t.Errorf("Modified contains %q; want it absent — only whitespace changed", uid("F"))
			}
		}
	})

	t.Run("AnonymousTokenRegression", func(t *testing.T) {
		t.Run("IncrementVsDecrement", func(t *testing.T) {
			before := "package p\n\nfunc F() {\n\tx := 0\n\tx++\n\t_ = x\n}\n"
			after := "package p\n\nfunc F() {\n\tx := 0\n\tx--\n\t_ = x\n}\n"
			ans := mustDelta(t, []DeltaEntry{{
				Path: "a.go", Before: []byte(before), After: []byte(after),
				BeforeUnit: testUnit, AfterUnit: testUnit,
			}})
			m := findModified(t, ans.Modified, uid("F"))
			if !hasChanged(m, ChangedBody) {
				t.Errorf("Changed = %v; want body — an anonymous ++/-- token differs", m.Changed)
			}
		})

		t.Run("PlusVsMinus", func(t *testing.T) {
			before := "package p\n\nfunc F() {\n\tx := 1 + 1\n\t_ = x\n}\n"
			after := "package p\n\nfunc F() {\n\tx := 1 - 1\n\t_ = x\n}\n"
			ans := mustDelta(t, []DeltaEntry{{
				Path: "a.go", Before: []byte(before), After: []byte(after),
				BeforeUnit: testUnit, AfterUnit: testUnit,
			}})
			m := findModified(t, ans.Modified, uid("F"))
			if !hasChanged(m, ChangedBody) {
				t.Errorf("Changed = %v; want body — an anonymous +/- token differs", m.Changed)
			}
		})
	})

	t.Run("StructFieldChanged", func(t *testing.T) {
		before := "package p\n\ntype S struct {\n\tX int\n}\n"
		after := "package p\n\ntype S struct {\n\tX string\n}\n"
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte(before), After: []byte(after),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		}})
		m := findModified(t, ans.Modified, uid("S"))
		if !hasChanged(m, ChangedBody) {
			t.Errorf("Changed = %v; want body — a struct field's type changed", m.Changed)
		}
	})

	t.Run("TypeAliasUnderlyingTypeChanged", func(t *testing.T) {
		// A type alias has no body-bearing child at all: BodyStart == DeclEnd on both sides, so the
		// nil-body branch must not collapse this into an empty, always-equal comparison — the change
		// must still surface through the signature dimension.
		before := "package p\n\ntype Alias = int\n"
		after := "package p\n\ntype Alias = string\n"
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte(before), After: []byte(after),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		}})
		m := findModified(t, ans.Modified, uid("Alias"))
		if !changedEquals(m, ChangedSignature) {
			t.Errorf("Changed = %v; want [signature] — the underlying type is part of the signature, "+
				"not a body neither side has", m.Changed)
		}
	})

	t.Run("AddedInterfaceMethod", func(t *testing.T) {
		before := "package p\n\ntype I interface {\n\tA()\n}\n"
		after := "package p\n\ntype I interface {\n\tA()\n\tB()\n}\n"
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte(before), After: []byte(after),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		}})
		if !containsID(ans.Created, umid("I", "B")) {
			t.Errorf("Created = %+v; want the new method %q", ans.Created, umid("I", "B"))
		}
		m := findModified(t, ans.Modified, uid("I"))
		if !hasChanged(m, ChangedBody) {
			t.Errorf("Changed = %v; want the interface type itself modified (body) too", m.Changed)
		}
	})

	t.Run("ConstReplacedByVar_CreateAndDeleteNeverModify", func(t *testing.T) {
		before := "package p\n\nconst X = 1\n"
		after := "package p\n\nvar X = 1\n"
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte(before), After: []byte(after),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		}})
		if !containsID(ans.Created, uid("X")) || !containsID(ans.Deleted, uid("X")) {
			t.Errorf("Created=%+v Deleted=%+v; want X both created (var) and deleted (const)", ans.Created, ans.Deleted)
		}
		for _, m := range ans.Modified {
			if m.ID == uid("X") {
				t.Errorf("Modified contains %q; want a kind change reported as create+delete, never a modification", uid("X"))
			}
		}
	})

	t.Run("TwoInits_DocOnlyDifference", func(t *testing.T) {
		before := "package p\n\nfunc init() {}\n\n// Doc A.\nfunc init() {}\n"
		after := "package p\n\nfunc init() {}\n\n// Doc B.\nfunc init() {}\n"
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte(before), After: []byte(after),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		}})
		m := findModified(t, ans.Modified, uid("init"))
		if !changedEquals(m, ChangedDoc) {
			t.Errorf("Changed = %v; want [doc] only", m.Changed)
		}
		if len(m.Before) != 2 || len(m.After) != 2 {
			t.Errorf("Before=%+v After=%+v; want length 2 each for a two-occurrence key", m.Before, m.After)
		}
	})

	t.Run("TwoInits_SignatureOnlyDifference", func(t *testing.T) {
		before := "package p\n\nfunc init() {}\n\nfunc init() {}\n"
		after := "package p\n\nfunc init() {}\n\nfunc init(a int) {}\n"
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte(before), After: []byte(after),
			BeforeUnit: testUnit, AfterUnit: testUnit,
		}})
		m := findModified(t, ans.Modified, uid("init"))
		if !changedEquals(m, ChangedSignature) {
			t.Errorf("Changed = %v; want [signature] only", m.Changed)
		}
	})

	t.Run("UnspellableUnit_NoSymbols", func(t *testing.T) {
		ans := mustDelta(t, []DeltaEntry{{
			Path: "a.go", Before: []byte("package p\n\nfunc F() {}\n"), BeforeUnit: "bad unit",
		}})
		if len(ans.Files) != 1 || ans.Files[0].Disposition != DispositionRemoved || ans.Files[0].Error != "" {
			t.Errorf("Files[0] = %+v; want disposition removed with no error", ans.Files[0])
		}
		if len(ans.Deleted) != 0 {
			t.Errorf("Deleted = %+v; want none — the supplied unit is unspellable", ans.Deleted)
		}
	})
}
