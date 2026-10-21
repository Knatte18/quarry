// csharp_test.go covers the C# Strategy: the symbols it mints per declaration shape, the ids it drops, and the file-level facts it reports.

package engine

import (
	"errors"
	"testing"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/Knatte18/quarry/glyph"
	"github.com/Knatte18/quarry/internal/engine/treesitter"
)

// csharpExtraction is everything the C# strategy reports about one source.
type csharpExtraction struct {
	Symbols    []Symbol
	Dropped    []DroppedSymbol
	Header     string
	Package    string
	PackageDoc string
	Partial    bool
	Generated  bool
}

// extractCSharp parses src as C# and runs every registered csharpStrategy method against the tree.
// It fails the test immediately when the "csharp" strategy is not registered or the parse errors.
func extractCSharp(t *testing.T, src string) csharpExtraction {
	t.Helper()
	strategy, ok := StrategyFor("csharp")
	if !ok {
		t.Fatal(`StrategyFor("csharp") reported ok == false; want the registered C# strategy`)
	}
	var got csharpExtraction
	b := []byte(src)
	err := treesitter.WithTree("csharp", b, func(root *ts.Node, partial bool) error {
		got.Symbols, got.Dropped = strategy.Symbols("ignored/unit", root, b)
		got.Header = strategy.Header(root, b)
		got.Package = strategy.Package(root, b)
		got.PackageDoc = strategy.PackageDoc(root, b)
		got.Partial = partial
		got.Generated = strategy.Generated(root, b)
		return nil
	})
	if err != nil {
		t.Fatalf("treesitter.WithTree(\"csharp\", ...) returned error: %v", err)
	}
	return got
}

// csharpWant is the expected shape of one minted Symbol.
type csharpWant struct {
	id        string
	kind      Kind
	start     int
	end       int
	doc       string
	headStart int
	headEnd   int
	partial   bool
}

