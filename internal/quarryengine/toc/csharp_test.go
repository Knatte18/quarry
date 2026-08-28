// csharp_test.go covers the C# Strategy's Symbols, Header, Package, Generated, and TestFile methods
// over inline fixtures, following the helper shape golang_test.go established.

package toc

import (
	"testing"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/Knatte18/quarry/internal/quarryengine/treesitter"
)

// csharpExtraction is every C# Strategy method's output for one parsed fixture, gathered in a
// single treesitter.WithTree call so each table row stays one fixture plus its expectation.
type csharpExtraction struct {
	Symbols        []Symbol
	Header         string
	Package        string
	Partial        bool
	Generated      bool
	GeneratedKnown bool
}

// extractCSharp parses src as C# and runs every registered csharpStrategy method against the
// resulting tree, returning the combined result. It fails the test immediately if the "csharp"
// strategy is not registered or the parse itself errors — both would be a setup bug, not a case
// under test.
func extractCSharp(t *testing.T, src string) csharpExtraction {
	t.Helper()
	strategy, ok := StrategyFor("csharp")
	if !ok {
		t.Fatal(`StrategyFor("csharp") reported ok == false; want the registered C# strategy`)
	}
	var got csharpExtraction
	b := []byte(src)
	err := treesitter.WithTree("csharp", b, func(root *ts.Node, partial bool) error {
		got.Symbols = strategy.Symbols(root, b)
		got.Header = strategy.Header(root, b)
		got.Package = strategy.Package(root, b)
		got.Partial = partial
		got.Generated, got.GeneratedKnown = strategy.Generated(root, b)
		return nil
	})
	if err != nil {
		t.Fatalf("treesitter.WithTree(\"csharp\", ...) returned error: %v", err)
	}
	return got
}

