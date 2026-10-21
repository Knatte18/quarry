// csharp_test.go covers the C# alphabet.
// It holds the accept table (csharpAccept), the reject table (csharpReject), and the check that Self builds the self forms the accept table spells.

package glyph

import (
	"errors"
	"reflect"
	"testing"
)

// csharpAccept is the C# alphabet's accept table.
// Every row is driven by TestParse_CSharpAccept and by string_test.go's round-trip tests.
var csharpAccept = []acceptCase{
	{
		name: "type with arity", input: "Ns#Box<>", section: "3",
		want: Glyph{Lang: CSharp, Unit: "Ns", Name: "Box<>"},
	},
	{
		name: "type with two-parameter arity", input: "Ns#Pair<,>", section: "3",
		want: Glyph{Lang: CSharp, Unit: "Ns", Name: "Pair<,>"},
	},
	{
		name: "nested types", input: "Loomyard.Engine#Outer<>.Inner.Leaf", section: "3",
		want: Glyph{Lang: CSharp, Unit: "Loomyard.Engine", Owner: []string{"Outer<>", "Inner"}, Name: "Leaf"},
	},
	{
		name: "method with parameters", input: "Loomyard.Engine.Layout#Renderer.Draw(int,string)", section: "1",
		want: Glyph{
			Lang: CSharp, Unit: "Loomyard.Engine.Layout",
			Owner: []string{"Renderer"}, Name: "Draw", Params: []string{"int", "string"},
		},
	},
	{
		name: "method without parameters", input: "Ns#Renderer.Draw()", section: "3",
		want: Glyph{Lang: CSharp, Unit: "Ns", Owner: []string{"Renderer"}, Name: "Draw", Params: []string{}},
	},
	{
		name: "property has no parentheses", input: "Loomyard.Engine.Layout#Renderer.Title", section: "1",
		want: Glyph{Lang: CSharp, Unit: "Loomyard.Engine.Layout", Owner: []string{"Renderer"}, Name: "Title"},
	},
	{
		name: "generic method", input: "Ns#Renderer.Draw<>(T)", section: "3",
		want: Glyph{Lang: CSharp, Unit: "Ns", Owner: []string{"Renderer"}, Name: "Draw<>", Params: []string{"T"}},
	},
	{
		name: "constructor", input: "Ns#Renderer.Renderer(Box)", section: "3",
		want: Glyph{Lang: CSharp, Unit: "Ns", Owner: []string{"Renderer"}, Name: "Renderer", Params: []string{"Box"}},
	},
	{
		name: "static constructor", input: "Ns#Renderer.static Renderer()", section: "3",
		want: Glyph{Lang: CSharp, Unit: "Ns", Owner: []string{"Renderer"}, Name: "static Renderer", Params: []string{}},
	},
	{
		name: "finalizer", input: "Ns#Renderer.~Renderer()", section: "3",
		want: Glyph{Lang: CSharp, Unit: "Ns", Owner: []string{"Renderer"}, Name: "~Renderer", Params: []string{}},
	},
	{
		name: "indexer", input: "Ns#Renderer.this[int]", section: "3",
		want: Glyph{Lang: CSharp, Unit: "Ns", Owner: []string{"Renderer"}, Name: "this", Params: []string{"int"}},
	},
	{
		name: "operator plus", input: "Ns#Box.operator +(Box,Box)", section: "3",
		want: Glyph{Lang: CSharp, Unit: "Ns", Owner: []string{"Box"}, Name: "operator +", Params: []string{"Box", "Box"}},
	},
	{
		name: "operator less-than", input: "Ns#Box.operator <(Box,Box)", section: "3",
		want: Glyph{Lang: CSharp, Unit: "Ns", Owner: []string{"Box"}, Name: "operator <", Params: []string{"Box", "Box"}},
	},
	{
		name: "operator shift-right", input: "Ns#Box.operator >>(Box,int)", section: "3",
		want: Glyph{Lang: CSharp, Unit: "Ns", Owner: []string{"Box"}, Name: "operator >>", Params: []string{"Box", "int"}},
	},
	{
		name: "operator equality", input: "Ns#Box.operator ==(Box,Box)", section: "3",
		want: Glyph{Lang: CSharp, Unit: "Ns", Owner: []string{"Box"}, Name: "operator ==", Params: []string{"Box", "Box"}},
	},
	{
		name: "checked operator", input: "Ns#Box.operator checked +(Box,Box)", section: "3",
		want: Glyph{Lang: CSharp, Unit: "Ns", Owner: []string{"Box"}, Name: "operator checked +", Params: []string{"Box", "Box"}},
	},
	{
		name: "conversion to a qualified type", input: "Ns#Box.implicit operator System.Text.Rune(Box)", section: "3",
		want: Glyph{
			Lang: CSharp, Unit: "Ns", Owner: []string{"Box"},
			Name: "implicit operator System.Text.Rune", Params: []string{"Box"},
		},
	},
	{
		name: "explicit conversion to a keyword type", input: "Ns#Box.explicit operator int(Box)", section: "3",
		want: Glyph{Lang: CSharp, Unit: "Ns", Owner: []string{"Box"}, Name: "explicit operator int", Params: []string{"Box"}},
	},
	{
		name: "explicit interface implementation with type arguments", input: "Ns#Renderer.IProvider<A>.Value", section: "3",
		want: Glyph{Lang: CSharp, Unit: "Ns", Owner: []string{"Renderer", "IProvider<A>"}, Name: "Value"},
	},
	{
		name: "explicit interface implementation, qualified interface", input: "Ns#Renderer.System.IDisposable.Dispose()", section: "3",
		want: Glyph{
			Lang: CSharp, Unit: "Ns",
			Owner: []string{"Renderer", "System", "IDisposable"}, Name: "Dispose", Params: []string{},
		},
	},
	{
		name: "global unit", input: "global#Box", section: "2",
		want: Glyph{Lang: CSharp, Unit: "global", Name: "Box"},
	},
	{
		name: "verbatim keyword identifier", input: "Ns#@class", section: "3",
		want: Glyph{Lang: CSharp, Unit: "Ns", Name: "@class"},
	},
	{
		name: "verbatim keyword unit segment", input: "Ns.@this#Box", section: "2",
		want: Glyph{Lang: CSharp, Unit: "Ns.@this", Name: "Box"},
	},
	{
		name: "ref modifier", input: "Ns#Renderer.Draw(ref int)", section: "3",
		want: Glyph{Lang: CSharp, Unit: "Ns", Owner: []string{"Renderer"}, Name: "Draw", Params: []string{"ref int"}},
	},
	{
		name: "params modifier", input: "Ns#Renderer.Draw(params string[])", section: "3",
		want: Glyph{Lang: CSharp, Unit: "Ns", Owner: []string{"Renderer"}, Name: "Draw", Params: []string{"params string[]"}},
	},
	{
		name: "several modifiers and generic and tuple types", input: "Ns#Renderer.Draw(ref readonly int,out Box,List<int>,(int,string))", section: "3",
		want: Glyph{
			Lang: CSharp, Unit: "Ns", Owner: []string{"Renderer"}, Name: "Draw",
			Params: []string{"ref readonly int", "out Box", "List<int>", "(int,string)"},
		},
	},
	{
		name: "namespace self form", input: "Ns#", section: "2",
		want: Glyph{Lang: CSharp, Unit: "Ns"},
	},
	{
		name: "global self form", input: "global#", section: "2",
		want: Glyph{Lang: CSharp, Unit: "global"},
	},
}

