// view_test.go covers GlyphView's projection contract and the glyphs view's two renderers, over
// hand-built DirAnswer and GlyphsAnswer values only — no filesystem, no Open, no TOC — matching
// quarry/render_test.go's own posture, plus (from card 4 onward) the facade-side drift assertion
// that Repo.Glyphs equals GlyphView composed with TOC and GlyphsOptions.

package quarry

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// TestGlyphView covers the projection's shape over a table of hand-built DirAnswer values: a
// single-file answer; a two-directory-deep nesting asserting depth-first order; a nil Symbols
// pointer and a non-nil pointer to an empty slice, both contributing nothing; Lossy and Error
// entries landing in Incomplete, sorted; and File filled through joinRel, including the Dir == "."
// root case where the joined path is the bare file name.
func TestGlyphView(t *testing.T) {
	tests := []struct {
		name   string
		target string
		a      DirAnswer
		want   GlyphsAnswer
	}{
		{
			name:   "SingleFile",
			target: "sub",
			a: DirAnswer{
				Dir: "sub",
				Files: []FileEntry{
					{Name: "a.go", Symbols: &[]Symbol{
						{ID: "sub#Foo", Kind: KindFunction, Start: 1, End: 2},
					}},
				},
			},
			want: GlyphsAnswer{
				Target: "sub",
				Symbols: []Symbol{
					{ID: "sub#Foo", Kind: KindFunction, Start: 1, End: 2, File: "sub/a.go"},
				},
			},
		},
		{
			name:   "NestedTwoDeep_DepthFirstOrder",
			target: ".",
			a: DirAnswer{
				Dir: ".",
				Files: []FileEntry{
					{Name: "root.go", Symbols: &[]Symbol{{ID: "#Root", Kind: KindFunction, Start: 1, End: 1}}},
				},
				Dirs: []DirAnswer{
					{
						Dir: "a",
						Files: []FileEntry{
							{Name: "a1.go", Symbols: &[]Symbol{{ID: "a#A1", Kind: KindFunction, Start: 1, End: 1}}},
						},
						Dirs: []DirAnswer{
							{
								Dir: "a/b",
								Files: []FileEntry{
									{Name: "b1.go", Symbols: &[]Symbol{{ID: "a/b#B1", Kind: KindFunction, Start: 1, End: 1}}},
								},
							},
						},
					},
				},
			},
			want: GlyphsAnswer{
				Target: ".",
				Symbols: []Symbol{
					{ID: "#Root", Kind: KindFunction, Start: 1, End: 1, File: "root.go"},
					{ID: "a#A1", Kind: KindFunction, Start: 1, End: 1, File: "a/a1.go"},
					{ID: "a/b#B1", Kind: KindFunction, Start: 1, End: 1, File: "a/b/b1.go"},
				},
			},
		},
		{
			name:   "NilSymbolsContributesNothing",
			target: "sub",
			a: DirAnswer{
				Dir:   "sub",
				Files: []FileEntry{{Name: "a.go"}},
			},
			want: GlyphsAnswer{Target: "sub"},
		},
		{
			name:   "EmptySliceSymbolsContributesNothing",
			target: "sub",
			a: DirAnswer{
				Dir:   "sub",
				Files: []FileEntry{{Name: "a.go", Symbols: &[]Symbol{}}},
			},
			want: GlyphsAnswer{Target: "sub"},
		},
		{
			name:   "LossyAndErrorLandInIncompleteSorted",
			target: ".",
			a: DirAnswer{
				Dir: ".",
				Files: []FileEntry{
					{Name: "z.go", Lossy: true},
					{Name: "a.go", Error: "boom"},
				},
			},
			want: GlyphsAnswer{Target: ".", Incomplete: []string{"a.go", "z.go"}},
		},
		{
			name:   "FileJoinedThroughRootDir",
			target: ".",
			a: DirAnswer{
				Dir: ".",
				Files: []FileEntry{
					{Name: "root.go", Symbols: &[]Symbol{{ID: "#Root", Kind: KindFunction, Start: 1, End: 1}}},
				},
			},
			want: GlyphsAnswer{
				Target:  ".",
				Symbols: []Symbol{{ID: "#Root", Kind: KindFunction, Start: 1, End: 1, File: "root.go"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GlyphView(tt.target, tt.a)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GlyphView(%q, ...) = %+v; want %+v", tt.target, got, tt.want)
			}
		})
	}
}

// TestGlyphView_ClearsDocSignatureSigEnd covers that Doc, Signature and SigEnd are cleared on
// every returned symbol while every other field, including a non-empty Signature in the input, is
// carried across untouched in the input value itself.
func TestGlyphView_ClearsDocSignatureSigEnd(t *testing.T) {
	sym := Symbol{
		ID: "sub#Foo", Kind: KindFunction, Start: 1, End: 5, SigEnd: 3,
		Signature: "func Foo()", Doc: "Foo does things.",
	}
	symbols := []Symbol{sym}
	a := DirAnswer{Dir: "sub", Files: []FileEntry{{Name: "a.go", Symbols: &symbols}}}

	got := GlyphView("sub", a)

	if len(got.Symbols) != 1 {
		t.Fatalf("len(Symbols) = %d; want 1", len(got.Symbols))
	}
	gotSym := got.Symbols[0]
	if gotSym.Doc != "" || gotSym.Signature != "" || gotSym.SigEnd != 0 {
		t.Errorf("GlyphView(...) symbol = %+v; want Doc, Signature and SigEnd all cleared", gotSym)
	}
	if gotSym.ID != sym.ID || gotSym.Kind != sym.Kind || gotSym.Start != sym.Start || gotSym.End != sym.End {
		t.Errorf("GlyphView(...) symbol = %+v; want other fields carried across untouched", gotSym)
	}
	if gotSym.File != "sub/a.go" {
		t.Errorf("GlyphView(...) symbol.File = %q; want %q", gotSym.File, "sub/a.go")
	}
}

// TestGlyphView_NoMutation is the explicit no-mutation case: it builds an input, calls GlyphView,
// and asserts the input's own symbols still carry their original Doc/Signature/SigEnd/File
// afterward — GlyphView must not mutate a or share a backing array with any slice inside it.
func TestGlyphView_NoMutation(t *testing.T) {
	original := Symbol{
		ID: "sub#Foo", Kind: KindFunction, Start: 1, End: 5, SigEnd: 3,
		Signature: "func Foo()", Doc: "Foo does things.",
	}
	symbols := []Symbol{original}
	a := DirAnswer{Dir: "sub", Files: []FileEntry{{Name: "a.go", Symbols: &symbols}}}

	_ = GlyphView("sub", a)

	if !reflect.DeepEqual(symbols[0], original) {
		t.Errorf("input symbol after GlyphView = %+v; want unchanged %+v", symbols[0], original)
	}
}

// TestRenderGlyphsJSON_EnvelopeKeyOrder pins the emitted object's key order to target, symbols,
// incomplete, and asserts incomplete is present when non-empty.
func TestRenderGlyphsJSON_EnvelopeKeyOrder(t *testing.T) {
	a := GlyphsAnswer{
		Target:     "sub",
		Symbols:    []Symbol{{ID: "sub#Foo", Kind: KindFunction, File: "sub/a.go", Start: 1, End: 2}},
		Incomplete: []string{"sub/b.go"},
	}
	got, err := RenderGlyphsJSON(a)
	if err != nil {
		t.Fatalf("RenderGlyphsJSON() error = %v", err)
	}
	assertKeyOrder(t, string(got), []string{`"target"`, `"symbols"`, `"incomplete"`})
}

// TestRenderGlyphsJSON_IncompleteAbsentWhenEmpty covers that an empty Incomplete slice omits the
// "incomplete" key entirely.
func TestRenderGlyphsJSON_IncompleteAbsentWhenEmpty(t *testing.T) {
	a := GlyphsAnswer{Target: "sub"}
	got, err := RenderGlyphsJSON(a)
	if err != nil {
		t.Fatalf("RenderGlyphsJSON() error = %v", err)
	}
	if strings.Contains(string(got), `"incomplete"`) {
		t.Errorf("RenderGlyphsJSON() = %s; want no \"incomplete\" key", got)
	}
}

// TestRenderGlyphsJSON_ZeroSymbolsEmitsEmptyArray covers the case a nil Symbols would render
// "symbols": null: RenderGlyphsJSON must instead emit "symbols": [], since nothing else in this
// suite would catch a regression to a nil-slice marshal.
func TestRenderGlyphsJSON_ZeroSymbolsEmitsEmptyArray(t *testing.T) {
	a := GlyphsAnswer{Target: "sub"}
	got, err := RenderGlyphsJSON(a)
	if err != nil {
		t.Fatalf("RenderGlyphsJSON() error = %v", err)
	}
	s := string(got)
	if !strings.Contains(s, `"symbols": []`) {
		t.Errorf("RenderGlyphsJSON() = %s; want \"symbols\": []", s)
	}
	if strings.Contains(s, `"symbols": null`) {
		t.Errorf("RenderGlyphsJSON() = %s; want no \"symbols\": null", s)
	}
}

// TestRenderGlyphsJSON_SymbolObjectKeyOrder covers that each symbol object carries exactly id,
// kind, file, start, end, in that order.
func TestRenderGlyphsJSON_SymbolObjectKeyOrder(t *testing.T) {
	a := GlyphsAnswer{
		Target:  "sub",
		Symbols: []Symbol{{ID: "sub#Foo", Kind: KindFunction, File: "sub/a.go", Start: 1, End: 2}},
	}
	got, err := RenderGlyphsJSON(a)
	if err != nil {
		t.Fatalf("RenderGlyphsJSON() error = %v", err)
	}
	s := string(got)
	symbolsIdx := strings.Index(s, `"symbols"`)
	if symbolsIdx == -1 {
		t.Fatalf("RenderGlyphsJSON() = %s; want a \"symbols\" key", s)
	}
	assertKeyOrder(t, s[symbolsIdx:], []string{`"symbols"`, `"id"`, `"kind"`, `"file"`, `"start"`, `"end"`})
}

// TestRenderGlyphsJSON_ClearedFieldsAbsent covers the reason glyphSymbol exists as a shadow struct
// rather than a cleared-field marshal of Symbol itself: an input symbol carrying a non-empty
// Signature before GlyphView cleared it must render with no "signature", "doc" or "sigend" key —
// a symbol with an empty signature to begin with would prove nothing here.
func TestRenderGlyphsJSON_ClearedFieldsAbsent(t *testing.T) {
	symbols := []Symbol{{
		ID: "sub#Foo", Kind: KindFunction, Start: 1, End: 5, SigEnd: 3,
		Signature: "func Foo()", Doc: "Foo does things.",
	}}
	src := DirAnswer{Dir: "sub", Files: []FileEntry{{Name: "a.go", Symbols: &symbols}}}
	view := GlyphView("sub", src)

	got, err := RenderGlyphsJSON(view)
	if err != nil {
		t.Fatalf("RenderGlyphsJSON() error = %v", err)
	}
	s := string(got)
	for _, absent := range []string{`"signature"`, `"doc"`, `"sigend"`} {
		if strings.Contains(s, absent) {
			t.Errorf("RenderGlyphsJSON() = %s; want no %s key", s, absent)
		}
	}
}

// TestRenderGlyphsJSON_ByteContract covers the shared byte contract: two-space indentation,
// exactly one trailing newline, and a '<' in a symbol id or target left unescaped.
func TestRenderGlyphsJSON_ByteContract(t *testing.T) {
	a := GlyphsAnswer{
		Target:  "sub<pkg>",
		Symbols: []Symbol{{ID: "sub#Foo<T>", Kind: KindFunction, File: "sub/a.go", Start: 1, End: 2}},
	}
	got, err := RenderGlyphsJSON(a)
	if err != nil {
		t.Fatalf("RenderGlyphsJSON() error = %v", err)
	}
	s := string(got)
	if !strings.Contains(s, "\n  \"target\"") {
		t.Errorf("RenderGlyphsJSON() = %q; want two-space indentation before \"target\"", s)
	}
	if !strings.HasSuffix(s, "\n") || strings.HasSuffix(s, "\n\n") {
		t.Errorf("RenderGlyphsJSON() = %q; want exactly one trailing newline", s)
	}
	if !strings.Contains(s, "sub<pkg>") || !strings.Contains(s, "sub#Foo<T>") {
		t.Errorf("RenderGlyphsJSON() = %s; want '<' left unescaped", s)
	}
}

// assertGlyphsTextInvariant is the whole-output invariant every RenderGlyphsText case must
// satisfy: no line has trailing whitespace, and a non-empty result ends with exactly one "\n".
func assertGlyphsTextInvariant(t *testing.T, got string) {
	t.Helper()
	if got == "" {
		return
	}
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("RenderGlyphsText() = %q; want it to end with a newline", got)
	}
	if strings.HasSuffix(got, "\n\n") {
		t.Fatalf("RenderGlyphsText() = %q; want exactly one trailing newline, not two", got)
	}
	for _, line := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
		if line != strings.TrimRight(line, " \t") {
			t.Errorf("RenderGlyphsText() = %q; line %q has trailing whitespace", got, line)
		}
	}
}

