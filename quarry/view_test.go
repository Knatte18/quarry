// view_test.go covers GlyphView's projection contract and the glyphs view's two renderers, over
// hand-built DirAnswer and GlyphsAnswer values only — no filesystem, no Open, no TOC — matching
// quarry/render_test.go's own posture, plus (from card 4 onward) the facade-side drift assertion
// that Repo.Glyphs equals GlyphView composed with TOC and GlyphsOptions.

package quarry

import (
	"reflect"
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
