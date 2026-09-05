// name_test.go covers Name directly: every accepted declaration kind, the completion retry, every
// failure reason the maker itself produces, propagated glyph.Reason words for a bad unit, the
// reason vocabulary's completeness, and the batch's positional semantics. No fixture on disk, no
// repository — Name reads nothing but its own argument.

package engine

import (
	"testing"

	"github.com/Knatte18/quarry/glyph"
)

// nameUnit is the fixed glyph unit every fixture in this file is named under, unless a test is
// specifically exercising unit independence.
const nameUnit = "u"

// nameID builds the expected package-level glyph id under nameUnit.
func nameID(name string) string {
	g := glyph.Glyph{Lang: glyph.Go, Unit: nameUnit, Name: name}
	return g.String()
}

// nameMethodID builds the expected method glyph id under nameUnit.
func nameMethodID(owner, name string) string {
	g := glyph.Glyph{Lang: glyph.Go, Unit: nameUnit, Owner: []string{owner}, Name: name}
	return g.String()
}

// TestName_AcceptedKinds covers one case per declaration kind the maker accepts, each asserting
// the produced id and kind.
func TestName_AcceptedKinds(t *testing.T) {
	tests := []struct {
		name     string
		decl     string
		wantID   string
		wantKind Kind
	}{
		{"FreeFunction", "func F() {}", nameID("F"), KindFunction},
		{"Method", "func (t T) M() {}", nameMethodID("T", "M"), KindMethod},
		{"StructType", "type S struct { F int }", nameID("S"), KindType},
		{"InterfaceType", "type I interface{}", nameID("I"), KindType},
		{"TypeAlias", "type A = int", nameID("A"), KindType},
		{"NamedType", "type N int", nameID("N"), KindType},
		{"UngroupedConst", "const X = 1", nameID("X"), KindConst},
		{"UngroupedVar", "var Y = 1", nameID("Y"), KindVar},
		{"GroupedConstMember", "const (\n\tX = 1\n)", nameID("X"), KindConst},
		{"GroupedVarMember", "var (\n\tY = 1\n)", nameID("Y"), KindVar},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Name([]Declaration{{Unit: nameUnit, Decl: tt.decl}})
			if len(got) != 1 {
				t.Fatalf("Name(%q) returned %d results; want 1", tt.decl, len(got))
			}
			if got[0].Error != "" {
				t.Fatalf("Name(%q)[0].Error = %q; want no error", tt.decl, got[0].Error)
			}
			if got[0].ID != tt.wantID {
				t.Errorf("Name(%q)[0].ID = %q; want %q", tt.decl, got[0].ID, tt.wantID)
			}
			if got[0].Kind != tt.wantKind {
				t.Errorf("Name(%q)[0].Kind = %q; want %q", tt.decl, got[0].Kind, tt.wantKind)
			}
		})
	}
}

// TestName_IotaContinuation covers a bare iota-continuation const member — "const B", the shape
// goGroupedConstOrVarSymbols emits for a continuation spec in an iota block — as its own case: an
// iota enum is the most common grouped shape in the pinned Loomyard checkout, and a regression here
// must be attributable to this one named test rather than to a whole-repository walk.
func TestName_IotaContinuation(t *testing.T) {
	got := Name([]Declaration{{Unit: nameUnit, Decl: "const B"}})
	if len(got) != 1 {
		t.Fatalf("Name(\"const B\") returned %d results; want 1", len(got))
	}
	if got[0].Error != "" {
		t.Fatalf("Name(\"const B\")[0].Error = %q; want no error", got[0].Error)
	}
	if want := nameID("B"); got[0].ID != want {
		t.Errorf("Name(\"const B\")[0].ID = %q; want %q", got[0].ID, want)
	}
	if got[0].Kind != KindConst {
		t.Errorf("Name(\"const B\")[0].Kind = %q; want %q", got[0].Kind, KindConst)
	}
}

// TestName_UnitIndependence proves the synthetic package clause is inert: a unit unrelated to it
// produces a glyph carrying that unit, never "q".
func TestName_UnitIndependence(t *testing.T) {
	const unit = "internal/reedengine"
	got := Name([]Declaration{{Unit: unit, Decl: "func F() {}"}})
	if len(got) != 1 {
		t.Fatalf("Name(...) returned %d results; want 1", len(got))
	}
	want := glyph.Glyph{Lang: glyph.Go, Unit: unit, Name: "F"}.String()
	if got[0].ID != want {
		t.Errorf("Name(...)[0].ID = %q; want %q", got[0].ID, want)
	}
	if got[0].Unit != unit {
		t.Errorf("Name(...)[0].Unit = %q; want %q", got[0].Unit, unit)
	}
}

