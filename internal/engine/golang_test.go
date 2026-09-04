// golang_test.go covers the Go Strategy's Symbols, Header, Package, Generated, and TestFile
// methods, and the parse-error tolerance every strategy inherits from treesitter.WithTree. Every
// fixture is an inline string constant parsed through treesitter.WithTree, so the package stays
// hermetic and parallel-safe.

package engine

import (
	"reflect"
	"testing"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/Knatte18/quarry/glyph"
	"github.com/Knatte18/quarry/internal/engine/treesitter"
)

// testUnit is the fixed glyph unit every fixture in this file is extracted under. Its own
// spellability is never in question here — this file calls Strategy.Symbols directly, bypassing
// the walk's unitFor/unitSpellable gate entirely, so a fixture's expected id is always
// testUnit + "#" + member.
const testUnit = "u"

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
		got.Symbols = strategy.Symbols(testUnit, root, b)
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

// id builds the expected glyph id for a package-level name under testUnit.
func id(name string) string {
	return testUnit + "#" + name
}

// methodID builds the expected glyph id for a method or interface method under testUnit.
func methodID(owner, name string) string {
	g := glyph.Glyph{Lang: glyph.Go, Unit: testUnit, Owner: []string{owner}, Name: name}
	return g.String()
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
				{Kind: KindFunction, ID: id("Foo"), Signature: "func Foo()", Doc: "Foo does a thing.", Start: 3, SigEnd: 4, End: 4},
			},
		},
		{
			name: "FunctionWithNoDocstring",
			src: "package p\n" +
				"\n" +
				"func Bar() {}\n",
			want: []Symbol{
				{Kind: KindFunction, ID: id("Bar"), Signature: "func Bar()", Start: 3, SigEnd: 3, End: 3},
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
				{Kind: KindFunction, ID: id("A"), Signature: "func A()", Start: 3, SigEnd: 3, End: 3},
				{Kind: KindFunction, ID: id("B"), Signature: "func B()", Start: 6, SigEnd: 6, End: 6},
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
				{Kind: KindFunction, ID: id("C"), Signature: "func C()", Start: 5, SigEnd: 5, End: 5},
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
				{Kind: KindType, ID: id("T"), Signature: "type T struct", SigEnd: 3, Start: 3, End: 3, HeadStart: 3, HeadEnd: 3},
				{Kind: KindMethod, ID: methodID("T", "Ptr"), Signature: "func (t *T) Ptr()", Start: 5, SigEnd: 5, End: 5},
				{Kind: KindMethod, ID: methodID("T", "Val"), Signature: "func (t T) Val()", Start: 7, SigEnd: 7, End: 7},
			},
		},
		{
			name: "GenericReceiverOwnerStripsTypeParameters",
			src: "package p\n" +
				"\n" +
				"type Box[T any] struct{ v T }\n" +
				"\n" +
				"func (b *Box[T]) Ptr() {}\n" +
				"\n" +
				"func (b Box[T]) Val() {}\n",
			want: []Symbol{
				{Kind: KindType, ID: id("Box"), Signature: "type Box[T any] struct", SigEnd: 3, Start: 3, End: 3, HeadStart: 3, HeadEnd: 3},
				{Kind: KindMethod, ID: methodID("Box", "Ptr"), Signature: "func (b *Box[T]) Ptr()", Start: 5, SigEnd: 5, End: 5},
				{Kind: KindMethod, ID: methodID("Box", "Val"), Signature: "func (b Box[T]) Val()", Start: 7, SigEnd: 7, End: 7},
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
				{Kind: KindFunction, ID: id("Outer"), Signature: "func Outer()", Start: 3, SigEnd: 3, End: 6},
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
					Kind: KindFunction, ID: id("Multi"),
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
				{Kind: KindType, ID: id("FileLock"), Signature: "type FileLock struct", Start: 3, SigEnd: 3, End: 5, HeadStart: 3, HeadEnd: 5},
			},
		},
		{
			name: "BodylessTypeSignatureIsTheWholeSpec",
			src: "package p\n" +
				"\n" +
				"type ID string\n",
			want: []Symbol{
				{Kind: KindType, ID: id("ID"), Signature: "type ID string", Start: 3, End: 3, HeadStart: 3, HeadEnd: 3},
			},
		},
		{
			name: "TypeAliasIsListedWithWholeSpecAsSignature",
			src: "package p\n" +
				"\n" +
				"type Alias = int\n",
			want: []Symbol{
				{Kind: KindType, ID: id("Alias"), Signature: "type Alias = int", Start: 3, End: 3, HeadStart: 3, HeadEnd: 3},
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
				{Kind: KindType, ID: id("Reader"), Signature: "type Reader interface", Start: 3, SigEnd: 3, End: 5, HeadStart: 3, HeadEnd: 5},
				{Kind: KindMethod, ID: methodID("Reader", "Read"), Signature: "Read() (int, error)", Start: 4, End: 4},
			},
		},
		{
			name: "InterfaceMethodDocstringAndAnonymousEmbeddedContributeNoExtraSymbol",
			src: "package p\n" +
				"\n" +
				"type Iface interface {\n" +
				"\t// M1 has a doc.\n" +
				"\tM1(x int) error\n" +
				"\tM2()\n" +
				"\tEmbedded\n" +
				"\tinterface{ Anon() }\n" +
				"}\n",
			want: []Symbol{
				{Kind: KindType, ID: id("Iface"), Signature: "type Iface interface", Start: 3, SigEnd: 3, End: 9, HeadStart: 3, HeadEnd: 9},
				{Kind: KindMethod, ID: methodID("Iface", "M1"), Signature: "M1(x int) error", Doc: "M1 has a doc.", Start: 4, End: 5},
				{Kind: KindMethod, ID: methodID("Iface", "M2"), Signature: "M2()", Start: 6, End: 6},
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
				{Kind: KindType, ID: id("X"), Signature: "type X int", Doc: "X is a thing.", Start: 5, End: 6, HeadStart: 5, HeadEnd: 6},
				{Kind: KindType, ID: id("Y"), Signature: "type Y struct", Doc: "Y is another.", Start: 7, SigEnd: 8, End: 10, HeadStart: 7, HeadEnd: 10},
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
				{Kind: KindType, ID: id("W"), Signature: "type W int", Start: 4, End: 4, HeadStart: 4, HeadEnd: 4},
			},
		},
		{
			name: "BlankIdentifierNeverListed",
			src: "package p\n" +
				"\n" +
				"func _() {}\n" +
				"\n" +
				"func (t T) _() {}\n" +
				"\n" +
				"type _ struct{}\n" +
				"\n" +
				"type T struct{}\n",
			want: []Symbol{
				{Kind: KindType, ID: id("T"), Signature: "type T struct", SigEnd: 9, Start: 9, End: 9, HeadStart: 9, HeadEnd: 9},
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
//
// Comparison uses reflect.DeepEqual, never ==: Symbol carries a Glyph field whose Owner is a
// slice, which makes Symbol itself non-comparable with ==. The Glyph field is zeroed on a copy of
// got before comparing, since it is redundant with ID (which every "want" table asserts directly)
// and spelling out a full glyph.Glyph literal per table row would add no coverage over asserting ID.
func assertSymbolsEqual(t *testing.T, got, want []Symbol) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Symbols() returned %d symbols %+v; want %d symbols %+v", len(got), got, len(want), want)
	}
	for i := range want {
		gotCompare := got[i]
		gotCompare.Glyph = glyph.Glyph{}
		if !reflect.DeepEqual(gotCompare, want[i]) {
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
	wantIDs := []string{id("First"), id("Second"), id("Third")}
	for i, want := range wantIDs {
		if got[i].ID != want {
			t.Errorf("Symbols()[%d].ID = %q; want %q", i, got[i].ID, want)
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
