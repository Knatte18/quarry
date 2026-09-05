// render_test.go covers RenderJSON's key order, absent-field omission, unescaped HTML characters,
// indentation, and trailing-newline discipline, and RenderErrorJSON's exact byte output, over
// hand-built DirAnswer values only — no filesystem, no Open, no TOC.

package quarry

import (
	"strings"
	"testing"
)

// TestRenderJSON_KeyOrder pins that the emitted object's key order is
// internal/engine/answer.go's struct field declaration order — dir, package, language, doc, files,
// dirs, and within a file entry name, header, test, generated, package, language, lossy, error,
// symbols — asserted on the rendered bytes rather than a decoded map, since a map loses order.
func TestRenderJSON_KeyOrder(t *testing.T) {
	symbols := []Symbol{{ID: "pkg#Sym", Kind: KindFunction, Start: 1, End: 2, Signature: "func Sym()"}}
	a := DirAnswer{
		Dir:      "pkg",
		Package:  "pkg",
		Language: "go",
		Doc:      "Package pkg is a fixture.",
		Files: []FileEntry{
			{
				Name:      "file.go",
				Header:    "file.go is a fixture.",
				Test:      true,
				Generated: true,
				Package:   "pkg_test",
				Language:  "go",
				Lossy:     true,
				Error:     "boom",
				Symbols:   &symbols,
			},
		},
		Dirs: []DirAnswer{{Dir: "pkg/sub"}},
	}

	got, err := RenderJSON(a)
	if err != nil {
		t.Fatalf("RenderJSON() error = %v", err)
	}
	s := string(got)

	wantOrder := []string{
		`"dir"`, `"package"`, `"language"`, `"doc"`, `"files"`, `"dirs"`,
	}
	assertKeyOrder(t, s, wantOrder)

	fileKeysOrder := []string{
		`"name"`, `"header"`, `"test"`, `"generated"`, `"package"`, `"language"`, `"lossy"`, `"error"`, `"symbols"`,
	}
	// The file entry's own keys must appear, in order, after "files" and before "dirs".
	filesIdx := strings.Index(s, `"files"`)
	dirsIdx := strings.Index(s, `"dirs"`)
	if filesIdx == -1 || dirsIdx == -1 || filesIdx >= dirsIdx {
		t.Fatalf("RenderJSON() = %s; want \"files\" before \"dirs\"", s)
	}
	assertKeyOrder(t, s[filesIdx:dirsIdx], fileKeysOrder)
}

// assertKeyOrder fails the test unless every key in wantOrder appears in s, in that relative order.
func assertKeyOrder(t *testing.T, s string, wantOrder []string) {
	t.Helper()
	pos := -1
	for _, key := range wantOrder {
		idx := strings.Index(s, key)
		if idx == -1 {
			t.Fatalf("RenderJSON() = %s; want key %s present", s, key)
		}
		if idx <= pos {
			t.Fatalf("RenderJSON() = %s; want key %s after position %d, found at %d", s, key, pos, idx)
		}
		pos = idx
	}
}

// TestRenderJSON_AbsentFieldsOmitted covers that a DirAnswer with Test false and no Dirs renders
// with no "test" key and no "dirs" key, and that no "ok" key appears on the success path.
func TestRenderJSON_AbsentFieldsOmitted(t *testing.T) {
	a := DirAnswer{
		Dir:   "pkg",
		Files: []FileEntry{{Name: "file.go"}},
	}
	got, err := RenderJSON(a)
	if err != nil {
		t.Fatalf("RenderJSON() error = %v", err)
	}
	s := string(got)

	for _, absent := range []string{`"test"`, `"dirs"`, `"ok"`} {
		if strings.Contains(s, absent) {
			t.Errorf("RenderJSON() = %s; want no %s key", s, absent)
		}
	}
}

