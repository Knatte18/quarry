// golang_test.go covers the Go Strategy's Symbols, Header, Package, Generated, and TestFile
// methods, and the parse-error tolerance every strategy inherits from treesitter.WithTree. Every
// fixture is an inline string constant parsed through treesitter.WithTree, so the package stays
// hermetic and parallel-safe.

package engine

import (
	"testing"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/Knatte18/quarry/internal/engine/treesitter"
)

// goExtraction is every Go Strategy method's output for one parsed fixture, gathered in a single
// treesitter.WithTree call so each table row stays one fixture plus its expectation.
type goExtraction struct {
	Symbols   []Symbol
	Header    string
	Package   string
	Partial   bool
	Generated bool
}

// extractGo parses src as Go and runs every registered goStrategy method against the resulting
// tree, returning the combined result. It fails the test immediately if the "go" strategy is not
// registered or the parse itself errors — both would be a setup bug, not a case under test.
func extractGo(t *testing.T, src string) goExtraction {
	t.Helper()
	strategy, ok := StrategyFor("go")
	if !ok {
		t.Fatal(`StrategyFor("go") reported ok == false; want the registered Go strategy`)
	}
	var got goExtraction
	b := []byte(src)
	err := treesitter.WithTree("go", b, func(root *ts.Node, partial bool) error {
		got.Symbols = strategy.Symbols(root, b)
		got.Header = strategy.Header(root, b)
		got.Package = strategy.Package(root, b)
		got.Partial = partial
		got.Generated = strategy.Generated(root, b)
		return nil
	})
	if err != nil {
		t.Fatalf("treesitter.WithTree(\"go\", ...) returned error: %v", err)
	}
	return got
}

