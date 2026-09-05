// text_test.go pins RenderText's grammar, per _mill/discussion.md's text-view-grammar decision, to
// the character, over hand-built DirAnswer values only — no filesystem.

package quarry

import (
	"encoding/json"
	"strconv"
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

// TestRenderDeltaText asserts the exact rendered string for a hand-built GitDeltaAnswer exercising
// every section RenderDeltaText's grammar defines: a file echo carrying each disposition and both
// lossy flags, a created symbol, a deleted symbol, a modified entry naming several changed dimensions
// with a multi-occurrence before and after array, a renamed pair, and a candidate entry with every
// signal. It also asserts the two invariants RenderDeltaText promises: no trailing whitespace on any
// line and exactly one closing newline.
func TestRenderDeltaText(t *testing.T) {
	to := "def456"
	a := GitDeltaAnswer{
		From: "abc123",
		To:   &to,
		DeltaAnswer: DeltaAnswer{
			Files: []DeltaFile{
				{Path: "pkg/added.go", Disposition: DispositionAdded},
				{Path: "pkg/removed.go", Disposition: DispositionRemoved},
				{Path: "pkg/a.go", Disposition: DispositionChanged, LossyBefore: true, LossyAfter: true},
				{Path: "pkg/bad.go", Disposition: DispositionError, Error: "line one\nline two"},
			},
			Created: []Symbol{
				{ID: "pkg#New", Kind: KindFunction, File: "pkg/a.go", Start: 1, End: 2, Signature: "func New()"},
			},
			Deleted: []Symbol{
				{ID: "pkg#Gone", Kind: KindFunction, File: "pkg/a.go", Start: 3, End: 4, Signature: "func Gone()"},
			},
			Modified: []ModifiedSymbol{
				{
					ID:      "pkg#init",
					Kind:    KindFunction,
					Changed: []ChangedDimension{ChangedBody, ChangedSignature, ChangedDoc, ChangedFile},
					Before: []SymbolLocation{
						{File: "pkg/a.go", Start: 5, SigEnd: 6, End: 7},
						{File: "pkg/a.go", Start: 20, End: 21},
					},
					After: []Symbol{
						{ID: "pkg#init", Kind: KindFunction, File: "pkg/b.go", Start: 9, End: 12, SigEnd: 10, Signature: "func init()"},
						{ID: "pkg#init", Kind: KindFunction, File: "pkg/b.go", Start: 30, End: 31, Signature: "func init()"},
					},
				},
			},
			Renamed: []RenamedPair{
				{
					From: Symbol{ID: "pkg#OldName", Kind: KindFunction, File: "pkg/a.go", Start: 8, End: 9, Signature: "func OldName()"},
					To:   Symbol{ID: "pkg#NewName", Kind: KindFunction, File: "pkg/a.go", Start: 10, End: 11, Signature: "func NewName()"},
				},
			},
			RenameCandidates: []RenameCandidateEntry{
				{
					ID:   "pkg#Deleted2",
					Kind: KindFunction,
					Candidates: []RenameCandidate{
						{
							ID:   "pkg#Created2",
							Kind: KindFunction,
							File: "pkg/a.go",
							Signals: RenameSignals{
								SignatureIdenticalModuloName: true,
								BodyTokenSimilarity:          0.75,
								BodyTokensBefore:             10,
								BodyTokensAfter:              12,
								DocIdentical:                 false,
							},
						},
					},
				},
			},
		},
	}

	want := "from abc123 to def456\n" +
		"\n" +
		"files:\n" +
		"pkg/added.go added\n" +
		"pkg/removed.go removed\n" +
		"pkg/a.go changed [lossy_before] [lossy_after]\n" +
		"pkg/bad.go error [error line one line two]\n" +
		"\n" +
		"created:\n" +
		"function pkg/a.go:1-2 pkg#New: func New()\n" +
		"\n" +
		"deleted:\n" +
		"function pkg/a.go:3-4 pkg#Gone: func Gone()\n" +
		"\n" +
		"modified:\n" +
		"pkg#init function changed:body,signature,doc,file\n" +
		"before pkg/a.go:5-7 (sig 5-6)\n" +
		"before pkg/a.go:20-21\n" +
		"after function pkg/b.go:9-12 (sig 9-10) pkg#init: func init()\n" +
		"after function pkg/b.go:30-31 pkg#init: func init()\n" +
		"\n" +
		"renamed:\n" +
		"from function pkg/a.go:8-9 pkg#OldName: func OldName()\n" +
		"to function pkg/a.go:10-11 pkg#NewName: func NewName()\n" +
		"\n" +
		"rename_candidates:\n" +
		"pkg#Deleted2 function\n" +
		"candidate function pkg#Created2 pkg/a.go signals sig_identical=true body_similarity=0.75 body_tokens_before=10 body_tokens_after=12 doc_identical=false\n"

	got := RenderDeltaText(a)
	if got != want {
		t.Errorf("RenderDeltaText() = %q; want %q", got, want)
	}
	assertNoTrailingWhitespaceAndOneNewline(t, got)
}

// TestRenderDeltaText_Lossless asserts losslessness structurally rather than by eye: every value
// present in the JSON view of a fully populated GitDeltaAnswer appears somewhere in the text view,
// including each signal's value and each changed dimension's word. It also covers the case that makes
// the kind load-bearing — a const replaced by a var of the same name, giving one created and one
// deleted symbol with an identical identifier and differing kinds, asserted to render as two
// distinguishable records — a case whose doc text spans several lines and contains runs of whitespace,
// asserted collapsed to single spaces, and a case whose after-side revision is the working tree,
// asserted to render the explicit word rather than an empty field.
func TestRenderDeltaText_Lossless(t *testing.T) {
	a := GitDeltaAnswer{
		From: "abc123",
		// To is nil: the after side is the working tree.
		DeltaAnswer: DeltaAnswer{
			Files: []DeltaFile{
				{Path: "pkg/a.go", Disposition: DispositionChanged, LossyBefore: true, LossyAfter: true},
			},
			// A const replaced by a var of the same identifier: one created and one deleted symbol
			// sharing an ID, differing only in Kind. Without the kind prefix these two lines would be
			// indistinguishable.
			Created: []Symbol{
				{ID: "pkg#Same", Kind: KindVar, File: "pkg/a.go", Start: 1, End: 1, Signature: "var Same = 1",
					Doc: "Same is a value.\nIt   has   whitespace."},
			},
			Deleted: []Symbol{
				{ID: "pkg#Same", Kind: KindConst, File: "pkg/a.go", Start: 2, End: 2, Signature: "const Same = 1"},
			},
			Modified: []ModifiedSymbol{
				{
					ID:      "pkg#Changed",
					Kind:    KindFunction,
					Changed: []ChangedDimension{ChangedBody},
					Before:  []SymbolLocation{{File: "pkg/a.go", Start: 3, End: 4}},
					After:   []Symbol{{ID: "pkg#Changed", Kind: KindFunction, File: "pkg/a.go", Start: 3, End: 5, Signature: "func Changed()"}},
				},
			},
			Renamed: []RenamedPair{
				{
					From: Symbol{ID: "pkg#OldName", Kind: KindFunction, File: "pkg/a.go", Start: 8, End: 9, Signature: "func OldName()"},
					To:   Symbol{ID: "pkg#NewName", Kind: KindFunction, File: "pkg/a.go", Start: 10, End: 11, Signature: "func NewName()"},
				},
			},
			RenameCandidates: []RenameCandidateEntry{
				{
					ID:   "pkg#Deleted2",
					Kind: KindFunction,
					Candidates: []RenameCandidate{
						{
							ID:   "pkg#Created2",
							Kind: KindFunction,
							File: "pkg/a.go",
							Signals: RenameSignals{
								SignatureIdenticalModuloName: true,
								BodyTokenSimilarity:          0.625,
								BodyTokensBefore:             7,
								BodyTokensAfter:              9,
								DocIdentical:                 true,
							},
						},
					},
				},
			},
		},
	}

	jsonBytes, err := RenderDeltaJSON(a)
	if err != nil {
		t.Fatalf("RenderDeltaJSON() error = %v", err)
	}
	text := RenderDeltaText(a)
	assertNoTrailingWhitespaceAndOneNewline(t, text)

	if !strings.Contains(text, "working tree") {
		t.Errorf("RenderDeltaText() = %q; want the explicit \"working tree\" word for a nil To", text)
	}

	// Both the created var and the deleted const must be distinguishable: their shared identifier
	// each appears once per kind word.
	if !strings.Contains(text, "var pkg/a.go:1-1 pkg#Same") {
		t.Errorf("RenderDeltaText() = %q; want the created entry's kind (var) to distinguish it", text)
	}
	if !strings.Contains(text, "const pkg/a.go:2-2 pkg#Same") {
		t.Errorf("RenderDeltaText() = %q; want the deleted entry's kind (const) to distinguish it", text)
	}

	// The multi-line, whitespace-heavy doc collapses to single spaces, losing no words.
	if !strings.Contains(text, "Same is a value. It has whitespace.") {
		t.Errorf("RenderDeltaText() = %q; want the doc collapsed to single spaces with no words dropped", text)
	}

	// Every JSON-carried value must appear somewhere in the text view, decoded generically rather
	// than compared against a hand-written string, so this assertion cannot silently drift from the
	// answer's own shape.
	var decoded map[string]any
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(RenderDeltaJSON()) error = %v", err)
	}
	for _, v := range collectJSONScalars(decoded) {
		if !strings.Contains(text, v) {
			t.Errorf("RenderDeltaText() = %q; missing JSON-carried value %q", text, v)
		}
	}
}