// TestRenderGlyphsText_LineGrammar covers the "<File>:<Start>-<End> <Kind> <ID>" line for each of
// the five kinds.
func TestRenderGlyphsText_LineGrammar(t *testing.T) {
	tests := []struct {
		name string
		kind Kind
		id   string
		want string
	}{
		{"Function", KindFunction, "sub#Foo", "sub/a.go:1-2 function sub#Foo\n"},
		{"Method", KindMethod, "sub#Thing.Method", "sub/a.go:1-2 method sub#Thing.Method\n"},
		{"Type", KindType, "sub#Thing", "sub/a.go:1-2 type sub#Thing\n"},
		{"Const", KindConst, "sub#X", "sub/a.go:1-2 const sub#X\n"},
		{"Var", KindVar, "sub#Y", "sub/a.go:1-2 var sub#Y\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := GlyphsAnswer{
				Symbols: []Symbol{{ID: tt.id, Kind: tt.kind, File: "sub/a.go", Start: 1, End: 2}},
			}
			got := RenderGlyphsText(a)
			if got != tt.want {
				t.Errorf("RenderGlyphsText() = %q; want %q", got, tt.want)
			}
			assertGlyphsTextInvariant(t, got)
		})
	}
}

// TestRenderGlyphsText_EmptyAnswerIsEmptyString covers that an answer with no symbols and no
// incomplete files renders as exactly "", never "\n".
func TestRenderGlyphsText_EmptyAnswerIsEmptyString(t *testing.T) {
	got := RenderGlyphsText(GlyphsAnswer{Target: "sub"})
	if got != "" {
		t.Errorf("RenderGlyphsText() = %q; want the empty string", got)
	}
}