// TestGoStrategy_Symbols covers Symbols across every declaration shape the Go strategy handles:
// docstring association, comment/blank-line adjacency, method receivers, body exclusion, and every
// type_declaration shape.
func TestGoStrategy_Symbols(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []Symbol
	}{
		{
			name: "FunctionWithDocstring",
			src: "package p\n" +
				"\n" +
				"// Foo does a thing.\n" +
				"func Foo() {}\n",
			want: []Symbol{
				{Kind: KindFunction, Name: "Foo", Signature: "func Foo()", Docstring: "Foo does a thing.", Start: 3, SigEnd: 4, End: 4},
			},
		},
		{
			name: "FunctionWithNoDocstring",
			src: "package p\n" +
				"\n" +
				"func Bar() {}\n",
			want: []Symbol{
				{Kind: KindFunction, Name: "Bar", Signature: "func Bar()", Start: 3, SigEnd: 3, End: 3},
			},
		},
		{
			name: "TrailingCommentSeparatedFromNextDeclByBlankLineIsNotMisattributed",
			src: "package p\n" +
				"\n" +
				"func A() {}\n" +
				"// trailing note about A\n" +
				"\n" +
				"func B() {}\n",
			want: []Symbol{
				{Kind: KindFunction, Name: "A", Signature: "func A()", Start: 3, SigEnd: 3, End: 3},
				{Kind: KindFunction, Name: "B", Signature: "func B()", Start: 6, SigEnd: 6, End: 6},
			},
		},
		{
			name: "CommentBlockSeparatedFromDeclarationByBlankLineIsNotADocstring",
			src: "package p\n" +
				"\n" +
				"// Not a docstring, separated.\n" +
				"\n" +
				"func C() {}\n",
			want: []Symbol{
				{Kind: KindFunction, Name: "C", Signature: "func C()", Start: 5, SigEnd: 5, End: 5},
			},
		},
		{
			name: "MethodOwnerStripsPointerStar",
			src: "package p\n" +
				"\n" +
				"type T struct{}\n" +
				"\n" +
				"func (t *T) Ptr() {}\n" +
				"\n" +
				"func (t T) Val() {}\n",
			want: []Symbol{
				{Kind: KindType, Name: "T", Signature: "type T struct", SigEnd: 3, Start: 3, End: 3},
				{Kind: KindMethod, Name: "Ptr", Owner: "T", Signature: "func (t *T) Ptr()", Start: 5, SigEnd: 5, End: 5},
				{Kind: KindMethod, Name: "Val", Owner: "T", Signature: "func (t T) Val()", Start: 7, SigEnd: 7, End: 7},
			},
		},
		{
			name: "DeclarationInsideFunctionBodyIsNotListed",
			src: "package p\n" +
				"\n" +
				"func Outer() {\n" +
				"\ttype inner int\n" +
				"\tfunc() {}()\n" +
				"}\n",
			want: []Symbol{
				{Kind: KindFunction, Name: "Outer", Signature: "func Outer()", Start: 3, SigEnd: 3, End: 6},
			},
		},
		{
			name: "MultiLineSignatureReturnedWhole",
			src: "package p\n" +
				"\n" +
				"func Multi(\n" +
				"\ta int,\n" +
				"\tb string,\n" +
				") error {\n" +
				"\treturn nil\n" +
				"}\n",
			want: []Symbol{
				{
					Kind: KindFunction, Name: "Multi",
					Signature: "func Multi(\n\ta int,\n\tb string,\n) error",
					Start:     3, SigEnd: 6, End: 8,
				},
			},
		},
		{
			name: "StructTypeSignatureStopsAtOpeningBrace",
			src: "package p\n" +
				"\n" +
				"type FileLock struct {\n" +
				"\tpath string\n" +
				"}\n",
			want: []Symbol{
				{Kind: KindType, Name: "FileLock", Signature: "type FileLock struct", Start: 3, SigEnd: 3, End: 5},
			},
		},
		{
			name: "BodylessTypeSignatureIsTheWholeSpec",
			src: "package p\n" +
				"\n" +
				"type ID string\n",
			want: []Symbol{
				{Kind: KindType, Name: "ID", Signature: "type ID string", Start: 3, End: 3},
			},
		},
		{
			name: "TypeAliasIsListedWithWholeSpecAsSignature",
			src: "package p\n" +
				"\n" +
				"type Alias = int\n",
			want: []Symbol{
				{Kind: KindType, Name: "Alias", Signature: "type Alias = int", Start: 3, End: 3},
			},
		},
		{
			name: "InterfaceTypeSignatureStopsAtOpeningBraceExcludingMethodSet",
			src: "package p\n" +
				"\n" +
				"type Reader interface {\n" +
				"\tRead() (int, error)\n" +
				"}\n",
			want: []Symbol{
				{Kind: KindType, Name: "Reader", Signature: "type Reader interface", Start: 3, SigEnd: 3, End: 5},
			},
		},
		{
			name: "GroupedTypeBlockOneSymbolPerSpecCommentOnGroupLineAttributedToNoSpec",
			src: "package p\n" +
				"\n" +
				"// Group doc, attaches to no spec.\n" +
				"type (\n" +
				"\t// X is a thing.\n" +
				"\tX int\n" +
				"\t// Y is another.\n" +
				"\tY struct {\n" +
				"\t\tZ int\n" +
				"\t}\n" +
				")\n",
			want: []Symbol{
				{Kind: KindType, Name: "X", Signature: "type X int", Docstring: "X is a thing.", Start: 5, End: 6},
				{Kind: KindType, Name: "Y", Signature: "type Y struct", Docstring: "Y is another.", Start: 7, SigEnd: 8, End: 10},
			},
		},
		{
			name: "SingleSpecGroupedBlockRangeAndSignatureAreTheSpecsOwnNotTheWholeGroup",
			src: "package p\n" +
				"\n" +
				"type (\n" +
				"\tW int\n" +
				")\n",
			want: []Symbol{
				{Kind: KindType, Name: "W", Signature: "type W int", Start: 4, End: 4},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractGo(t, tt.src).Symbols
			assertSymbolsEqual(t, got, tt.want)
		})
	}
}

