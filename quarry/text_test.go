// text_test.go pins RenderText's grammar, per _mill/discussion.md's text-view-grammar decision, to
// the character, over hand-built DirAnswer values only — no filesystem.

package quarry

import (
	"strings"
	"testing"
)

// assertNoTrailingWhitespaceAndOneNewline is the whole-output invariant every case in this file
// must satisfy: no line has trailing whitespace, and the output ends with exactly one "\n".
func assertNoTrailingWhitespaceAndOneNewline(t *testing.T, got string) {
	t.Helper()
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("RenderText() = %q; want it to end with a newline", got)
	}
	if strings.HasSuffix(got, "\n\n") {
		t.Fatalf("RenderText() = %q; want exactly one trailing newline, not two", got)
	}
	for _, line := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
		if line != strings.TrimRight(line, " \t") {
			t.Errorf("RenderText() = %q; line %q has trailing whitespace", got, line)
		}
	}
}

func TestRenderText_DirectoryForm(t *testing.T) {
	tests := []struct {
		name string
		a    DirAnswer
		want string
	}{
		{
			name: "PackageAndDoc",
			a:    DirAnswer{Dir: "pkg", Package: "pkg", Language: "go", Doc: "Package pkg is a fixture.", Files: []FileEntry{{Name: "a.go"}}},
			want: "pkg (package pkg, go), 1 file\nPackage pkg is a fixture.\na.go\n",
		},
		{
			name: "NoPackageNoDoc",
			a:    DirAnswer{Dir: "pkg", Files: []FileEntry{{Name: "a.go"}}},
			want: "pkg, 1 file\na.go\n",
		},
		{
			name: "PackageNoLanguage",
			a:    DirAnswer{Dir: "pkg", Package: "pkg", Files: []FileEntry{{Name: "a.go"}}},
			want: "pkg (package pkg), 1 file\na.go\n",
		},
		{
			name: "OneFileSingular",
			a:    DirAnswer{Dir: "pkg", Files: []FileEntry{{Name: "a.go"}}},
			want: "pkg, 1 file\na.go\n",
		},
		{
			name: "TwoFilesPlural",
			a:    DirAnswer{Dir: "pkg", Files: []FileEntry{{Name: "a.go"}, {Name: "b.go"}}},
			want: "pkg, 2 files\na.go\nb.go\n",
		},
		{
			name: "NoFilesNoCountClause",
			a:    DirAnswer{Dir: "pkg", Package: "pkg"},
			want: "pkg (package pkg)\n",
		},
		{
			name: "DepthCutSubdirectoryIdentityOnly",
			a:    DirAnswer{Dir: "pkg", Doc: "Package pkg is a fixture."},
			want: "pkg\nPackage pkg is a fixture.\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderText(tt.a, false)
			if got != tt.want {
				t.Errorf("RenderText() = %q; want %q", got, tt.want)
			}
			assertNoTrailingWhitespaceAndOneNewline(t, got)
		})
	}
}

func TestRenderText_NestedBlocksDepthFirst(t *testing.T) {
	a := DirAnswer{
		Dir: ".",
		Dirs: []DirAnswer{
			{
				Dir: "a",
				Dirs: []DirAnswer{
					{Dir: "a/b"},
				},
			},
			{Dir: "c"},
		},
	}
	want := ".\n\na\n\na/b\n\nc\n"
	got := RenderText(a, false)
	if got != want {
		t.Errorf("RenderText() = %q; want %q", got, want)
	}
	assertNoTrailingWhitespaceAndOneNewline(t, got)
	if strings.HasPrefix(got, "\n") {
		t.Errorf("RenderText() = %q; want no leading blank line", got)
	}
}

