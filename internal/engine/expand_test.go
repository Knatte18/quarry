// expand_test.go covers Repo.Expand: the found row over a struct with members across two files, an
// interface whose members lie inside its head span, and a type with no members at all — plus, in
// card 17's own tests, the remaining rows of the disposition table statusForMatches and the kind gate
// together produce, and in card 18's, the collision row and the cross-verb agreement with Resolve.
//
// Fixtures are the same testdata/glyphs/, testdata/methods/ and testdata/tags/ packages
// resolve_test.go reads, opened through the same openModuleRepo helper, plus one run-time unit
// collision tree built through the existing openScratchRepo helper — this file adds no new helper.

package engine

import (
	"errors"
	"testing"

	"github.com/Knatte18/quarry/glyph"
)

// isZeroExpandAnswer reports whether got is the zero ExpandAnswer, field by field: ExpandAnswer
// holds two slices and a pointer, none of them comparable with ==, so a plain got != (ExpandAnswer{})
// does not compile.
func isZeroExpandAnswer(got ExpandAnswer) bool {
	return got.ID == "" && got.Status == "" && got.Unit == "" && got.Head == nil && got.Members == nil && got.Candidates == nil
}

// TestExpand_Struct expands the methods fixture package's Widget type, whose three methods span two
// files, and asserts the head span, the member order, and the marshalled shape of a found answer
// carrying both head and members.
func TestExpand_Struct(t *testing.T) {
	r := openModuleRepo(t)
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
	r := openModuleRepo(t)
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
	r := openModuleRepo(t)
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

// TestExpand_NotAType asserts a package-level function glyph and a package-level const glyph — each
// matching exactly once, so the single-match requirement is met by construction — return a
// *NotATypeError naming the walk's own kind and the zero ExpandAnswer. It then asserts the case a
// match-count gate would miss: the bare init glyph, which matches three declarations and still
// returns a *NotATypeError, never a multipart answer, which ExpandAnswer's Status does not admit.
func TestExpand_NotAType(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		wantKind Kind
	}{
		// AnonParam is declared once in iface.go — its anonymous parameter interface contributes no
		// symbol of its own — so the glyph matches exactly one declaration.
		{"Function", "internal/engine/testdata/glyphs#AnonParam", KindFunction},
		// UngroupedConst is declared once in decls.go.
		{"Const", "internal/engine/testdata/glyphs#UngroupedConst", KindConst},
		// The bare init glyph matches three declarations in inits.go, all of them functions. A
		// match-count gate would call this multipart; the kind gate catches it instead.
		{"Init", "internal/engine/testdata/glyphs#init", KindFunction},
	}

	r := openModuleRepo(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Expand(tt.target)
			var notATypeErr *NotATypeError
			if !errors.As(err, &notATypeErr) {
				t.Fatalf("Expand(%q) error = %v; want errors.As to *NotATypeError", tt.target, err)
			}
			if notATypeErr.Kind != tt.wantKind {
				t.Errorf("NotATypeError.Kind = %q; want %q", notATypeErr.Kind, tt.wantKind)
			}
			if !isZeroExpandAnswer(got) {
				t.Errorf("Expand(%q) = %+v; want the zero ExpandAnswer", tt.target, got)
			}
		})
	}
}

// TestExpand_AmbiguousBuildTags asserts the duplicated type glyph of the tags fixture package
// answers ambiguous with both declarations in Candidates and no head or members, and that the mixed
// glyph — a type in one file and a function in the other — answers ambiguous too rather than a
// *NotATypeError, because the set holds a type and choosing between the two would be a silent pick.
func TestExpand_AmbiguousBuildTags(t *testing.T) {
	r := openModuleRepo(t)

	dupTarget := "internal/engine/testdata/tags#DupType"
	dup, err := r.Expand(dupTarget)
	if err != nil {
		t.Fatalf("Expand(%q) returned error: %v", dupTarget, err)
	}
	if dup.Status != StatusAmbiguous {
		t.Fatalf("Status = %q; want %q", dup.Status, StatusAmbiguous)
	}
	if len(dup.Candidates) != 2 {
		t.Fatalf("Candidates = %d entries; want 2", len(dup.Candidates))
	}
	if dup.Head != nil {
		t.Errorf("Head = %+v; want absent", dup.Head)
	}
	if dup.Members != nil {
		t.Errorf("Members = %v; want absent", dup.Members)
	}

	mixedTarget := "internal/engine/testdata/tags#Mixed"
	mixed, err := r.Expand(mixedTarget)
	if err != nil {
		t.Fatalf("Expand(%q) returned error: %v", mixedTarget, err)
	}
	if mixed.Status != StatusAmbiguous {
		t.Fatalf("Status = %q; want %q — a type and a function both named Mixed, choosing would be a silent pick", mixed.Status, StatusAmbiguous)
	}

	// The only marshal in the plan that observes candidates in its present state.
	m := marshalToMap(t, dup)
	candidatesVal, ok := m["candidates"].([]any)
	if !ok || len(candidatesVal) != 2 {
		t.Errorf("marshalled: candidates = %v; want two entries present under %q", m["candidates"], "candidates")
	}
	for _, key := range []string{"head", "members"} {
		if _, ok := m[key]; ok {
			t.Errorf("marshalled: unexpected key %q present in %v", key, m)
		}
	}
}