// assertSymbolsEqual compares got against want element-by-element, reporting the full mismatched
// Symbol value rather than one field, per the plan's "assert the full Symbol value" requirement.
func assertSymbolsEqual(t *testing.T, got, want []Symbol) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Symbols() returned %d symbols %+v; want %d symbols %+v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Symbols()[%d] = %+v; want %+v", i, got[i], want[i])
		}
	}
}

// TestGoStrategy_SymbolsAscendingByStart asserts Symbols returns several declarations of mixed
// kinds in ascending Start order, as the plan's ordering decision requires.
func TestGoStrategy_SymbolsAscendingByStart(t *testing.T) {
	src := "package p\n" +
		"\n" +
		"func First() {}\n" +
		"\n" +
		"type Second int\n" +
		"\n" +
		"func Third() {}\n"
	got := extractGo(t, src).Symbols
	if len(got) != 3 {
		t.Fatalf("Symbols() returned %d symbols; want 3", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].Start >= got[i].Start {
			t.Errorf("Symbols()[%d].Start = %d >= Symbols()[%d].Start = %d; want ascending order", i-1, got[i-1].Start, i, got[i].Start)
		}
	}
	wantNames := []string{"First", "Second", "Third"}
	for i, name := range wantNames {
		if got[i].Name != name {
			t.Errorf("Symbols()[%d].Name = %q; want %q", i, got[i].Name, name)
		}
	}
}

// TestGoStrategy_SigEnd covers SigEnd's per-shape derivation: the single-line, multi-line, and
// bodyless cases, and the documented single-line-function imprecision.
func TestGoStrategy_SigEnd(t *testing.T) {
	tests := []struct {
		name       string
		src        string
		wantSigEnd []int
	}{
		{
			name: "DocstringPlusSingleLineSignatureSigEndIsTheSignatureLine",
			src: "package p\n" +
				"\n" +
				"// Foo does a thing.\n" +
				"func Foo() {\n" +
				"\treturn\n" +
				"}\n",
			wantSigEnd: []int{4},
		},
		{
			name: "MultiLineSignatureSigEndIsTheLastSignatureLineNotTheFirst",
			src: "package p\n" +
				"\n" +
				"func Multi(\n" +
				"\ta int,\n" +
				") error {\n" +
				"\treturn nil\n" +
				"}\n",
			wantSigEnd: []int{5},
		},
		{
			name: "StructBodySigEndIsTheOpeningBraceLine",
			src: "package p\n" +
				"\n" +
				"type S struct {\n" +
				"\tX int\n" +
				"}\n",
			wantSigEnd: []int{3},
		},
		{
			name: "TypeAliasAndBodylessDefinedTypeSigEndIsZeroAndOmitted",
			src: "package p\n" +
				"\n" +
				"type Alias = int\n" +
				"type ID string\n",
			wantSigEnd: []int{0, 0},
		},
		{
			name:       "SingleLineFunctionSigEndIncludesTheBody",
			src:        "package p\n\nfunc f() int { return 1 }\n",
			wantSigEnd: []int{3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractGo(t, tt.src).Symbols
			if len(got) != len(tt.wantSigEnd) {
				t.Fatalf("Symbols() returned %d symbols; want %d", len(got), len(tt.wantSigEnd))
			}
			for i, want := range tt.wantSigEnd {
				if got[i].SigEnd != want {
					t.Errorf("Symbols()[%d].SigEnd = %d; want %d", i, got[i].SigEnd, want)
				}
			}
		})
	}
}

// TestGoStrategy_Package covers Package's package_clause lookup and its absent-clause fallback.
func TestGoStrategy_Package(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{name: "NormalFile", src: "package mypkg\n\nfunc F() {}\n", want: "mypkg"},
		{name: "NoPackageClause", src: "func F() {}\n", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractGo(t, tt.src).Package
			if got != tt.want {
				t.Errorf("Package() = %q; want %q", got, tt.want)
			}
		})
	}
}