// TestParse_CSharpAccept drives csharpAccept, asserting each whole Glyph with reflect.DeepEqual.
// It also asserts that Self builds the same Glyph for every self-form row.
func TestParse_CSharpAccept(t *testing.T) {
	t.Parallel()
	for _, tt := range csharpAccept {
		t.Run(tt.name+"/"+tt.section, func(t *testing.T) {
			t.Parallel()
			got, err := Parse(CSharp, tt.input)
			if err != nil {
				t.Fatalf("Parse(CSharp, %q) error = %v; want nil", tt.input, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse(CSharp, %q) = %+v; want %+v", tt.input, got, tt.want)
			}
			if tt.want.Name != "" {
				return
			}
			self, err := Self(CSharp, tt.want.Unit)
			if err != nil {
				t.Fatalf("Self(CSharp, %q) error = %v; want nil", tt.want.Unit, err)
			}
			if !reflect.DeepEqual(self, tt.want) {
				t.Errorf("Self(CSharp, %q) = %+v; want %+v", tt.want.Unit, self, tt.want)
			}
		})
	}
}

// csharpReject is the C# alphabet's reject table.
// It has one row per reason the alphabet adds and per non-canonical spelling it refuses.
var csharpReject = []rejectCase{
	{name: "slash in unit", lang: CSharp, input: "Ns/Sub#C",
		reason: ReasonUnitBadRune, detail: `'/'`},
	{name: "double colon in unit", lang: CSharp, input: "Ns::Sub#C",
		reason: ReasonUnitBadRune, detail: `':'`},
	{name: "empty unit segment", lang: CSharp, input: "Ns..Sub#C",
		reason: ReasonUnitEmptySegment, detail: ""},
	{name: "leading dot in unit", lang: CSharp, input: ".Ns#C",
		reason: ReasonUnitEmptySegment, detail: ""},
	{name: "empty unit", lang: CSharp, input: "#Foo",
		reason: ReasonUnitEmpty, detail: ""},
	{name: "unit segment starts with a digit", lang: CSharp, input: "1Ns#C",
		reason: ReasonUnitNotIdentifier, detail: "1Ns"},
	{name: "space in unit segment", lang: CSharp, input: "Ns Sub#C",
		reason: ReasonUnitNotIdentifier, detail: "Ns Sub"},
	{name: "needless escape in unit", lang: CSharp, input: "@Foo#C",
		reason: ReasonNeedlessEscape, detail: "Foo"},
	{name: "needless escape in member", lang: CSharp, input: "Ns#@Foo",
		reason: ReasonNeedlessEscape, detail: "Foo"},
	{name: "alias type", lang: CSharp, input: "Ns#C.Draw(Int32)",
		reason: ReasonNonCanonicalType, detail: "int"},
	{name: "global qualifier", lang: CSharp, input: "Ns#C.Draw(global::X)",
		reason: ReasonNonCanonicalType, detail: "X"},
	{name: "extra whitespace", lang: CSharp, input: "Ns#C.Draw(List< int >)",
		reason: ReasonNonCanonicalType, detail: "List<int>"},
	{name: "tuple element name", lang: CSharp, input: "Ns#C.Draw((int a,string b))",
		reason: ReasonNonCanonicalType, detail: "(int,string)"},
	{name: "parameter name", lang: CSharp, input: "Ns#C.Draw(int a)",
		reason: ReasonNonCanonicalType, detail: "int"},
	{name: "parameter name after a modifier", lang: CSharp, input: "Ns#C.Draw(ref int x)",
		reason: ReasonNonCanonicalType, detail: "ref int"},
	{name: "parameter default", lang: CSharp, input: "Ns#C.Draw(int a = 5)",
		reason: ReasonNonCanonicalType, detail: "int"},
	{name: "parameter attribute", lang: CSharp, input: "Ns#C.Draw([In] int)",
		reason: ReasonNonCanonicalType, detail: "int"},
	{name: "parameter attribute, alias and name", lang: CSharp, input: "Ns#C.Draw([In] Int32 a)",
		reason: ReasonNonCanonicalType, detail: "int"},
	{name: "alias in type arguments", lang: CSharp, input: "Ns#C.IProvider<Int32>.Value",
		reason: ReasonNonCanonicalType, detail: "IProvider<int>"},
	{name: "alias in conversion target", lang: CSharp, input: "Ns#C.implicit operator Int32(C)",
		reason: ReasonNonCanonicalType, detail: "int"},
	{name: "empty parameter entry", lang: CSharp, input: "Ns#C.Draw(int,)",
		reason: ReasonMemberEmptyComponent, detail: ""},
	{name: "only empty parameter entries", lang: CSharp, input: "Ns#C.Draw(,)",
		reason: ReasonMemberEmptyComponent, detail: ""},
	{name: "parameter entry holds only a modifier", lang: CSharp, input: "Ns#C.Draw(ref)",
		reason: ReasonMemberNotIdentifier, detail: "ref"},
	{name: "unclosed parameter list", lang: CSharp, input: "Ns#C.Draw(int",
		reason: ReasonUnbalancedBrackets, detail: "C.Draw(int"},
	{name: "unclosed type arguments", lang: CSharp, input: "Ns#Box<",
		reason: ReasonUnbalancedBrackets, detail: "Box<"},
	{name: "unclosed indexer list", lang: CSharp, input: "Ns#C.this[int",
		reason: ReasonUnbalancedBrackets, detail: "C.this[int"},
	{name: "indexer without brackets", lang: CSharp, input: "Ns#C.this",
		reason: ReasonIndexerBrackets, detail: "this"},
	{name: "indexer with parentheses", lang: CSharp, input: "Ns#C.this(int)",
		reason: ReasonIndexerBrackets, detail: "this"},
	{name: "brackets after another name", lang: CSharp, input: "Ns#C.Foo[int]",
		reason: ReasonIndexerBrackets, detail: "Foo"},
	{name: "finalizer without parentheses", lang: CSharp, input: "Ns#C.~C",
		reason: ReasonMissingParens, detail: "~C"},
	{name: "static constructor without parentheses", lang: CSharp, input: "Ns#C.static C",
		reason: ReasonMissingParens, detail: "static C"},
	{name: "operator without parentheses", lang: CSharp, input: "Ns#C.operator +",
		reason: ReasonMissingParens, detail: "operator +"},
	{name: "unknown operator", lang: CSharp, input: "Ns#C.operator @(C,C)",
		reason: ReasonUnknownOperator, detail: "@"},
	{name: "operator with no token", lang: CSharp, input: "Ns#C.operator (C,C)",
		reason: ReasonUnknownOperator, detail: ""},
	{name: "checked form of an operator without one", lang: CSharp, input: "Ns#C.operator checked ==(C,C)",
		reason: ReasonUnknownOperator, detail: "checked =="},
	{name: "empty member component", lang: CSharp, input: "Ns#A..B",
		reason: ReasonMemberEmptyComponent, detail: ""},
	{name: "bare keyword component", lang: CSharp, input: "Ns#class",
		reason: ReasonMemberKeyword, detail: "class"},
	{name: "component starts with a digit", lang: CSharp, input: "Ns#C.1x",
		reason: ReasonMemberNotIdentifier, detail: "1x"},
	{name: "string failing both halves reports the unit", lang: CSharp, input: "a/b#class",
		reason: ReasonUnitBadRune, detail: `'/'`},
}

// TestParse_CSharpReject drives csharpReject, asserting each row's Reason and Detail via errors.As,
// and that the returned Glyph is always the zero value.
func TestParse_CSharpReject(t *testing.T) {
	t.Parallel()
	for _, tt := range csharpReject {
		t.Run(tt.name+"/"+tt.section, func(t *testing.T) {
			t.Parallel()
			got, err := Parse(tt.lang, tt.input)

			var pe *ParseError
			if !errors.As(err, &pe) {
				t.Fatalf("Parse(%v, %q) error = %v; want *ParseError", tt.lang, tt.input, err)
			}
			if pe.Reason != tt.reason {
				t.Errorf("Parse(%v, %q) Reason = %v; want %v", tt.lang, tt.input, pe.Reason, tt.reason)
			}
			if pe.Detail != tt.detail {
				t.Errorf("Parse(%v, %q) Detail = %q; want %q", tt.lang, tt.input, pe.Detail, tt.detail)
			}
			if !reflect.DeepEqual(got, Glyph{}) {
				t.Errorf("Parse(%v, %q) Glyph = %+v; want zero value", tt.lang, tt.input, got)
			}
		})
	}
}