func TestCSharpStrategy_Symbols(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		src  string
		want []csharpWant
	}{
		{
			name: "TypeKindsAndEnumMembers",
			src: "namespace N\n" +
				"{\n" +
				"    public class C { }\n" +
				"    public struct S { }\n" +
				"    public interface I { }\n" +
				"    public enum E { A, B = 2 }\n" +
				"    public record R(int X);\n" +
				"    public record struct RS(int Z);\n" +
				"    public delegate void D(int a);\n" +
				"}\n",
			want: []csharpWant{
				{id: "N#C", kind: KindType, start: 3, end: 3, headStart: 3, headEnd: 3},
				{id: "N#S", kind: KindType, start: 4, end: 4, headStart: 4, headEnd: 4},
				{id: "N#I", kind: KindType, start: 5, end: 5, headStart: 5, headEnd: 5},
				{id: "N#E", kind: KindType, start: 6, end: 6, headStart: 6, headEnd: 6},
				{id: "N#E.A", kind: KindConst, start: 6, end: 6},
				{id: "N#E.B", kind: KindConst, start: 6, end: 6},
				{id: "N#R", kind: KindType, start: 7, end: 7, headStart: 7, headEnd: 7},
				{id: "N#R.X", kind: KindProperty, start: 7, end: 7},
				{id: "N#RS", kind: KindType, start: 8, end: 8, headStart: 8, headEnd: 8},
				{id: "N#RS.Z", kind: KindProperty, start: 8, end: 8},
				{id: "N#D", kind: KindType, start: 9, end: 9, headStart: 9, headEnd: 9},
			},
		},
		{
			name: "DocCommentForms",
			src: "namespace N\n" +
				"{\n" +
				"    /// <summary>xml</summary>\n" +
				"    /// <remarks>r</remarks>\n" +
				"    public class A { }\n" +
				"    // plain\n" +
				"    // doc\n" +
				"    public class B { }\n" +
				"    /* block\n" +
				"     * doc */\n" +
				"    public class C { }\n" +
				"}\n",
			want: []csharpWant{
				{id: "N#A", kind: KindType, start: 3, end: 5, doc: "<summary>xml</summary>\n<remarks>r</remarks>", headStart: 3, headEnd: 5},
				{id: "N#B", kind: KindType, start: 6, end: 8, doc: "plain\ndoc", headStart: 6, headEnd: 8},
				{id: "N#C", kind: KindType, start: 9, end: 11, doc: "block\ndoc", headStart: 9, headEnd: 11},
			},
		},
		{
			name: "MemberKinds",
			src: "namespace N\n" +
				"{\n" +
				"    public class C\n" +
				"    {\n" +
				"        public C(int a) { }\n" +
				"        static C() { }\n" +
				"        ~C() { }\n" +
				"        public void M() { }\n" +
				"        public int P { get; set; }\n" +
				"        public int Q => 1;\n" +
				"        public int F;\n" +
				"        public const int K = 1, L = 2;\n" +
				"        public event System.Action Ev;\n" +
				"        public event System.Action Ev2 { add { } remove { } }\n" +
				"        public int this[int i] => i;\n" +
				"        public static C operator +(C a, C b) => a;\n" +
				"        public static implicit operator int(C c) => 0;\n" +
				"    }\n" +
				"}\n",
			want: []csharpWant{
				{id: "N#C", kind: KindType, start: 3, end: 18, headStart: 3, headEnd: 3},
				{id: "N#C.C(int)", kind: KindConstructor, start: 5, end: 5},
				{id: "N#C.static C()", kind: KindConstructor, start: 6, end: 6},
				{id: "N#C.~C()", kind: KindMethod, start: 7, end: 7},
				{id: "N#C.M()", kind: KindMethod, start: 8, end: 8},
				{id: "N#C.P", kind: KindProperty, start: 9, end: 9},
				{id: "N#C.Q", kind: KindProperty, start: 10, end: 10},
				{id: "N#C.F", kind: KindField, start: 11, end: 11},
				{id: "N#C.K", kind: KindConst, start: 12, end: 12},
				{id: "N#C.L", kind: KindConst, start: 12, end: 12},
				{id: "N#C.Ev", kind: KindEvent, start: 13, end: 13},
				{id: "N#C.Ev2", kind: KindEvent, start: 14, end: 14},
				{id: "N#C.this[int]", kind: KindIndexer, start: 15, end: 15},
				{id: "N#C.operator +(C,C)", kind: KindOperator, start: 16, end: 16},
				{id: "N#C.implicit operator int(C)", kind: KindOperator, start: 17, end: 17},
			},
		},
		{
			name: "GenericsOverloadsAndParameterSpelling",
			src: "namespace N\n" +
				"{\n" +
				"    public class Box<T>\n" +
				"    {\n" +
				"        public void M(int a) { }\n" +
				"        public void M(ref int a) { }\n" +
				"        public void M(params string[] r) { }\n" +
				"        public void G<U>(U u, global::System.String s) { }\n" +
				"        public void H(in int a, out int b, (int x, string y) t, int? n, List<int> l) { b = 0; }\n" +
				"        public class Inner<A, B> { public void X() { } }\n" +
				"    }\n" +
				"}\n",
			want: []csharpWant{
				{id: "N#Box<>", kind: KindType, start: 3, end: 11, headStart: 3, headEnd: 3},
				{id: "N#Box<>.M(int)", kind: KindMethod, start: 5, end: 5},
				{id: "N#Box<>.M(ref int)", kind: KindMethod, start: 6, end: 6},
				{id: "N#Box<>.M(params string[])", kind: KindMethod, start: 7, end: 7},
				{id: "N#Box<>.G<>(U,string)", kind: KindMethod, start: 8, end: 8},
				{id: "N#Box<>.H(in int,out int,(int,string),int?,List<int>)", kind: KindMethod, start: 9, end: 9},
				{id: "N#Box<>.Inner<,>", kind: KindType, start: 10, end: 10, headStart: 10, headEnd: 10},
				{id: "N#Box<>.Inner<,>.X()", kind: KindMethod, start: 10, end: 10},
			},
		},
		{
			name: "OperatorForms",
			src: "namespace N\n" +
				"{\n" +
				"    public class Box\n" +
				"    {\n" +
				"        public static bool operator <(Box a, Box b) => true;\n" +
				"        public static bool operator >(Box a, Box b) => true;\n" +
				"        public static Box operator >>(Box a, int b) => a;\n" +
				"        public static Box operator checked +(Box a, Box b) => a;\n" +
				"        public static explicit operator System.Text.Rune(Box b) => default;\n" +
				"    }\n" +
				"}\n",
			want: []csharpWant{
				{id: "N#Box", kind: KindType, start: 3, end: 10, headStart: 3, headEnd: 3},
				{id: "N#Box.operator <(Box,Box)", kind: KindOperator, start: 5, end: 5},
				{id: "N#Box.operator >(Box,Box)", kind: KindOperator, start: 6, end: 6},
				{id: "N#Box.operator >>(Box,int)", kind: KindOperator, start: 7, end: 7},
				{id: "N#Box.operator checked +(Box,Box)", kind: KindOperator, start: 8, end: 8},
				{id: "N#Box.explicit operator System.Text.Rune(Box)", kind: KindOperator, start: 9, end: 9},
			},
		},
		{
			name: "ExplicitInterfaceImplementations",
			src: "namespace N\n" +
				"{\n" +
				"    public class R\n" +
				"    {\n" +
				"        int IProvider<A>.Value { get; }\n" +
				"        int IProvider<B>.Value { get; }\n" +
				"        void System.IDisposable.Dispose() { }\n" +
				"        int IProvider<System.Int32>.this[int i] => 0;\n" +
				"    }\n" +
				"}\n",
			want: []csharpWant{
				{id: "N#R", kind: KindType, start: 3, end: 9, headStart: 3, headEnd: 3},
				{id: "N#R.IProvider<A>.Value", kind: KindProperty, start: 5, end: 5},
				{id: "N#R.IProvider<B>.Value", kind: KindProperty, start: 6, end: 6},
				{id: "N#R.System.IDisposable.Dispose()", kind: KindMethod, start: 7, end: 7},
				{id: "N#R.IProvider<int>.this[int]", kind: KindIndexer, start: 8, end: 8},
			},
		},
		{
			name: "PrimaryConstructorMintsItsOwnConstructorOnly",
			src: "namespace N\n" +
				"{\n" +
				"    public class P(int a, string b) { }\n" +
				"}\n",
			want: []csharpWant{
				{id: "N#P", kind: KindType, start: 3, end: 3, headStart: 3, headEnd: 3},
				{id: "N#P.P(int,string)", kind: KindConstructor, start: 3, end: 3},
			},
		},
		{
			name: "PositionalPropertyDeclaredInTheBodyIsMintedOnce",
			src: "namespace N\n" +
				"{\n" +
				"    public record Point(int X) { public int X { get; init; } = X; }\n" +
				"}\n",
			want: []csharpWant{
				{id: "N#Point", kind: KindType, start: 3, end: 3, headStart: 3, headEnd: 3},
				{id: "N#Point.X", kind: KindProperty, start: 3, end: 3},
			},
		},
		{
			name: "PositionalPropertyPerPartialDeclarationPart",
			src: "namespace N\n" +
				"{\n" +
				"    public partial record Q(int Z) { }\n" +
				"    public partial record Q { public int Z => 1; }\n" +
				"}\n",
			want: []csharpWant{
				{id: "N#Q", kind: KindType, start: 3, end: 3, headStart: 3, headEnd: 3, partial: true},
				{id: "N#Q.Z", kind: KindProperty, start: 3, end: 3},
				{id: "N#Q", kind: KindType, start: 4, end: 4, headStart: 4, headEnd: 4, partial: true},
				{id: "N#Q.Z", kind: KindProperty, start: 4, end: 4},
			},
		},
		{
			name: "InheritedPositionalPropertyIsStillMinted",
			src: "namespace N\n" +
				"{\n" +
				"    public record Base(int Id);\n" +
				"    public record Derived(int Id, int Extra) : Base(Id);\n" +
				"}\n",
			want: []csharpWant{
				{id: "N#Base", kind: KindType, start: 3, end: 3, headStart: 3, headEnd: 3},
				{id: "N#Base.Id", kind: KindProperty, start: 3, end: 3},
				{id: "N#Derived", kind: KindType, start: 4, end: 4, headStart: 4, headEnd: 4},
				{id: "N#Derived.Id", kind: KindProperty, start: 4, end: 4},
				{id: "N#Derived.Extra", kind: KindProperty, start: 4, end: 4},
			},
		},
		{
			name: "TypeHeadSpansBaseListAndConstraints",
			src: "namespace N\n" +
				"{\n" +
				"    /// <summary>d</summary>\n" +
				"    public class Box<T>\n" +
				"        : Base\n" +
				"        where T : class\n" +
				"    {\n" +
				"        public int X;\n" +
				"    }\n" +
				"}\n",
			want: []csharpWant{
				{id: "N#Box<>", kind: KindType, start: 3, end: 9, doc: "<summary>d</summary>", headStart: 3, headEnd: 6},
				{id: "N#Box<>.X", kind: KindField, start: 8, end: 8},
			},
		},
		{
			name: "NestedAndFileScopedNamespacesAndGlobal",
			src: "namespace A.B\n" +
				"{\n" +
				"    namespace C { public class X { } }\n" +
				"    public class Y { }\n" +
				"}\n" +
				"public class G { }\n",
			want: []csharpWant{
				{id: "A.B.C#X", kind: KindType, start: 3, end: 3, headStart: 3, headEnd: 3},
				{id: "A.B#Y", kind: KindType, start: 4, end: 4, headStart: 4, headEnd: 4},
				{id: "global#G", kind: KindType, start: 6, end: 6, headStart: 6, headEnd: 6},
			},
		},
		{
			name: "FileScopedNamespace",
			src: "namespace F.G;\n" +
				"\n" +
				"public class X { }\n",
			want: []csharpWant{
				{id: "F.G#X", kind: KindType, start: 3, end: 3, headStart: 3, headEnd: 3},
			},
		},
		{
			name: "VerbatimIdentifiers",
			src: "namespace N\n" +
				"{\n" +
				"    public class @class { public int @this; public int @Foo; }\n" +
				"}\n",
			want: []csharpWant{
				{id: "N#@class", kind: KindType, start: 3, end: 3, headStart: 3, headEnd: 3},
				{id: "N#@class.@this", kind: KindField, start: 3, end: 3},
				{id: "N#@class.Foo", kind: KindField, start: 3, end: 3},
			},
		},
		{
			name: "PartialTypeAndMethod",
			src: "namespace N\n" +
				"{\n" +
				"    public partial class P { partial void M(); public void K() { } }\n" +
				"}\n",
			want: []csharpWant{
				{id: "N#P", kind: KindType, start: 3, end: 3, headStart: 3, headEnd: 3, partial: true},
				{id: "N#P.M()", kind: KindMethod, start: 3, end: 3, partial: true},
				{id: "N#P.K()", kind: KindMethod, start: 3, end: 3},
			},
		},
		{
			name: "PreprocessorBranchesAtNamespaceLevel",
			src: "namespace N\n" +
				"{\n" +
				"#if A\n" +
				"    public class X { }\n" +
				"#else\n" +
				"    public class X { }\n" +
				"#endif\n" +
				"}\n",
			want: []csharpWant{
				{id: "N#X", kind: KindType, start: 4, end: 4, headStart: 4, headEnd: 4},
				{id: "N#X", kind: KindType, start: 6, end: 6, headStart: 6, headEnd: 6},
			},
		},
		{
			name: "PreprocessorBranchesInATypeBody",
			src: "namespace N\n" +
				"{\n" +
				"    public class C\n" +
				"    {\n" +
				"#if A\n" +
				"        public void M() { }\n" +
				"#elif B\n" +
				"        public void M(int a) { }\n" +
				"#else\n" +
				"        public void M() { }\n" +
				"#endif\n" +
				"    }\n" +
				"}\n",
			want: []csharpWant{
				{id: "N#C", kind: KindType, start: 3, end: 12, headStart: 3, headEnd: 3},
				{id: "N#C.M()", kind: KindMethod, start: 6, end: 6},
				{id: "N#C.M(int)", kind: KindMethod, start: 8, end: 8},
				{id: "N#C.M()", kind: KindMethod, start: 10, end: 10},
			},
		},
		{
			name: "NothingInsideBodiesOrTopLevelStatements",
			src: "namespace N\n" +
				"{\n" +
				"    public class C\n" +
				"    {\n" +
				"        public void M()\n" +
				"        {\n" +
				"            void Local() { }\n" +
				"            System.Action a = () => { };\n" +
				"        }\n" +
				"        public int P { get { return 0; } set { } }\n" +
				"    }\n" +
				"}\n" +
				"System.Console.WriteLine();\n",
			want: []csharpWant{
				{id: "N#C", kind: KindType, start: 3, end: 11, headStart: 3, headEnd: 3},
				{id: "N#C.M()", kind: KindMethod, start: 5, end: 9},
				{id: "N#C.P", kind: KindProperty, start: 10, end: 10},
			},
		},
		{
			name: "MembersOutsideAnyTypeAreNotGlyphed",
			src: "namespace N\n" +
				"{\n" +
				"    int IProvider<A>.Value { get; }\n" +
				"    void System.IDisposable.Dispose() { }\n" +
				"    public void Free() { }\n" +
				"}\n" +
				"int Top() { return 1; }\n" +
				"void System.IDisposable.Dispose2() { }\n",
			want: nil,
		},
		{
			name: "ExtensionBlockMembersAreNotGlyphed",
			src: "namespace N\n" +
				"{\n" +
				"    public static class Ext\n" +
				"    {\n" +
				"        extension(string s)\n" +
				"        {\n" +
				"            public int Len => s.Length;\n" +
				"        }\n" +
				"    }\n" +
				"}\n",
			want: []csharpWant{
				{id: "N#Ext", kind: KindType, start: 3, end: 9, headStart: 3, headEnd: 3},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractCSharp(t, tt.src)
			if len(got.Dropped) != 0 {
				t.Errorf("dropped = %+v; want none", got.Dropped)
			}
			if len(got.Symbols) != len(tt.want) {
				t.Fatalf("minted %d symbols %v; want %d", len(got.Symbols), csharpIDs(got.Symbols), len(tt.want))
			}
			for i, want := range tt.want {
				sym := got.Symbols[i]
				if sym.ID != want.id || sym.Kind != want.kind {
					t.Errorf("symbol %d = (%q, %s); want (%q, %s)", i, sym.ID, sym.Kind, want.id, want.kind)
				}
				if sym.Start != want.start || sym.End != want.end {
					t.Errorf("%s span = %d-%d; want %d-%d", want.id, sym.Start, sym.End, want.start, want.end)
				}
				if sym.Doc != want.doc {
					t.Errorf("%s Doc = %q; want %q", want.id, sym.Doc, want.doc)
				}
				if sym.HeadStart != want.headStart || sym.HeadEnd != want.headEnd {
					t.Errorf("%s head = %d-%d; want %d-%d", want.id, sym.HeadStart, sym.HeadEnd, want.headStart, want.headEnd)
				}
				if sym.Partial != want.partial {
					t.Errorf("%s Partial = %t; want %t", want.id, sym.Partial, want.partial)
				}
				if _, err := glyph.Parse(glyph.CSharp, sym.ID); err != nil {
					t.Errorf("%s does not parse: %v", sym.ID, err)
				}
			}
		})
	}
}