// TestGoStrategy_Header covers Header's directive-skipping and first-non-directive-block selection,
// including its two deliberate departures from docstring association: tolerating a blank line, and
// taking the first block rather than the one adjacent to package.
func TestGoStrategy_Header(t *testing.T) {
	tests := []struct {
		name          string
		src           string
		wantHeader    string
		wantGenerated bool
	}{
		{
			name: "HeaderSeparatedFromPackageByBlankLineIsStillFound",
			src: "// Package p does things.\n" +
				"\n" +
				"package p\n",
			wantHeader: "Package p does things.",
		},
		{
			name: "FileHeaderAndSeparatePackageDocCommentTheFirstBlockWins",
			src: "// File header describing this file.\n" +
				"\n" +
				"// Package p does things.\n" +
				"package p\n",
			wantHeader: "File header describing this file.",
		},
		{
			name:       "NoHeaderAtAllHeaderIsEmptyAndSymbolsStillReturned",
			src:        "package p\n\nfunc F() {}\n",
			wantHeader: "",
		},
		{
			name: "BuildConstraintBlankLineThenHeaderConstraintNotReturned",
			src: "//go:build linux\n" +
				"\n" +
				"// Real header text.\n" +
				"package p\n",
			wantHeader: "Real header text.",
		},
		{
			name: "OnlyLeadingBlockIsABuildConstraintNoHeader",
			src: "//go:build linux\n" +
				"package p\n",
			wantHeader: "",
		},
		{
			name: "BlockMixingGoGenerateWithProseIsTreatedAsHeaderNotSkipped",
			src: "//go:generate mockgen -source=foo.go\n" +
				"// This explains what the mock is for.\n" +
				"package p\n",
			wantHeader: "go:generate mockgen -source=foo.go\nThis explains what the mock is for.",
		},
		{
			name: "GeneratedBannerThenRealHeaderBannerSkippedAsHeaderAndGeneratedReportsTrue",
			src: "// Code generated by mockgen. DO NOT EDIT.\n" +
				"\n" +
				"// Real header text.\n" +
				"package p\n",
			wantHeader:    "Real header text.",
			wantGenerated: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractGo(t, tt.src)
			if got.Header != tt.wantHeader {
				t.Errorf("Header() = %q; want %q", got.Header, tt.wantHeader)
			}
			if got.Generated != tt.wantGenerated {
				t.Errorf("Generated() generated = %v; want %v", got.Generated, tt.wantGenerated)
			}
		})
	}
}

// TestGoStrategy_TestFile covers TestFile's delegation to TestFileByName("go", ...).
func TestGoStrategy_TestFile(t *testing.T) {
	strategy, ok := StrategyFor("go")
	if !ok {
		t.Fatal(`StrategyFor("go") reported ok == false; want the registered Go strategy`)
	}
	tests := []struct {
		name       string
		base       string
		wantIsTest bool
	}{
		{name: "TestSuffix", base: "foo_test.go", wantIsTest: true},
		{name: "NonTest", base: "foo.go", wantIsTest: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIsTest := strategy.TestFile(tt.base)
			if gotIsTest != tt.wantIsTest {
				t.Errorf("TestFile(%q) isTest = %v; want %v", tt.base, gotIsTest, tt.wantIsTest)
			}
		})
	}
}

// TestGoStrategy_PartialParseIsLossyNotJustIncomplete asserts a broken declaration that swallows a
// later valid one still reports partial == true and still returns the symbols the recovery managed
// to keep, documenting that recovery is lossy rather than merely incomplete.
func TestGoStrategy_PartialParseIsLossyNotJustIncomplete(t *testing.T) {
	src := "package p\n" +
		"\n" +
		"func Broken(\n" +
		"\n" +
		"func Recovered() {}\n"
	got := extractGo(t, src)
	if !got.Partial {
		t.Fatal("partial = false for a deliberately broken fixture; want true")
	}
	if len(got.Symbols) == 0 {
		t.Error("Symbols() returned no symbols for a partial parse; want the surviving symbols returned, lossy but not empty")
	}
}