// TestExpand_MalformedTarget asserts a target the grammar rejects and a target with no "#" each
// return a non-nil error and the zero answer, that errors.As reaches a *glyph.ParseError in both
// cases, and that the no-separator case carries the grammar's own no-separator reason — proof that
// expand writes no separator check of its own.
func TestExpand_MalformedTarget(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		wantReason glyph.Reason
	}{
		{"RejectedByGrammar", "internal/engine/testdata/tree/pkg#A.B.C", glyph.ReasonMemberTooDeep},
		{"NoSeparator", "internal/engine/testdata/tree/pkg", glyph.ReasonNoSeparator},
	}

	r := openModuleRepo(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Expand(tt.target)
			if err == nil {
				t.Fatalf("Expand(%q) = %+v, nil error; want a non-nil error", tt.target, got)
			}
			if !isZeroExpandAnswer(got) {
				t.Errorf("Expand(%q) = %+v; want the zero ExpandAnswer", tt.target, got)
			}
			var parseErr *glyph.ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("Expand(%q) error = %v; want errors.As to *glyph.ParseError", tt.target, err)
			}
			if parseErr.Reason != tt.wantReason {
				t.Errorf("ParseError.Reason = %q; want %q", parseErr.Reason, tt.wantReason)
			}
		})
	}
}

// TestExpand_NotFound asserts that a name that does not exist inside an existing unit answers
// not_found with unit: found and a nil error, and that a glyph whose unit directory does not exist
// answers not_found with unit: not_found and a nil error — a miss is a legitimate answer with a
// status, never a failure.
func TestExpand_NotFound(t *testing.T) {
	r := openModuleRepo(t)

	missingName := "internal/engine/testdata/glyphs#NoSuchDeclaration"
	nameRes, err := r.Expand(missingName)
	if err != nil {
		t.Fatalf("Expand(%q) returned error: %v", missingName, err)
	}
	if nameRes.Status != StatusNotFound || nameRes.Unit != StatusFound {
		t.Errorf("Expand(%q) = %+v; want not_found with unit: found", missingName, nameRes)
	}

	missingUnit := "internal/engine/testdata/does-not-exist#X"
	unitRes, err := r.Expand(missingUnit)
	if err != nil {
		t.Fatalf("Expand(%q) returned error: %v", missingUnit, err)
	}
	if unitRes.Status != StatusNotFound || unitRes.Unit != StatusNotFound {
		t.Errorf("Expand(%q) = %+v; want not_found with unit: not_found", missingUnit, unitRes)
	}

	// The fourth and last of the four marshals across cards 16 and 17 that together cover every one
	// of ExpandAnswer's six keys in both its present and its omitted state — this one observes unit
	// present and head absent.
	m := marshalToMap(t, nameRes)
	for _, key := range []string{"id", "status", "unit"} {
		if _, ok := m[key]; !ok {
			t.Errorf("marshalled: missing key %q in %v", key, m)
		}
	}
	if got, want := m["status"], string(StatusNotFound); got != want {
		t.Errorf("marshalled status = %v; want %q", got, want)
	}
	if got, want := m["unit"], string(nameRes.Unit); got != want {
		t.Errorf("marshalled unit = %v; want %q", got, want)
	}
	for _, key := range []string{"head", "members", "candidates"} {
		if _, ok := m[key]; ok {
			t.Errorf("marshalled: unexpected key %q present in %v", key, m)
		}
	}
}