func TestCSharpStrategy_SymbolOffsetsAndSignatures(t *testing.T) {
	t.Parallel()
	src := "namespace N\n" +
		"{\n" +
		"    public class C(int a)\n" +
		"    {\n" +
		"        public int M(int x)\n" +
		"        {\n" +
		"            return x;\n" +
		"        }\n" +
		"        public int F;\n" +
		"    }\n" +
		"}\n"
	got := extractCSharp(t, src)
	byID := make(map[string]Symbol, len(got.Symbols))
	for _, sym := range got.Symbols {
		byID[sym.ID] = sym
	}

	method := byID["N#C.M(int)"]
	if text := src[method.DeclStart:method.DeclEnd]; text != "public int M(int x)\n        {\n            return x;\n        }" {
		t.Errorf("M declaration text = %q", text)
	}
	if method.Signature != "public int M(int x)" {
		t.Errorf("M Signature = %q; want %q", method.Signature, "public int M(int x)")
	}
	if text := src[method.BodyStart:method.DeclEnd]; text != "{\n            return x;\n        }" {
		t.Errorf("M body text = %q", text)
	}
	if method.SigEnd != 6 {
		t.Errorf("M SigEnd = %d; want 6", method.SigEnd)
	}

	field := byID["N#C.F"]
	if field.BodyStart != field.DeclEnd || field.SigEnd != 0 {
		t.Errorf("F BodyStart = %d, DeclEnd = %d, SigEnd = %d; want BodyStart == DeclEnd and SigEnd 0", field.BodyStart, field.DeclEnd, field.SigEnd)
	}

	primary := byID["N#C.C(int)"]
	if text := src[primary.DeclStart:primary.DeclEnd]; text != "(int a)" {
		t.Errorf("primary constructor declaration text = %q; want %q", text, "(int a)")
	}
	if primary.Signature != "(int a)" {
		t.Errorf("primary constructor Signature = %q; want %q", primary.Signature, "(int a)")
	}
}

