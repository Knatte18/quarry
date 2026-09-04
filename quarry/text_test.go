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