// TestCSharpStrategy_Symbols covers Symbols across every declaration shape the C# strategy
// handles: XML doc stripping, both namespace forms, block- and expression-bodied members, the
// closed member-kind set, and container descent into nested types.
func TestCSharpStrategy_Symbols(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []Symbol
	}{
		{
			name: "ClassWithXMLDocCommentTagsStrippedTextKeptRangeStartsAtComment",
			src: "/// <summary>Foo does a thing.</summary>\n" +
				"/// <param name=\"x\">the value</param>\n" +
				"public class Foo\n" +
				"{\n" +
				"}\n",
			want: []Symbol{
				{
					Kind: KindType, Name: "Foo", Signature: "public class Foo",
					Docstring: "Foo does a thing.\nthe value",
					Start:     1, SigEnd: 4, End: 5,
				},
			},
		},
		{
			name: "MethodInsideFileScopedNamespaceInsideClassFoundWithOwner",
			src: "namespace App;\n" +
				"\n" +
				"public class Greeter\n" +
				"{\n" +
				"    public void Hello()\n" +
				"    {\n" +
				"    }\n" +
				"}\n",
			want: []Symbol{
				{Kind: KindType, Name: "Greeter", Signature: "public class Greeter", Start: 3, SigEnd: 4, End: 8},
				{Kind: KindMethod, Name: "Hello", Owner: "Greeter", Signature: "public void Hello()", Start: 5, SigEnd: 6, End: 7},
			},
		},
		{
			name: "MethodInsideBracedNamespaceInsideClassAlsoFound",
			src: "namespace App\n" +
				"{\n" +
				"    public class Greeter\n" +
				"    {\n" +
				"        public void Hello()\n" +
				"        {\n" +
				"        }\n" +
				"    }\n" +
				"}\n",
			want: []Symbol{
				{Kind: KindType, Name: "Greeter", Signature: "public class Greeter", Start: 3, SigEnd: 4, End: 8},
				{Kind: KindMethod, Name: "Hello", Owner: "Greeter", Signature: "public void Hello()", Start: 5, SigEnd: 6, End: 7},
			},
		},
		{
			name: "ExpressionBodiedMemberSignatureStopsBeforeArrowExpressionExcluded",
			src: "public class Calc\n" +
				"{\n" +
				"    public int Double(int x)\n" +
				"        => x * 2;\n" +
				"}\n",
			want: []Symbol{
				{Kind: KindType, Name: "Calc", Signature: "public class Calc", Start: 1, SigEnd: 2, End: 5},
				{Kind: KindMethod, Name: "Double", Owner: "Calc", Signature: "public int Double(int x)", Start: 3, SigEnd: 3, End: 4},
			},
		},
		{
			name: "MultiLineExpressionBodiedMemberArrowOnOwnLineSigEndIsLineBeforeArrow",
			src: "public class Calc\n" +
				"{\n" +
				"    public int Sum(\n" +
				"        int a,\n" +
				"        int b)\n" +
				"        => a + b;\n" +
				"}\n",
			want: []Symbol{
				{Kind: KindType, Name: "Calc", Signature: "public class Calc", Start: 1, SigEnd: 2, End: 7},
				{
					Kind: KindMethod, Name: "Sum", Owner: "Calc",
					Signature: "public int Sum(\n        int a,\n        int b)",
					Start:     3, SigEnd: 5, End: 6,
				},
			},
		},
		{
			name: "SingleLineExpressionBodiedMemberSigEndClampedToDeclarationLine",
			src: "public class Calc\n" +
				"{\n" +
				"    public int One() => 1;\n" +
				"}\n",
			want: []Symbol{
				{Kind: KindType, Name: "Calc", Signature: "public class Calc", Start: 1, SigEnd: 2, End: 4},
				{Kind: KindMethod, Name: "One", Owner: "Calc", Signature: "public int One()", Start: 3, SigEnd: 3, End: 3},
			},
		},
		{
			name: "BlockBodiedMemberMultiLineParameterListWholeSignatureSigEndIsOpeningBraceLine",
			src: "public class Calc\n" +
				"{\n" +
				"    public int Sum(\n" +
				"        int a,\n" +
				"        int b)\n" +
				"    {\n" +
				"        return a + b;\n" +
				"    }\n" +
				"}\n",
			want: []Symbol{
				{Kind: KindType, Name: "Calc", Signature: "public class Calc", Start: 1, SigEnd: 2, End: 9},
				{
					Kind: KindMethod, Name: "Sum", Owner: "Calc",
					Signature: "public int Sum(\n        int a,\n        int b)",
					Start:     3, SigEnd: 6, End: 8,
				},
			},
		},
		{
			name: "PositionalRecordNoBodyNoTrailingSemicolonSigEndZeroOmitted",
			src:  "public record Point(int X, int Y);\n",
			want: []Symbol{
				{Kind: KindType, Name: "Point", Signature: "public record Point(int X, int Y)", Start: 1, End: 1},
			},
		},
		{
			name: "InterfaceAndItsMethodBothListedInterfaceAsKindType",
			src: "public interface IGreeter\n" +
				"{\n" +
				"    void Hello();\n" +
				"}\n",
			want: []Symbol{
				{Kind: KindType, Name: "IGreeter", Signature: "public interface IGreeter", Start: 1, SigEnd: 2, End: 4},
				{Kind: KindMethod, Name: "Hello", Owner: "IGreeter", Signature: "void Hello()", Start: 3, End: 3},
			},
		},
		{
			name: "DeclarationInsideMethodBodyNotListed",
			src: "public class Outer\n" +
				"{\n" +
				"    public void M()\n" +
				"    {\n" +
				"        void Local() { }\n" +
				"    }\n" +
				"}\n",
			want: []Symbol{
				{Kind: KindType, Name: "Outer", Signature: "public class Outer", Start: 1, SigEnd: 2, End: 7},
				{Kind: KindMethod, Name: "M", Owner: "Outer", Signature: "public void M()", Start: 3, SigEnd: 4, End: 6},
			},
		},
		{
			name: "ClassWithOneOfEachEmittedMemberKindAllListedAsKindMethod",
			src: "public class C\n" +
				"{\n" +
				"    public C() { }\n" +
				"    ~C() { }\n" +
				"    public void M() { }\n" +
				"}\n",
			want: []Symbol{
				{Kind: KindType, Name: "C", Signature: "public class C", Start: 1, SigEnd: 2, End: 6},
				{Kind: KindMethod, Name: "C", Owner: "C", Signature: "public C()", Start: 3, SigEnd: 3, End: 3},
				{Kind: KindMethod, Name: "C", Owner: "C", Signature: "~C()", Start: 4, SigEnd: 4, End: 4},
				{Kind: KindMethod, Name: "M", Owner: "C", Signature: "public void M()", Start: 5, SigEnd: 5, End: 5},
			},
		},
		{
			name: "ClassWithEachExcludedMemberKindNoneListedClosedSetAssertion",
			src: "public class Widget\n" +
				"{\n" +
				"    public int Count { get; set; }\n" +
				"    private int x;\n" +
				"    public event System.EventHandler Changed;\n" +
				"    public delegate void Handler();\n" +
				"    public static Widget operator +(Widget a, Widget b) => a;\n" +
				"    public static explicit operator int(Widget w) => 0;\n" +
				"}\n",
			want: []Symbol{
				{Kind: KindType, Name: "Widget", Signature: "public class Widget", Start: 1, SigEnd: 2, End: 9},
			},
		},
		{
			name: "NestedClassInsideClassListedAsKindTypeNotAsAMember",
			src: "public class Outer\n" +
				"{\n" +
				"    public class Inner\n" +
				"    {\n" +
				"    }\n" +
				"}\n",
			want: []Symbol{
				{Kind: KindType, Name: "Outer", Signature: "public class Outer", Start: 1, SigEnd: 2, End: 6},
				{Kind: KindType, Name: "Inner", Signature: "public class Inner", Start: 3, SigEnd: 4, End: 5},
			},
		},
		{
			name: "MemberWithNoDocCommentDocstringEmptyRangeStartsAtDeclaration",
			src: "public class Plain\n" +
				"{\n" +
				"    public void M() { }\n" +
				"}\n",
			want: []Symbol{
				{Kind: KindType, Name: "Plain", Signature: "public class Plain", Start: 1, SigEnd: 2, End: 4},
				{Kind: KindMethod, Name: "M", Owner: "Plain", Signature: "public void M()", Start: 3, SigEnd: 3, End: 3},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCSharp(t, tt.src).Symbols
			assertSymbolsEqual(t, got, tt.want)
		})
	}
}