// TestName_NonexistentReceiver pins that the maker never type-checks: a method on a type declared
// nowhere still answers normally, since tree-sitter parses syntax only.
func TestName_NonexistentReceiver(t *testing.T) {
	const decl = "func (f *Focus) Reset() error"
	got := Name([]Declaration{{Unit: nameUnit, Decl: decl}})
	if len(got) != 1 {
		t.Fatalf("Name(%q) returned %d results; want 1", decl, len(got))
	}
	if got[0].Error != "" {
		t.Fatalf("Name(%q)[0].Error = %q; want no error", decl, got[0].Error)
	}
	if want := nameMethodID("Focus", "Reset"); got[0].ID != want {
		t.Errorf("Name(%q)[0].ID = %q; want %q", decl, got[0].ID, want)
	}
	if got[0].Kind != KindMethod {
		t.Errorf("Name(%q)[0].Kind = %q; want %q", decl, got[0].Kind, KindMethod)
	}
}

// TestName_CompletionRetry covers the retry-on-partial-parse path: a struct or interface head with
// no body answers with the same id its empty-bodied form produces, and a populated struct agrees
// with its own head form too — pinning that the asymmetry with a populated interface (see
// TestName_PopulatedInterfaceRejected) is a property of the interface shape, not of every type
// completed this way.
func TestName_CompletionRetry(t *testing.T) {
	tests := []struct {
		name string
		head string
		full string
	}{
		{"Struct", "type T struct", "type T struct {}"},
		{"Interface", "type T interface", "type T interface {}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			head := Name([]Declaration{{Unit: nameUnit, Decl: tt.head}})[0]
			full := Name([]Declaration{{Unit: nameUnit, Decl: tt.full}})[0]
			if head.Error != "" {
				t.Fatalf("Name(%q)[0].Error = %q; want no error", tt.head, head.Error)
			}
			if full.Error != "" {
				t.Fatalf("Name(%q)[0].Error = %q; want no error", tt.full, full.Error)
			}
			if head.ID != full.ID {
				t.Errorf("Name(%q)[0].ID = %q; Name(%q)[0].ID = %q; want equal", tt.head, head.ID, tt.full, full.ID)
			}
			if head.Kind != full.Kind {
				t.Errorf("Name(%q)[0].Kind = %q; Name(%q)[0].Kind = %q; want equal", tt.head, head.Kind, tt.full, full.Kind)
			}
		})
	}

	t.Run("PopulatedStructAgreesWithHead", func(t *testing.T) {
		head := Name([]Declaration{{Unit: nameUnit, Decl: "type S struct"}})[0]
		populated := Name([]Declaration{{Unit: nameUnit, Decl: "type S struct { F int }"}})[0]
		if head.Error != "" {
			t.Fatalf("Name(\"type S struct\")[0].Error = %q; want no error", head.Error)
		}
		if populated.Error != "" {
			t.Fatalf("Name(\"type S struct { F int }\")[0].Error = %q; want no error", populated.Error)
		}
		if head.ID != populated.ID {
			t.Errorf("head ID = %q; populated ID = %q; want equal", head.ID, populated.ID)
		}
		if head.Kind != populated.Kind {
			t.Errorf("head Kind = %q; populated Kind = %q; want equal", head.Kind, populated.Kind)
		}
	})
}

// TestName_PopulatedInterfaceRejected covers a populated interface: its own type symbol plus its
// method symbol are two, so it is rejected as several_declarations even though a reader of the
// accepted-forms list would expect it to work.
func TestName_PopulatedInterfaceRejected(t *testing.T) {
	const decl = "type R interface { Read() error }"
	got := Name([]Declaration{{Unit: nameUnit, Decl: decl}})[0]
	if got.Reason != NameReasonSeveralDeclarations {
		t.Errorf("Name(%q)[0].Reason = %q; want %q", decl, got.Reason, NameReasonSeveralDeclarations)
	}
	if got.ID != "" || got.Kind != "" {
		t.Errorf("Name(%q)[0] = %+v; want no ID and no Kind on failure", decl, got)
	}
}

// TestName_Malformed covers a fragment that still fails to parse after the completion retry.
func TestName_Malformed(t *testing.T) {
	const decl = "const"
	got := Name([]Declaration{{Unit: nameUnit, Decl: decl}})[0]
	if got.Reason != NameReasonParse {
		t.Errorf("Name(%q)[0].Reason = %q; want %q", decl, got.Reason, NameReasonParse)
	}
	if got.Error != "declaration does not parse" {
		t.Errorf("Name(%q)[0].Error = %q; want %q", decl, got.Error, "declaration does not parse")
	}
}

// TestName_ZeroSymbols covers two shapes that parse cleanly but declare no listable symbol: a
// blank-named function, and a comment-only fragment.
func TestName_ZeroSymbols(t *testing.T) {
	tests := []struct {
		name string
		decl string
	}{
		{"BlankFunction", "func _() {}"},
		{"CommentOnly", "// just a comment"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Name([]Declaration{{Unit: nameUnit, Decl: tt.decl}})[0]
			if got.Reason != NameReasonNoDeclaration {
				t.Errorf("Name(%q)[0].Reason = %q; want %q", tt.decl, got.Reason, NameReasonNoDeclaration)
			}
			if got.Error != "declaration declares no symbol" {
				t.Errorf("Name(%q)[0].Error = %q; want %q", tt.decl, got.Error, "declaration declares no symbol")
			}
		})
	}
}