// TestRenderGlyphsText_SymbolsNoIncomplete covers an answer with symbols and no incomplete files.
func TestRenderGlyphsText_SymbolsNoIncomplete(t *testing.T) {
	a := GlyphsAnswer{
		Symbols: []Symbol{
			{ID: "sub#Foo", Kind: KindFunction, File: "sub/a.go", Start: 1, End: 2},
			{ID: "sub#Bar", Kind: KindFunction, File: "sub/b.go", Start: 3, End: 4},
		},
	}
	want := "sub/a.go:1-2 function sub#Foo\nsub/b.go:3-4 function sub#Bar\n"
	got := RenderGlyphsText(a)
	if got != want {
		t.Errorf("RenderGlyphsText() = %q; want %q", got, want)
	}
	assertGlyphsTextInvariant(t, got)
}

// TestRenderGlyphsText_IncompleteNoSymbols covers an answer with no symbols but one or more
// incomplete files: the block is present, and the leading blank line does not produce a leading
// empty line before any symbol line exists.
func TestRenderGlyphsText_IncompleteNoSymbols(t *testing.T) {
	a := GlyphsAnswer{Incomplete: []string{"sub/a.go", "sub/b.go"}}
	want := "[incomplete] sub/a.go\n[incomplete] sub/b.go\n"
	got := RenderGlyphsText(a)
	if got != want {
		t.Errorf("RenderGlyphsText() = %q; want %q", got, want)
	}
	if strings.HasPrefix(got, "\n") {
		t.Errorf("RenderGlyphsText() = %q; want no leading blank line", got)
	}
	assertGlyphsTextInvariant(t, got)
}