func TestRenderText_Tags(t *testing.T) {
	tests := []struct {
		name string
		fe   FileEntry
		want string
	}{
		{"Test", FileEntry{Name: "a.go", Test: true}, "a.go [test]\n"},
		{"Generated", FileEntry{Name: "a.go", Generated: true}, "a.go [generated]\n"},
		{"Package", FileEntry{Name: "a.go", Package: "p"}, "a.go [package p]\n"},
		{"Language", FileEntry{Name: "a.go", Language: "go"}, "a.go [language go]\n"},
		{"Lossy", FileEntry{Name: "a.go", Lossy: true}, "a.go [lossy]\n"},
		{"Error", FileEntry{Name: "a.go", Error: "boom"}, "a.go [error boom]\n"},
		{
			"AllSix",
			FileEntry{Name: "a.go", Test: true, Generated: true, Package: "p", Language: "go", Lossy: true, Error: "msg"},
			"a.go [test] [generated] [package p] [language go] [lossy] [error msg]\n",
		},
		{
			"MultiLineErrorCollapses",
			FileEntry{Name: "a.go", Error: "line one\nline two"},
			"a.go [error line one line two]\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := DirAnswer{Dir: "pkg", Files: []FileEntry{tt.fe}}
			got := RenderText(a, false)
			want := "pkg, 1 file\n" + tt.want
			if got != want {
				t.Errorf("RenderText() = %q; want %q", got, want)
			}
			assertNoTrailingWhitespaceAndOneNewline(t, got)
		})
	}
}

func TestRenderText_FileForm(t *testing.T) {
	tests := []struct {
		name string
		a    DirAnswer
		want string
	}{
		{
			name: "RootDirNoDotSlashPrefix",
			a:    DirAnswer{Dir: ".", Files: []FileEntry{{Name: "README.md"}}},
			want: "README.md\n",
		},
		{
			name: "PackageFactsSameLineNoDoc",
			a: DirAnswer{
				Dir: "pkg", Package: "pkg", Language: "go", Doc: "Package pkg is a fixture.",
				Files: []FileEntry{{Name: "a.go", Header: "a.go does a thing."}},
			},
			want: "pkg/a.go (package pkg, go): a.go does a thing.\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderText(tt.a, true)
			if got != tt.want {
				t.Errorf("RenderText() = %q; want %q", got, tt.want)
			}
			assertNoTrailingWhitespaceAndOneNewline(t, got)
		})
	}
}

// TestRenderText_FormChosenByArgumentNotShape covers that the same one-file answer renders
// differently depending on targetIsFile, proving the form is chosen by the caller's argument and
// not inferred from the answer's shape.
func TestRenderText_FormChosenByArgumentNotShape(t *testing.T) {
	a := DirAnswer{Dir: "pkg", Package: "pkg", Files: []FileEntry{{Name: "a.go"}}}

	dirForm := RenderText(a, false)
	wantDirForm := "pkg (package pkg), 1 file\na.go\n"
	if dirForm != wantDirForm {
		t.Errorf("RenderText(a, false) = %q; want %q", dirForm, wantDirForm)
	}

	fileForm := RenderText(a, true)
	wantFileForm := "pkg/a.go (package pkg)\n"
	if fileForm != wantFileForm {
		t.Errorf("RenderText(a, true) = %q; want %q", fileForm, wantFileForm)
	}
}

func TestRenderText_Symbols(t *testing.T) {
	tests := []struct {
		name    string
		symbols *[]Symbol
		want    string
	}{
		{
			"WithDoc",
			&[]Symbol{{ID: "pkg#Sym", Start: 1, End: 3, Signature: "func Sym()", Doc: "Sym does a thing."}},
			"1-3 pkg#Sym: func Sym()\n    Sym does a thing.\n",
		},
		{
			"WithoutDoc",
			&[]Symbol{{ID: "pkg#Sym", Start: 1, End: 3, Signature: "func Sym()"}},
			"1-3 pkg#Sym: func Sym()\n",
		},
		{
			"SigEndZeroNoSigGroup",
			&[]Symbol{{ID: "pkg#Alias", Start: 1, End: 1, SigEnd: 0, Signature: "type Alias = int"}},
			"1-1 pkg#Alias: type Alias = int\n",
		},
		{
			"SigEndNonZero",
			&[]Symbol{{ID: "pkg#Fn", Start: 1, End: 5, SigEnd: 2, Signature: "func Fn()"}},
			"1-5 (sig 1-2) pkg#Fn: func Fn()\n",
		},
		{
			"NilSymbolsNoLines",
			nil,
			"",
		},
		{
			"EmptySliceSymbolsNoLines",
			&[]Symbol{},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := DirAnswer{Dir: "pkg", Files: []FileEntry{{Name: "a.go", Symbols: tt.symbols}}}
			got := RenderText(a, false)
			want := "pkg, 1 file\na.go\n" + tt.want
			if got != want {
				t.Errorf("RenderText() = %q; want %q", got, want)
			}
			assertNoTrailingWhitespaceAndOneNewline(t, got)
		})
	}
}