func TestCSharpStrategy_DroppedIDs(t *testing.T) {
	t.Parallel()
	src := "namespace N\n" +
		"{\n" +
		"    public class Café { }\n" +
		"    public class Fine { }\n" +
		"}\n"
	got := extractCSharp(t, src)
	if ids := csharpIDs(got.Symbols); len(ids) != 1 || ids[0] != "N#Fine" {
		t.Errorf("minted ids = %v; want [N#Fine]", ids)
	}
	if len(got.Dropped) != 1 {
		t.Fatalf("dropped = %+v; want exactly one entry", got.Dropped)
	}
	dropped := got.Dropped[0]
	if dropped.ID != "N#Café" {
		t.Errorf("dropped ID = %q; want %q", dropped.ID, "N#Café")
	}
	var parseErr *glyph.ParseError
	if !errors.As(dropped.Err, &parseErr) || parseErr.Reason != glyph.ReasonMemberNotIdentifier {
		t.Errorf("dropped Err = %v; want a ParseError with Reason %s", dropped.Err, glyph.ReasonMemberNotIdentifier)
	}
	if text := src[dropped.DeclStart:dropped.DeclEnd]; text != "public class Café { }" {
		t.Errorf("dropped declaration text = %q", text)
	}
}

func TestCSharpStrategy_ExtensionBlockLeavesTheFileLossy(t *testing.T) {
	t.Parallel()
	src := "namespace N\n" +
		"{\n" +
		"    public static class Ext\n" +
		"    {\n" +
		"        extension(string s)\n" +
		"        {\n" +
		"            public int Len => s.Length;\n" +
		"        }\n" +
		"    }\n" +
		"}\n"
	got := extractCSharp(t, src)
	if !got.Partial {
		t.Errorf("Partial = false; want the linked grammar to report the extension block as a syntax error")
	}
}

