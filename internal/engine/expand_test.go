// expand_test.go covers Repo.Expand: the found row over a struct with members across two files, an
// interface whose members lie inside its head span, and a type with no members at all — plus, in
// card 17's own tests, the remaining rows of the disposition table statusForMatches and the kind gate
// together produce, and in card 18's, the collision row and the cross-verb agreement with Resolve.
//
// Fixtures are the same testdata/glyphs/, testdata/methods/ and testdata/tags/ packages
// resolve_test.go reads, opened through the same openQuarryRoot helper — this file adds no new
// helper.

package engine

import (
	"testing"

	"github.com/Knatte18/quarry/glyph"
)

// TestExpand_Struct expands the methods fixture package's Widget type, whose three methods span two
// files, and asserts the head span, the member order, and the marshalled shape of a found answer
// carrying both head and members.
func TestExpand_Struct(t *testing.T) {
	r := openQuarryRoot(t)
	target := "internal/engine/testdata/methods#Widget"

	got, err := r.Expand(target)
	if err != nil {
		t.Fatalf("Expand(%q) returned error: %v", target, err)
	}
	if got.Status != StatusFound {
		t.Fatalf("Status = %q; want %q", got.Status, StatusFound)
	}
	if got.Unit != "" {
		t.Errorf("Unit = %q; want empty", got.Unit)
	}
	if got.Candidates != nil {
		t.Errorf("Candidates = %v; want absent", got.Candidates)
	}
	if got.Head == nil {
		t.Fatalf("Head is nil; want the type's own head")
	}
	if got.Head.Kind != KindType {
		t.Errorf("Head.Kind = %q; want %q", got.Head.Kind, KindType)
	}
	if got.Head.ID != got.ID {
		t.Errorf("Head.ID = %q; want %q", got.Head.ID, got.ID)
	}
	wantHeadFile := "internal/engine/testdata/methods/widget.go"
	if got.Head.File != wantHeadFile {
		t.Errorf("Head.File = %q; want %q", got.Head.File, wantHeadFile)
	}

	// Read the expected head span from a SpansOf lookup of the same glyph, rather than hard-coding
	// line numbers, so this assertion survives an edit to the fixture's comments.
	spans, err := r.SpansOf(glyph.Glyph{Lang: glyph.Go, Unit: "internal/engine/testdata/methods", Name: "Widget"})
	if err != nil {
		t.Fatalf("SpansOf(Widget) returned error: %v", err)
	}
	if len(spans) != 1 {
		t.Fatalf("SpansOf(Widget) = %d symbols; want 1", len(spans))
	}
	wantSym := spans[0]
	if got.Head.Start != wantSym.HeadStart || got.Head.End != wantSym.HeadEnd {
		t.Errorf("Head.Start/End = %d/%d; want %d/%d (HeadStart/HeadEnd)", got.Head.Start, got.Head.End, wantSym.HeadStart, wantSym.HeadEnd)
	}

	if len(got.Members) != 3 {
		t.Fatalf("Members = %d entries; want 3", len(got.Members))
	}
	// aardvark.go sorts before widget.go, so its two methods — Alpha then Beta, in file order — come
	// back first, with widget.go's own Zeta last.
	wantFiles := []string{
		"internal/engine/testdata/methods/aardvark.go",
		"internal/engine/testdata/methods/aardvark.go",
		"internal/engine/testdata/methods/widget.go",
	}
	for i, mem := range got.Members {
		if mem.File != wantFiles[i] {
			t.Errorf("Members[%d].File = %q; want %q", i, mem.File, wantFiles[i])
		}
		if mem.ID == got.Head.ID {
			t.Errorf("Members[%d].ID = %q; equals the head's own ID, want distinct", i, mem.ID)
		}
	}

	m := marshalToMap(t, got)
	if _, ok := m["head"]; !ok {
		t.Errorf("marshalled: missing key %q in %v", "head", m)
	}
	membersVal, ok := m["members"].([]any)
	if !ok || len(membersVal) != 3 {
		t.Errorf("marshalled: members = %v; want three entries present under %q", m["members"], "members")
	}
}

// TestExpand_Interface expands the glyphs fixture package's named interface and asserts its members
// are exactly its own two methods, not the embedded interface's method, and that every member's span
// lies inside the head's span.
func TestExpand_Interface(t *testing.T) {
	r := openQuarryRoot(t)
	target := "internal/engine/testdata/glyphs#Iface"

	got, err := r.Expand(target)
	if err != nil {
		t.Fatalf("Expand(%q) returned error: %v", target, err)
	}
	if got.Status != StatusFound {
		t.Fatalf("Status = %q; want %q", got.Status, StatusFound)
	}
	if got.Head == nil {
		t.Fatalf("Head is nil; want the interface's own head")
	}
	if len(got.Members) != 2 {
		t.Fatalf("Members = %d entries; want 2 — Iface's own M1 and M2, not Embedded's E", len(got.Members))
	}
	for _, name := range []string{"M1", "M2"} {
		found := false
		for _, mem := range got.Members {
			if mem.Glyph.Name == name {
				found = true
			}
		}
		if !found {
			t.Errorf("Members = %+v; want an entry named %q", got.Members, name)
		}
	}
	for _, mem := range got.Members {
		if mem.Glyph.Name == "E" {
			t.Errorf("Members = %+v; want no entry for Embedded's own E", got.Members)
		}
	}

	for _, mem := range got.Members {
		if mem.Start < got.Head.Start || mem.End > got.Head.End {
			t.Errorf("member %q span %d-%d lies outside head span %d-%d", mem.ID, mem.Start, mem.End, got.Head.Start, got.Head.End)
		}
	}
	for i := 1; i < len(got.Members); i++ {
		if got.Members[i-1].Start > got.Members[i].Start {
			t.Errorf("Members not ordered by start line: %+v then %+v", got.Members[i-1], got.Members[i])
		}
	}
}

// TestExpand_TypeWithoutMembers expands the glyphs fixture package's defined scalar type, which
// declares no methods, and asserts a member-less type is a found answer with a head and nothing
// else — not an error and not a not_found. It marshals the answer to pin ExpandAnswer's six-key
// wire shape in the one found case where members is legitimately absent.
func TestExpand_TypeWithoutMembers(t *testing.T) {
	r := openQuarryRoot(t)
	target := "internal/engine/testdata/glyphs#Weekday"

	got, err := r.Expand(target)
	if err != nil {
		t.Fatalf("Expand(%q) returned error: %v", target, err)
	}
	if got.Status != StatusFound {
		t.Fatalf("Status = %q; want %q", got.Status, StatusFound)
	}
	if got.Head == nil {
		t.Fatalf("Head is nil; want the type's own head")
	}
	if got.Members != nil {
		t.Errorf("Members = %v; want nil — Weekday declares no methods", got.Members)
	}

	m := marshalToMap(t, got)
	for _, key := range []string{"id", "status", "head"} {
		if _, ok := m[key]; !ok {
			t.Errorf("marshalled: missing key %q in %v", key, m)
		}
	}
	if got, want := m["status"], string(StatusFound); got != want {
		t.Errorf("marshalled status = %v; want %q", got, want)
	}
	for _, key := range []string{"unit", "members", "candidates"} {
		if _, ok := m[key]; ok {
			t.Errorf("marshalled: unexpected key %q present in %v", key, m)
		}
	}
}