// TestRenderText_SymbolLine_EmptyFileRegressionGolden is a regression golden: it asserts that
// rendering a directory answer whose symbols carry an empty file field — the toc view's own shape —
// produces a byte-identical string to the one the pre-file-prefix implementation produced. The
// expected string is written as a fixed literal, not as a comparison against a stored file, so the
// card 4 file-prefix change cannot silently alter the toc view's own output.
func TestRenderText_SymbolLine_EmptyFileRegressionGolden(t *testing.T) {
	symbols := []Symbol{
		{ID: "pkg#Fn", Start: 1, End: 5, SigEnd: 2, Signature: "func Fn()", Doc: "Fn does a thing."},
		{ID: "pkg#Alias", Start: 7, End: 7, Signature: "type Alias = int"},
	}
	a := DirAnswer{Dir: "pkg", Files: []FileEntry{{Name: "a.go", Symbols: &symbols}}}
	want := "pkg, 1 file\na.go\n1-5 (sig 1-2) pkg#Fn: func Fn()\n    Fn does a thing.\n7-7 pkg#Alias: type Alias = int\n"
	got := RenderText(a, false)
	if got != want {
		t.Errorf("RenderText() = %q; want %q", got, want)
	}
	assertNoTrailingWhitespaceAndOneNewline(t, got)
}

// TestRenderText_SymbolLine_FilePrefix asserts a symbol carrying a non-empty file field renders with
// the "<file>:" prefix, covering both the with-signature-end and without-signature-end forms.
func TestRenderText_SymbolLine_FilePrefix(t *testing.T) {
	tests := []struct {
		name string
		sym  Symbol
		want string
	}{
		{
			"WithSigEnd",
			Symbol{ID: "pkg#Fn", File: "pkg/a.go", Start: 1, End: 5, SigEnd: 2, Signature: "func Fn()"},
			"pkg/a.go:1-5 (sig 1-2) pkg#Fn: func Fn()\n",
		},
		{
			"WithoutSigEnd",
			Symbol{ID: "pkg#Alias", File: "pkg/a.go", Start: 7, End: 7, Signature: "type Alias = int"},
			"pkg/a.go:7-7 pkg#Alias: type Alias = int\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b strings.Builder
			writeSymbolLine(&b, tt.sym)
			got := b.String()
			if got != tt.want {
				t.Errorf("writeSymbolLine() = %q; want %q", got, tt.want)
			}
		})
	}
}