// TestName_SeveralSymbols covers two shapes that declare more than one symbol: an ungrouped const
// spec naming two identifiers, and two declarations in one fragment. Both assert the error sentence
// carries the actual count.
func TestName_SeveralSymbols(t *testing.T) {
	tests := []struct {
		name string
		decl string
		want string
	}{
		{"TwoNamesOneSpec", "const X, Y = 1, 2", "declaration declares 2 symbols; exactly one is required"},
		{"TwoDeclarations", "func F() {}\nfunc G() {}", "declaration declares 2 symbols; exactly one is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Name([]Declaration{{Unit: nameUnit, Decl: tt.decl}})[0]
			if got.Reason != NameReasonSeveralDeclarations {
				t.Errorf("Name(%q)[0].Reason = %q; want %q", tt.decl, got.Reason, NameReasonSeveralDeclarations)
			}
			if got.Error != tt.want {
				t.Errorf("Name(%q)[0].Error = %q; want %q", tt.decl, got.Error, tt.want)
			}
		})
	}
}

// TestName_BadUnit covers three unit shapes glyph.Parse rejects, each propagating the grammar's
// own reason word verbatim — compared against the corresponding glyph.Reason constant converted to
// a string, never against a literal.
func TestName_BadUnit(t *testing.T) {
	tests := []struct {
		name string
		unit string
		want glyph.Reason
	}{
		{"HashInUnit", "bad#unit", glyph.ReasonMultipleSeparators},
		{"EmptySegment", "a//b", glyph.ReasonUnitEmptySegment},
		{"DotSegment", "a/../b", glyph.ReasonUnitDotSegment},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Name([]Declaration{{Unit: tt.unit, Decl: "func F() {}"}})[0]
			if got.Reason != string(tt.want) {
				t.Errorf("Name(unit=%q)[0].Reason = %q; want %q", tt.unit, got.Reason, string(tt.want))
			}
			if got.ID != "" || got.Kind != "" {
				t.Errorf("Name(unit=%q)[0] = %+v; want no ID and no Kind on failure", tt.unit, got)
			}
		})
	}
}

// TestName_ReasonCompleteness mirrors the glyph package's own completeness test: NameReasons
// contains each of the four maker-owned reason constants exactly once and nothing else.
func TestName_ReasonCompleteness(t *testing.T) {
	want := map[string]bool{
		NameReasonParse:               true,
		NameReasonNoDeclaration:       true,
		NameReasonSeveralDeclarations: true,
		NameReasonInternal:            true,
	}
	if len(NameReasons) != len(want) {
		t.Fatalf("len(NameReasons) = %d; want %d", len(NameReasons), len(want))
	}
	seen := make(map[string]bool, len(NameReasons))
	for _, r := range NameReasons {
		if seen[r] {
			t.Errorf("NameReasons contains %q more than once", r)
		}
		seen[r] = true
		if !want[r] {
			t.Errorf("NameReasons contains unexpected value %q", r)
		}
	}
	for r := range want {
		if !seen[r] {
			t.Errorf("NameReasons is missing %q", r)
		}
	}
}

// TestName_BatchSemantics covers Name's batch behaviour: a mixed batch preserves length and order,
// every entry's Unit and Target echo its input byte-identically, and a nil or empty input returns
// an empty, non-nil slice.
func TestName_BatchSemantics(t *testing.T) {
	decls := []Declaration{
		{Unit: "u1", Decl: "func F() {}"},
		{Unit: "u2", Decl: "const"},
		{Unit: "u3", Decl: "type T int"},
	}
	got := Name(decls)
	if len(got) != len(decls) {
		t.Fatalf("len(Name(decls)) = %d; want %d", len(got), len(decls))
	}
	for i, d := range decls {
		if got[i].Unit != d.Unit {
			t.Errorf("Name(decls)[%d].Unit = %q; want %q", i, got[i].Unit, d.Unit)
		}
		if got[i].Target != d.Decl {
			t.Errorf("Name(decls)[%d].Target = %q; want %q", i, got[i].Target, d.Decl)
		}
	}
	if got[0].Error != "" {
		t.Errorf("Name(decls)[0].Error = %q; want no error", got[0].Error)
	}
	if got[1].Error == "" {
		t.Errorf("Name(decls)[1].Error = %q; want an error", got[1].Error)
	}
	if got[2].Error != "" {
		t.Errorf("Name(decls)[2].Error = %q; want no error", got[2].Error)
	}

	t.Run("Nil", func(t *testing.T) {
		got := Name(nil)
		if got == nil {
			t.Fatal("Name(nil) = nil; want an empty, non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("len(Name(nil)) = %d; want 0", len(got))
		}
	})
	t.Run("Empty", func(t *testing.T) {
		got := Name([]Declaration{})
		if got == nil {
			t.Fatal("Name([]Declaration{}) = nil; want an empty, non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("len(Name([]Declaration{})) = %d; want 0", len(got))
		}
	})
}