// TestCSharpStrategy_SymbolsAscendingByStart asserts Symbols returns several declarations of mixed
// kinds in ascending Start order, as the plan's ordering decision requires.
func TestCSharpStrategy_SymbolsAscendingByStart(t *testing.T) {
	src := "public class First\n" +
		"{\n" +
		"}\n" +
		"\n" +
		"public class Second\n" +
		"{\n" +
		"    public void M() { }\n" +
		"}\n"
	got := extractCSharp(t, src).Symbols
	if len(got) != 3 {
		t.Fatalf("Symbols() returned %d symbols; want 3", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].Start >= got[i].Start {
			t.Errorf("Symbols()[%d].Start = %d >= Symbols()[%d].Start = %d; want ascending order", i-1, got[i-1].Start, i, got[i].Start)
		}
	}
	wantNames := []string{"First", "Second", "M"}
	for i, name := range wantNames {
		if got[i].Name != name {
			t.Errorf("Symbols()[%d].Name = %q; want %q", i, got[i].Name, name)
		}
	}
}

// TestCSharpStrategy_Header covers Header's directive-skipping and first-non-directive-block
// selection, including the auto-generated-banner-then-real-header case where Generated must still
// report true from the skipped block.
func TestCSharpStrategy_Header(t *testing.T) {
	tests := []struct {
		name          string
		src           string
		wantHeader    string
		wantGenerated bool
	}{
		{
			name: "AutoGeneratedBannerThenRealHeaderHeaderIsSecondBlockGeneratedStillTrue",
			src: "// <auto-generated />\n" +
				"\n" +
				"/// <summary>Real header.</summary>\n" +
				"namespace App;\n",
			wantHeader:    "Real header.",
			wantGenerated: true,
		},
		{
			name:       "OnlyAXMLDocHeaderIsTheHeader",
			src:        "/// <summary>Only header.</summary>\nnamespace App;\n",
			wantHeader: "Only header.",
		},
		{
			name:       "NoLeadingCommentNoHeaderSymbolsStillReturned",
			src:        "namespace App;\n\npublic class C\n{\n}\n",
			wantHeader: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCSharp(t, tt.src)
			if got.Header != tt.wantHeader {
				t.Errorf("Header() = %q; want %q", got.Header, tt.wantHeader)
			}
			if got.Generated != tt.wantGenerated {
				t.Errorf("Generated() generated = %v; want %v", got.Generated, tt.wantGenerated)
			}
			if !got.GeneratedKnown {
				t.Error("Generated() known = false; want true for csharp")
			}
		})
	}
}

// TestCSharpStrategy_NoLeadingCommentStillReturnsSymbols asserts a file with no leading comment at
// all still returns its declarations, since Header's empty result must not affect Symbols.
func TestCSharpStrategy_NoLeadingCommentStillReturnsSymbols(t *testing.T) {
	got := extractCSharp(t, "namespace App;\n\npublic class C\n{\n}\n")
	if len(got.Symbols) == 0 {
		t.Error("Symbols() returned no symbols; want the class still returned")
	}
}

// TestCSharpStrategy_Package covers Package's file-scoped-namespace, braced-namespace, and
// no-namespace cases.
func TestCSharpStrategy_Package(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{name: "FileScopedNamespace", src: "namespace X.Y;\n\npublic class C\n{\n}\n", want: "X.Y"},
		{name: "BracedNamespace", src: "namespace X\n{\n    public class C\n    {\n    }\n}\n", want: "X"},
		{name: "NoNamespace", src: "public class C\n{\n}\n", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCSharp(t, tt.src).Package
			if got != tt.want {
				t.Errorf("Package() = %q; want %q", got, tt.want)
			}
		})
	}
}

// TestCSharpStrategy_TestFile asserts TestFile reports known == false for a "Tests.cs"-shaped name,
// the explicit-omission case, not merely a false answer.
func TestCSharpStrategy_TestFile(t *testing.T) {
	strategy, ok := StrategyFor("csharp")
	if !ok {
		t.Fatal(`StrategyFor("csharp") reported ok == false; want the registered C# strategy`)
	}
	gotIsTest, gotKnown := strategy.TestFile("FooTests.cs")
	if gotKnown {
		t.Error(`TestFile("FooTests.cs") known = true; want false for csharp`)
	}
	if gotIsTest {
		t.Error(`TestFile("FooTests.cs") isTest = true; want false alongside known = false`)
	}
}