// TestRenderResolveText covers every branch of RenderResolveText's grammar, exercised in the same
// order the grammar is spelled: the pre-resolution rejection branch, the listing branch, the glyph
// branch, and the reduced default branch, plus every totality guard each branch owns.
func TestRenderResolveText(t *testing.T) {
	sym1 := Symbol{ID: "pkg#Fn", Start: 1, End: 5, SigEnd: 2, Signature: "func Fn()"}
	sym2 := Symbol{ID: "pkg#Fn2", Start: 7, End: 7, Signature: "func Fn2()"}

	tests := []struct {
		name string
		r    ResolveResult
		want string
	}{
		{
			"RejectionWithReasonAndError",
			ResolveResult{Target: "#bad", Error: "engine: bad glyph", Reason: "no_unit"},
			"#bad error no_unit: engine: bad glyph\n",
		},
		{
			"RejectionEmptyReason",
			ResolveResult{Target: "#bad", Error: "engine: bad glyph"},
			"#bad error: engine: bad glyph\n",
		},
		{
			"RejectionEmptyError_TotalityGuard",
			ResolveResult{Target: "#bad"},
			"#bad error\n",
		},
		{
			"GlyphNotFoundUnitFound",
			ResolveResult{Target: "pkg#Missing", ID: "pkg#Missing", Status: StatusNotFound, Unit: StatusFound},
			"pkg#Missing not_found (unit found)\n",
		},
		{
			"GlyphNotFoundUnitNotFound",
			ResolveResult{Target: "pkg#Missing", ID: "pkg#Missing", Status: StatusNotFound, Unit: StatusNotFound},
			"pkg#Missing not_found (unit not_found)\n",
		},
		{
			"GlyphNotFoundEmptyUnit_TotalityGuard",
			ResolveResult{Target: "pkg#Missing", ID: "pkg#Missing", Status: StatusNotFound},
			"pkg#Missing not_found\n",
		},
		{
			"GlyphFound",
			ResolveResult{Target: "pkg#Fn", ID: "pkg#Fn", Status: StatusFound, Symbols: []Symbol{sym1}},
			"pkg#Fn found\n1-5 (sig 1-2) pkg#Fn: func Fn()\n",
		},
		{
			"GlyphMultipart",
			ResolveResult{Target: "pkg#init", ID: "pkg#init", Status: StatusMultipart, Symbols: []Symbol{sym1, sym2}},
			"pkg#init multipart\n1-5 (sig 1-2) pkg#Fn: func Fn()\n7-7 pkg#Fn2: func Fn2()\n",
		},
		{
			"GlyphAmbiguous",
			ResolveResult{Target: "pkg#Fn", ID: "pkg#Fn", Status: StatusAmbiguous, Candidates: []Symbol{sym1, sym2}},
			"pkg#Fn ambiguous\n1-5 (sig 1-2) pkg#Fn: func Fn()\n7-7 pkg#Fn2: func Fn2()\n",
		},
		{
			// Retargeted from the bare-path "PathFound" row: card 12 turns a bare path into a
			// no_separator rejection, so this is now the self glyph naming that same directory.
			"SelfFound",
			ResolveResult{Target: "pkg#", ID: "pkg#", Status: StatusFound, Listing: &DirAnswer{Dir: "pkg", Files: []FileEntry{{Name: "a.go"}}}},
			"pkg# found\npkg, 1 file\na.go\n",
		},
		{
			// Retargeted from "PathFoundOneFileEntry" the same way, for a self glyph naming a file.
			"SelfFoundOneFileEntry",
			ResolveResult{Target: "pkg/a.go#", ID: "pkg/a.go#", Status: StatusFound, Listing: &DirAnswer{Dir: "pkg", Files: []FileEntry{{Name: "a.go"}}}},
			"pkg/a.go# found\npkg, 1 file\na.go\n",
		},
		{
			"SelfNotFoundUnitFound",
			ResolveResult{Target: "pkg#", ID: "pkg#", Status: StatusNotFound, Unit: StatusFound},
			"pkg# not_found (unit found)\n",
		},
		{
			// A self glyph whose unit directory does not exist either — the path's own unit does not
			// exist.
			"SelfNotFoundUnitNotFound",
			ResolveResult{Target: "pkg#", ID: "pkg#", Status: StatusNotFound, Unit: StatusNotFound},
			"pkg# not_found (unit not_found)\n",
		},
		{
			// Guard: a hand-built not_found value carrying a non-nil Listing, a shape the engine never
			// produces. The listing branch's explicit Status == StatusFound guard means line 1 is
			// emitted alone, with no directory block under a negative status.
			"ListingBranchGuard_NotFoundWithListingEmitsLine1Only",
			ResolveResult{Target: "pkg#", ID: "pkg#", Status: StatusNotFound, Listing: &DirAnswer{Dir: "pkg", Files: []FileEntry{{Name: "a.go"}}}},
			"pkg# not_found\n",
		},
		{
			// Guard: a hand-built value carrying a Listing and an empty ID, a shape the engine never
			// produces since a found self glyph always carries an ID. Line 1 falls back to Target so it
			// never begins with a space.
			"ListingBranchGuard_EmptyIDFallsBackToTarget",
			ResolveResult{Target: "pkg", Status: StatusFound, Listing: &DirAnswer{Dir: "pkg", Files: []FileEntry{{Name: "a.go"}}}},
			"pkg found\npkg, 1 file\na.go\n",
		},
		{
			// Renamed from "PathNotFound": this is the external-caller guard for a hand-built value
			// with neither an ID nor a Listing, not a path answer — the engine can no longer produce a
			// path result at all. It reaches the reduced default arm unchanged.
			"DefaultGuard_NoIDNoListingNotFound",
			ResolveResult{Target: "missing", Status: StatusNotFound},
			"missing not_found\n",
		},
		{
			// Renamed from "PathFoundNilDir_TotalityGuard": the reduced default arm's own totality
			// guard, for a hand-built value with no ID and no Listing. Line 1 alone.
			"DefaultGuard_NoIDNoListingFound",
			ResolveResult{Target: "pkg", Status: StatusFound},
			"pkg found\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderResolveText(tt.r)
			if got != tt.want {
				t.Errorf("RenderResolveText() = %q; want %q", got, tt.want)
			}
			assertNoTrailingWhitespaceAndOneNewline(t, got)
		})
	}
}