// TestRenderJSON_HTMLNotEscaped covers that a Doc containing '<', '>' and '&' renders those
// characters literally: the plain bytes are present, and none of Go's default HTML-escaping
// encoder's six-byte unicode substitutes (<, >, &) appear anywhere in the output.
func TestRenderJSON_HTMLNotEscaped(t *testing.T) {
	a := DirAnswer{Dir: ".", Doc: "a < b > c & d"}
	got, err := RenderJSON(a)
	if err != nil {
		t.Fatalf("RenderJSON() error = %v", err)
	}
	s := string(got)
	if !strings.Contains(s, "a < b > c & d") {
		t.Errorf("RenderJSON() = %s; want literal a < b > c & d", s)
	}
	// Built from individual bytes, not a literal backslash-u escape sequence in this source
	// file, so the search target is unambiguous: the six-byte substitutes Go's default
	// HTML-escaping encoder would emit for '<', '>' and '&'.
	backslash := string(rune(0x5c))
	escapedLT := backslash + "u003c"
	escapedGT := backslash + "u003e"
	escapedAmp := backslash + "u0026"
	for _, htmlEscape := range []string{escapedLT, escapedGT, escapedAmp} {
		if strings.Contains(s, htmlEscape) {
			t.Errorf("RenderJSON() = %s; want no %s escape", s, htmlEscape)
		}
	}
}

// TestRenderJSON_IndentAndNewline covers two-space indentation and exactly one trailing newline.
func TestRenderJSON_IndentAndNewline(t *testing.T) {
	a := DirAnswer{Dir: "."}
	got, err := RenderJSON(a)
	if err != nil {
		t.Fatalf("RenderJSON() error = %v", err)
	}
	s := string(got)
	if !strings.Contains(s, "\n  \"dir\"") {
		t.Errorf("RenderJSON() = %q; want two-space indentation before \"dir\"", s)
	}
	if !strings.HasSuffix(s, "\n") {
		t.Fatalf("RenderJSON() = %q; want it to end with a newline", s)
	}
	if strings.HasSuffix(s, "\n\n") {
		t.Errorf("RenderJSON() = %q; want exactly one trailing newline, not two", s)
	}
}

// TestRenderJSON_SymbolsPointerVsNil covers the wire-level distinction that matters: a nil Symbols
// omits the "symbols" key entirely (symbols were not requested), while a pointer to an empty slice
// keeps the key present as "symbols":[] (symbols were requested and the file declares none).
// encoding/json's omitempty checks pointer nilness here, not slice length, which is exactly why
// answer.go's doc comment calls Symbols "the one pointer field on this type": only the
// pointer-vs-nil distinction, not the slice's own length, can tell the two states apart on the wire.
func TestRenderJSON_SymbolsPointerVsNil(t *testing.T) {
	empty := []Symbol{}
	tests := []struct {
		name        string
		fe          FileEntry
		wantPresent bool
	}{
		{"NilSymbols", FileEntry{Name: "a.go"}, false},
		{"EmptySliceSymbols", FileEntry{Name: "a.go", Symbols: &empty}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := DirAnswer{Dir: ".", Files: []FileEntry{tt.fe}}
			got, err := RenderJSON(a)
			if err != nil {
				t.Fatalf("RenderJSON() error = %v", err)
			}
			present := strings.Contains(string(got), `"symbols"`)
			if present != tt.wantPresent {
				t.Errorf("RenderJSON() = %s; \"symbols\" key present = %v, want %v", got, present, tt.wantPresent)
			}
		})
	}
}