func TestCSharpStrategy_FileFacts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		src           string
		wantHeader    string
		wantPackage   string
		wantGenerated bool
	}{
		{
			name:        "HeaderAndNamespace",
			src:         "// The file header.\n\nnamespace A.B\n{\n}\n",
			wantHeader:  "The file header.",
			wantPackage: "A.B",
		},
		{
			name:          "AutoGeneratedBannerIsSkippedAsHeader",
			src:           "// <auto-generated>\n//   by a tool\n// </auto-generated>\n\n// The real header.\nnamespace N\n{\n}\n",
			wantHeader:    "The real header.",
			wantPackage:   "N",
			wantGenerated: true,
		},
		{
			name:          "OnlyAnAutoGeneratedBanner",
			src:           "// <auto-generated/>\nnamespace N\n{\n}\n",
			wantHeader:    "",
			wantPackage:   "N",
			wantGenerated: true,
		},
		{
			name:        "FileScopedNamespace",
			src:         "namespace F.G;\n\npublic class X { }\n",
			wantPackage: "F.G",
		},
		{
			name:        "FirstNamespaceWins",
			src:         "namespace First { }\nnamespace Second { }\n",
			wantPackage: "First",
		},
		{
			name:        "NoNamespace",
			src:         "public class X { }\n",
			wantPackage: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractCSharp(t, tt.src)
			if got.Header != tt.wantHeader {
				t.Errorf("Header = %q; want %q", got.Header, tt.wantHeader)
			}
			if got.Package != tt.wantPackage {
				t.Errorf("Package = %q; want %q", got.Package, tt.wantPackage)
			}
			if got.PackageDoc != "" {
				t.Errorf("PackageDoc = %q; want empty", got.PackageDoc)
			}
			if got.Generated != tt.wantGenerated {
				t.Errorf("Generated = %t; want %t", got.Generated, tt.wantGenerated)
			}
		})
	}

	strategy, ok := StrategyFor("csharp")
	if !ok {
		t.Fatal(`StrategyFor("csharp") reported ok == false`)
	}
	for _, base := range []string{"Widget.cs", "WidgetTests.cs", "WidgetTest.cs"} {
		if strategy.TestFile(base) {
			t.Errorf("TestFile(%q) = true; want false for every C# file name", base)
		}
	}
}

// csharpIDs lists the ids of symbols, for failure messages.
func csharpIDs(symbols []Symbol) []string {
	ids := make([]string, len(symbols))
	for i, sym := range symbols {
		ids[i] = sym.ID
	}
	return ids
}