// TestRenderExpandText covers every branch of RenderExpandText's grammar: not-found, found (with and
// without members, and the nil-head totality guard), ambiguous, and the fall-through default branch.
func TestRenderExpandText(t *testing.T) {
	head := Symbol{ID: "pkg#Thing", Start: 1, End: 3, Signature: "type Thing struct{}"}
	member := Symbol{ID: "pkg#Thing.Method", Start: 5, End: 5, Signature: "func (t Thing) Method()"}

	tests := []struct {
		name string
		a    ExpandAnswer
		want string
	}{
		{
			"NotFoundUnitFound",
			ExpandAnswer{ID: "pkg#Missing", Status: StatusNotFound, Unit: StatusFound},
			"pkg#Missing not_found (unit found)\n",
		},
		{
			"NotFoundUnitNotFound",
			ExpandAnswer{ID: "pkg#Missing", Status: StatusNotFound, Unit: StatusNotFound},
			"pkg#Missing not_found (unit not_found)\n",
		},
		{
			"FoundWithMembers",
			ExpandAnswer{ID: "pkg#Thing", Status: StatusFound, Head: &head, Members: []Symbol{member}},
			"pkg#Thing found\n1-3 pkg#Thing: type Thing struct{}\n\n5-5 pkg#Thing.Method: func (t Thing) Method()\n",
		},
		{
			"FoundNoMembers",
			ExpandAnswer{ID: "pkg#Thing", Status: StatusFound, Head: &head},
			"pkg#Thing found\n1-3 pkg#Thing: type Thing struct{}\n",
		},
		{
			"FoundNilHead_TotalityGuard",
			ExpandAnswer{ID: "pkg#Thing", Status: StatusFound},
			"pkg#Thing found\n",
		},
		{
			"Ambiguous",
			ExpandAnswer{ID: "pkg#Thing", Status: StatusAmbiguous, Candidates: []Symbol{head, member}},
			"pkg#Thing ambiguous\n1-3 pkg#Thing: type Thing struct{}\n5-5 pkg#Thing.Method: func (t Thing) Method()\n",
		},
		{
			"ZeroValue_TotalityGuard",
			ExpandAnswer{},
			"\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderExpandText(tt.a)
			if got != tt.want {
				t.Errorf("RenderExpandText() = %q; want %q", got, tt.want)
			}
			assertNoTrailingWhitespaceAndOneNewline(t, got)
		})
	}
}

// TestRenderText_ProseNormalisation uses docs/rewrite-plan.md §4's own "placement" example as the
// fixture: the one case the plan shows both sides of.
func TestRenderText_ProseNormalisation(t *testing.T) {
	doc := "placement is one resolved pane: its tmux pane id and the row height it\nhas been assigned."
	want := "placement is one resolved pane: its tmux pane id and the row height it has been assigned."
	if got := normalizeProse(doc); got != want {
		t.Errorf("normalizeProse(%q) = %q; want %q", doc, got, want)
	}
}

// TestRenderText_ProseNormalisation_SpacesAndTabsCollapse covers that a run of spaces and a tab
// collapse to one space and that nothing is truncated.
func TestRenderText_ProseNormalisation_SpacesAndTabsCollapse(t *testing.T) {
	doc := "a  b\tc   d"
	want := "a b c d"
	if got := normalizeProse(doc); got != want {
		t.Errorf("normalizeProse(%q) = %q; want %q", doc, got, want)
	}
}