// TestRenderResolveJSON covers the byte contract and key order for every payload shape a resolve
// query produces: found, multipart, ambiguous with candidates, not-found with each unit value, a
// pre-resolution rejection carrying error and reason, and a path result carrying a directory answer.
func TestRenderResolveJSON(t *testing.T) {
	sym := Symbol{ID: "pkg#Sym", Kind: KindFunction, Start: 1, End: 2, Signature: "func Sym()"}
	tests := []struct {
		name string
		r    ResolveResult
	}{
		{"Found", ResolveResult{Target: "pkg#Sym", ID: "pkg#Sym", Status: StatusFound, Symbols: []Symbol{sym}}},
		{"Multipart", ResolveResult{Target: "pkg#init", ID: "pkg#init", Status: StatusMultipart, Symbols: []Symbol{sym, sym}}},
		{"AmbiguousWithCandidates", ResolveResult{Target: "pkg#Sym", ID: "pkg#Sym", Status: StatusAmbiguous, Candidates: []Symbol{sym, sym}}},
		{"NotFoundUnitFound", ResolveResult{Target: "pkg#Missing", ID: "pkg#Missing", Status: StatusNotFound, Unit: StatusFound}},
		{"NotFoundUnitNotFound", ResolveResult{Target: "pkg#Missing", ID: "pkg#Missing", Status: StatusNotFound, Unit: StatusNotFound}},
		{"PreResolutionRejection", ResolveResult{Target: "#bad", Error: "engine: bad glyph", Reason: "no_unit"}},
		{"PathResult", ResolveResult{Target: "pkg", Status: StatusFound, Listing: &DirAnswer{Dir: "pkg"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderResolveJSON(tt.r)
			if err != nil {
				t.Fatalf("RenderResolveJSON() error = %v", err)
			}
			s := string(got)
			assertKeyOrder(t, s, []string{`"target"`})
			if !strings.Contains(s, "\n  \"target\"") {
				t.Errorf("RenderResolveJSON() = %q; want two-space indentation before \"target\"", s)
			}
			if !strings.HasSuffix(s, "\n") || strings.HasSuffix(s, "\n\n") {
				t.Errorf("RenderResolveJSON() = %q; want exactly one trailing newline", s)
			}
			if strings.Contains(s, `"ok"`) {
				t.Errorf("RenderResolveJSON() = %s; want no \"ok\" key", s)
			}
		})
	}
}

// TestRenderResolveJSON_KeyOrder pins the found result's key order to ResolveResult's own field
// declaration order.
func TestRenderResolveJSON_KeyOrder(t *testing.T) {
	sym := Symbol{ID: "pkg#Sym", Kind: KindFunction, Start: 1, End: 2, Signature: "func Sym()"}
	r := ResolveResult{
		Target: "pkg#Sym", ID: "pkg#Sym", Status: StatusFound,
		Symbols: []Symbol{sym}, Listing: &DirAnswer{Dir: "pkg"},
	}
	got, err := RenderResolveJSON(r)
	if err != nil {
		t.Fatalf("RenderResolveJSON() error = %v", err)
	}
	assertKeyOrder(t, string(got), []string{
		`"target"`, `"id"`, `"status"`, `"symbols"`, `"listing"`,
	})
}

// TestRenderResolveJSON_NotFoundOmitsSymbolsAndCandidates covers that a not-found result's absent
// symbols and candidates fields are dropped entirely, not emitted as null or as an empty array.
func TestRenderResolveJSON_NotFoundOmitsSymbolsAndCandidates(t *testing.T) {
	r := ResolveResult{Target: "pkg#Missing", ID: "pkg#Missing", Status: StatusNotFound, Unit: StatusFound}
	got, err := RenderResolveJSON(r)
	if err != nil {
		t.Fatalf("RenderResolveJSON() error = %v", err)
	}
	s := string(got)
	for _, absent := range []string{`"symbols"`, `"candidates"`} {
		if strings.Contains(s, absent) {
			t.Errorf("RenderResolveJSON() = %s; want no %s key", s, absent)
		}
	}
}

// TestRenderExpandJSON covers the byte contract for every payload shape an expand query produces: a
// found answer with head and members, a found answer with a head and no members, a not-found answer,
// and an ambiguous answer.
func TestRenderExpandJSON(t *testing.T) {
	head := Symbol{ID: "pkg#Thing", Kind: KindType, Start: 1, End: 3, Signature: "type Thing struct{}"}
	member := Symbol{ID: "pkg#Thing.Method", Kind: KindMethod, Start: 5, End: 5, Signature: "func (t Thing) Method()"}
	tests := []struct {
		name string
		a    ExpandAnswer
	}{
		{"FoundWithMembers", ExpandAnswer{ID: "pkg#Thing", Status: StatusFound, Head: &head, Members: []Symbol{member}}},
		{"FoundNoMembers", ExpandAnswer{ID: "pkg#Thing", Status: StatusFound, Head: &head}},
		{"NotFound", ExpandAnswer{ID: "pkg#Missing", Status: StatusNotFound, Unit: StatusNotFound}},
		{"Ambiguous", ExpandAnswer{ID: "pkg#Thing", Status: StatusAmbiguous, Candidates: []Symbol{head, head}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderExpandJSON(tt.a)
			if err != nil {
				t.Fatalf("RenderExpandJSON() error = %v", err)
			}
			s := string(got)
			assertKeyOrder(t, s, []string{`"id"`, `"status"`})
			if !strings.Contains(s, "\n  \"id\"") {
				t.Errorf("RenderExpandJSON() = %q; want two-space indentation before \"id\"", s)
			}
			if !strings.HasSuffix(s, "\n") || strings.HasSuffix(s, "\n\n") {
				t.Errorf("RenderExpandJSON() = %q; want exactly one trailing newline", s)
			}
			if strings.Contains(s, `"ok"`) {
				t.Errorf("RenderExpandJSON() = %s; want no \"ok\" key", s)
			}
		})
	}
}

// TestRenderExpandJSON_KeyOrder pins the found-with-members answer's key order to ExpandAnswer's own
// field declaration order.
func TestRenderExpandJSON_KeyOrder(t *testing.T) {
	head := Symbol{ID: "pkg#Thing", Kind: KindType, Start: 1, End: 3, Signature: "type Thing struct{}"}
	member := Symbol{ID: "pkg#Thing.Method", Kind: KindMethod, Start: 5, End: 5, Signature: "func (t Thing) Method()"}
	a := ExpandAnswer{ID: "pkg#Thing", Status: StatusFound, Head: &head, Members: []Symbol{member}}
	got, err := RenderExpandJSON(a)
	if err != nil {
		t.Fatalf("RenderExpandJSON() error = %v", err)
	}
	assertKeyOrder(t, string(got), []string{`"id"`, `"status"`, `"head"`, `"members"`})
}

// TestRenderExpandJSON_NotFoundOmitsHeadAndMembers covers that a not-found answer's absent head and
// members fields are dropped entirely, not emitted as null or as an empty array.
func TestRenderExpandJSON_NotFoundOmitsHeadAndMembers(t *testing.T) {
	a := ExpandAnswer{ID: "pkg#Missing", Status: StatusNotFound, Unit: StatusNotFound}
	got, err := RenderExpandJSON(a)
	if err != nil {
		t.Fatalf("RenderExpandJSON() error = %v", err)
	}
	s := string(got)
	for _, absent := range []string{`"head"`, `"members"`, `"candidates"`} {
		if strings.Contains(s, absent) {
			t.Errorf("RenderExpandJSON() = %s; want no %s key", s, absent)
		}
	}
}

// TestRenderErrorJSON covers the exact bytes RenderErrorJSON emits: a plain message, one with '<'
// and '&' left unescaped, and one with a double quote escaped normally.
func TestRenderErrorJSON(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want string
	}{
		{"PlainMessage", "boom", `{"ok":false,"error":"boom"}` + "\n"},
		{"HTMLCharsNotEscaped", "a < b & c", `{"ok":false,"error":"a < b & c"}` + "\n"},
		{"DoubleQuoteEscaped", `say "hi"`, `{"ok":false,"error":"say \"hi\""}` + "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(RenderErrorJSON(tt.msg))
			if got != tt.want {
				t.Errorf("RenderErrorJSON(%q) = %q; want %q", tt.msg, got, tt.want)
			}
		})
	}
}