// collectJSONScalars walks a decoded JSON value (as produced by json.Unmarshal into `any`) and
// returns the string form of every scalar leaf it holds: strings verbatim, numbers formatted with
// strconv.FormatFloat('f', -1, 64) exactly as RenderDeltaText's own signals line does, and bools as
// "true"/"false". Keys, nulls and structural values contribute nothing, since a text view is not
// required to spell a JSON key verbatim — only every carried value.
func collectJSONScalars(v any) []string {
	switch val := v.(type) {
	case string:
		if val == "" {
			return nil
		}
		// Normalised the same way RenderDeltaText itself normalises every doc, signature and error
		// value before printing: a raw multi-line or whitespace-heavy JSON string, such as a
		// docstring, is never expected to survive the text view's own newlines and runs of
		// whitespace collapsed to single spaces, so this check compares against the same normalised
		// form the renderer actually emits.
		return []string{normalizeProse(val)}
	case float64:
		return []string{strconv.FormatFloat(val, 'f', -1, 64)}
	case bool:
		return []string{strconv.FormatBool(val)}
	case map[string]any:
		var out []string
		for _, child := range val {
			out = append(out, collectJSONScalars(child)...)
		}
		return out
	case []any:
		var out []string
		for _, child := range val {
			out = append(out, collectJSONScalars(child)...)
		}
		return out
	default:
		return nil
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