// TestRenderGlyphsText_SymbolsAndIncomplete covers an answer with both symbol lines and incomplete
// files, asserting the single blank-line separator between the two blocks.
func TestRenderGlyphsText_SymbolsAndIncomplete(t *testing.T) {
	a := GlyphsAnswer{
		Symbols:    []Symbol{{ID: "sub#Foo", Kind: KindFunction, File: "sub/a.go", Start: 1, End: 2}},
		Incomplete: []string{"sub/b.go"},
	}
	want := "sub/a.go:1-2 function sub#Foo\n\n[incomplete] sub/b.go\n"
	got := RenderGlyphsText(a)
	if got != want {
		t.Errorf("RenderGlyphsText() = %q; want %q", got, want)
	}
	assertGlyphsTextInvariant(t, got)
}

// TestRepoGlyphs_MatchesGlyphViewOfTOC is the facade-side drift assertion: over a fixture tree,
// Glyphs(target) must be deep-equal to GlyphView(target, TOC(target, GlyphsOptions())) computed
// independently in the test.
func TestRepoGlyphs_MatchesGlyphViewOfTOC(t *testing.T) {
	root := writeScratchTree(t, "glyphs-drift", map[string]string{
		"sub/a.go": "package sub\n\nfunc Foo() {}\n\ntype Thing struct{}\n\nfunc (t Thing) Method() {}\n",
		"sub/b.go": "package sub\n\nfunc Bar() {}\n",
	})
	r, err := Open(root)
	if err != nil {
		t.Fatalf("Open(%q) returned error: %v", root, err)
	}

	toc, err := r.TOC("sub", GlyphsOptions())
	if err != nil {
		t.Fatalf("TOC(%q, GlyphsOptions()) returned error: %v", "sub", err)
	}
	want := GlyphView("sub", toc)

	got, err := r.Glyphs("sub")
	if err != nil {
		t.Fatalf("Glyphs(%q) returned error: %v", "sub", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Glyphs(%q) = %+v; want %+v", "sub", got, want)
	}
}

// TestRepoGlyphs_EmptyAndDotTargetBothYieldDotTarget covers that Glyphs("") and Glyphs(".") both
// return Target == "." — one query, one spelling, in its own answer.
func TestRepoGlyphs_EmptyAndDotTargetBothYieldDotTarget(t *testing.T) {
	root := writeScratchTree(t, "glyphs-empty-vs-dot", map[string]string{"a.go": "package p\n\nfunc Foo() {}\n"})
	r, err := Open(root)
	if err != nil {
		t.Fatalf("Open(%q) returned error: %v", root, err)
	}

	for _, target := range []string{"", "."} {
		got, err := r.Glyphs(target)
		if err != nil {
			t.Fatalf("Glyphs(%q) returned error: %v", target, err)
		}
		if got.Target != "." {
			t.Errorf("Glyphs(%q).Target = %q; want %q", target, got.Target, ".")
		}
	}
}

// TestRepoGlyphs_MissingTargetIsNotFound covers the pass-through error branch: nothing else in
// this plan reaches it, since the CLI's own glyphs path calls TOC and GlyphView directly and never
// calls this method. Glyphs on a target that does not exist in the fixture must return the zero
// GlyphsAnswer and an error satisfying errors.Is(err, ErrTargetNotFound).
func TestRepoGlyphs_MissingTargetIsNotFound(t *testing.T) {
	root := writeScratchTree(t, "glyphs-missing", map[string]string{"a.go": "package p\n"})
	r, err := Open(root)
	if err != nil {
		t.Fatalf("Open(%q) returned error: %v", root, err)
	}

	got, err := r.Glyphs("does-not-exist")
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("Glyphs(%q) error = %v; want errors.Is(err, ErrTargetNotFound)", "does-not-exist", err)
	}
	if !reflect.DeepEqual(got, GlyphsAnswer{}) {
		t.Errorf("Glyphs(%q) = %+v; want the zero GlyphsAnswer", "does-not-exist", got)
	}
}