// TestExpand_Collision builds a run-time unit collision tree, the same shape batch 3's own collision
// tests use — a "foo/" directory whose external test file and a literally-named "foo_test/" sibling
// directory both belong to the "foo_test" unit — declaring a Thing type in both files' external test
// package and a second type, Second, in the literal directory's file only.
//
// It asserts three things about Expand: the doubly-declared Thing glyph answers ambiguous with both
// declarations in Candidates and no head or members; the singly-declared Second glyph — one match,
// and a type — answers ambiguous too rather than found, which is the row that keeps Expand and
// Resolve from disagreeing; and Candidates come back ordered by file then start line even though the
// literal foo_test directory's symbols were appended first.
//
// It then calls Resolve on the same two glyphs and asserts both verbs report the same status and the
// same candidate ids in the same order, so the agreement is shown rather than assumed, and asserts a
// name declared in neither directory answers not_found with unit: found — the zero-match row, which
// statusForMatches evaluates before its collision row.
func TestExpand_Collision(t *testing.T) {
	r := openScratchRepo(t, "expand-unit-collision", map[string]string{
		"foo/own.go":      "package foo\n\nfunc FooOwn() {}\n",
		"foo/own_test.go": "package foo_test\n\ntype Thing struct{}\n",
		"foo_test/lit.go": "package foo_test\n\ntype Thing struct{}\n\ntype Second struct{}\n",
	})

	dupTarget := "foo_test#Thing"
	dupExpand, err := r.Expand(dupTarget)
	if err != nil {
		t.Fatalf("Expand(%q) returned error: %v", dupTarget, err)
	}
	if dupExpand.Status != StatusAmbiguous {
		t.Fatalf("Expand(%q) Status = %q; want %q", dupTarget, dupExpand.Status, StatusAmbiguous)
	}
	if len(dupExpand.Candidates) != 2 {
		t.Fatalf("Expand(%q) Candidates = %d entries; want 2", dupTarget, len(dupExpand.Candidates))
	}
	if dupExpand.Head != nil {
		t.Errorf("Expand(%q) Head = %+v; want absent", dupTarget, dupExpand.Head)
	}
	if dupExpand.Members != nil {
		t.Errorf("Expand(%q) Members = %v; want absent", dupTarget, dupExpand.Members)
	}
	// Candidates ordered by file then start line: "foo/own_test.go" sorts before "foo_test/lit.go"
	// even though unitDirs returns the literal foo_test directory first.
	if dupExpand.Candidates[0].File != "foo/own_test.go" {
		t.Errorf("Expand(%q) Candidates[0].File = %q; want %q", dupTarget, dupExpand.Candidates[0].File, "foo/own_test.go")
	}
	if dupExpand.Candidates[1].File != "foo_test/lit.go" {
		t.Errorf("Expand(%q) Candidates[1].File = %q; want %q", dupTarget, dupExpand.Candidates[1].File, "foo_test/lit.go")
	}

	secondTarget := "foo_test#Second"
	secondExpand, err := r.Expand(secondTarget)
	if err != nil {
		t.Fatalf("Expand(%q) returned error: %v", secondTarget, err)
	}
	if secondExpand.Status != StatusAmbiguous {
		t.Errorf("Expand(%q) Status = %q; want %q — a single match under a unit collision is still ambiguous", secondTarget, secondExpand.Status, StatusAmbiguous)
	}
	if len(secondExpand.Candidates) != 1 {
		t.Errorf("Expand(%q) Candidates = %d entries; want 1", secondTarget, len(secondExpand.Candidates))
	}

	// The two verbs must agree: same status, same candidate ids, in the same order.
	dupResolveResults, err := r.Resolve([]string{dupTarget})
	if err != nil {
		t.Fatalf("Resolve(%q) returned error: %v", dupTarget, err)
	}
	dupResolve := dupResolveResults[0]
	if dupResolve.Status != dupExpand.Status {
		t.Errorf("Resolve(%q) Status = %q; Expand Status = %q; want equal", dupTarget, dupResolve.Status, dupExpand.Status)
	}
	if len(dupResolve.Candidates) != len(dupExpand.Candidates) {
		t.Fatalf("Resolve(%q) Candidates = %d entries; Expand Candidates = %d; want equal", dupTarget, len(dupResolve.Candidates), len(dupExpand.Candidates))
	}
	for i := range dupResolve.Candidates {
		if dupResolve.Candidates[i].ID != dupExpand.Candidates[i].ID {
			t.Errorf("Candidates[%d].ID = %q (Resolve) vs %q (Expand); want equal", i, dupResolve.Candidates[i].ID, dupExpand.Candidates[i].ID)
		}
	}

	secondResolveResults, err := r.Resolve([]string{secondTarget})
	if err != nil {
		t.Fatalf("Resolve(%q) returned error: %v", secondTarget, err)
	}
	secondResolve := secondResolveResults[0]
	if secondResolve.Status != secondExpand.Status {
		t.Errorf("Resolve(%q) Status = %q; Expand Status = %q; want equal", secondTarget, secondResolve.Status, secondExpand.Status)
	}
	if len(secondResolve.Candidates) != len(secondExpand.Candidates) || secondResolve.Candidates[0].ID != secondExpand.Candidates[0].ID {
		t.Errorf("Resolve(%q) Candidates = %+v; Expand Candidates = %+v; want equal", secondTarget, secondResolve.Candidates, secondExpand.Candidates)
	}

	missingTarget := "foo_test#NoSuchName"
	missing, err := r.Expand(missingTarget)
	if err != nil {
		t.Fatalf("Expand(%q) returned error: %v", missingTarget, err)
	}
	if missing.Status != StatusNotFound || missing.Unit != StatusFound {
		t.Errorf("Expand(%q) = %+v; want not_found with unit: found", missingTarget, missing)
	}
}
