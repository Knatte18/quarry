// docs_test.go is the executable form of docs/glyph.md's examples: one table of accepts and one of
// rejects, each row reproducing an example or a reject the rewritten §1, §2, §3 or §5 states, cited
// by section. A row added to the document without a matching row here is the drift this file exists
// to catch.

package glyph

import (
	"errors"
	"reflect"
	"testing"
)

// docsAccept is every accept-case example from docs/glyph.md §1, §2, §3 and §5 that Parse can be
// asked to check today — every Go-alphabet and C#-alphabet row.
// The §2 Python self-form row is documentation only, since no Python alphabet is implemented.
// It is carried as a commented-out placeholder inside the table rather than silently omitted.
var docsAccept = []acceptCase{
	// §1 — the form and the unit: the example table's whole-glyph rows, including the two self-form
	// rows this task adds.
	{
		name: "§1 package-level function", section: "1",
		input: "internal/logger#stderrHandlerSnapshot",
		want:  Glyph{Lang: Go, Unit: "internal/logger", Name: "stderrHandlerSnapshot"},
	},
	{
		name: "§1 method", section: "1",
		input: "internal/logger#dualHandler.stderr",
		want:  Glyph{Lang: Go, Unit: "internal/logger", Owner: []string{"dualHandler"}, Name: "stderr"},
	},
	{
		name: "§1 method in a deeper unit", section: "1",
		input: "internal/reedengine/render#Renderer.Draw",
		want:  Glyph{Lang: Go, Unit: "internal/reedengine/render", Owner: []string{"Renderer"}, Name: "Draw"},
	},
	{
		name: "§1 function in a package main", section: "1",
		input: "cmd/lyx#run",
		want:  Glyph{Lang: Go, Unit: "cmd/lyx", Name: "run"},
	},
	{
		name: "§1 the package itself", section: "1",
		input: "internal/reedengine/render#",
		want:  Glyph{Lang: Go, Unit: "internal/reedengine/render"},
	},
	{
		name: "§1 the file itself", section: "1",
		input: "internal/reedengine/render/focus.go#",
		want:  Glyph{Lang: Go, Unit: "internal/reedengine/render/focus.go"},
	},

	// §2 — the unit, per language: what a trailing "#" means for Go, restated under the section
	// that specifies the rule, plus the external test unit's own self glyph, which §2 states parses
	// like any other rather than being denied a self form.
	{
		name: "§2 Go self form, package directory", section: "2",
		input: "internal/reedengine/render#",
		want:  Glyph{Lang: Go, Unit: "internal/reedengine/render"},
	},
	{
		name: "§2 Go self form, file", section: "2",
		input: "internal/reedengine/render/focus.go#",
		want:  Glyph{Lang: Go, Unit: "internal/reedengine/render/focus.go"},
	},
	{
		name: "§2 external test unit self glyph is well-formed", section: "2",
		input: "internal/logger_test#",
		want:  Glyph{Lang: Go, Unit: "internal/logger_test"},
	},
	{
		name: "§2 C# self form, namespace", section: "2",
		input: "Loomyard.Engine.Layout#",
		want:  Glyph{Lang: CSharp, Unit: "Loomyard.Engine.Layout"},
	},
	// The Python self form is documentation only: no Python alphabet is implemented.
	// {name: "§2 Python self form, module", section: "2", input: "loomyard.engine.layout#",
	//	want: Glyph{Lang: Python, Unit: "loomyard.engine.layout"}},

	// §1 — the C# example rows.
	{
		name: "§1 C# overload", section: "1",
		input: "Loomyard.Engine.Layout#Renderer.Draw(int)",
		want: Glyph{
			Lang: CSharp, Unit: "Loomyard.Engine.Layout",
			Owner: []string{"Renderer"}, Name: "Draw", Params: []string{"int"},
		},
	},
	{
		name: "§1 C# property", section: "1",
		input: "Loomyard.Engine.Layout#Renderer.Title",
		want:  Glyph{Lang: CSharp, Unit: "Loomyard.Engine.Layout", Owner: []string{"Renderer"}, Name: "Title"},
	},

	// §3 — the member, per language: an empty member is the self form in every alphabet.
	{
		name: "§3 empty member is the self form (Go)", section: "3",
		input: "internal/reedengine/render#",
		want:  Glyph{Lang: Go, Unit: "internal/reedengine/render"},
	},
	{
		name: "§3 empty member is the self form (C#)", section: "3",
		input: "global#",
		want:  Glyph{Lang: CSharp, Unit: "global"},
	},
	{
		name: "§3 C# two parameters", section: "3",
		input: "Loomyard.Engine.Layout#Renderer.Draw(int,string)",
		want: Glyph{
			Lang: CSharp, Unit: "Loomyard.Engine.Layout",
			Owner: []string{"Renderer"}, Name: "Draw", Params: []string{"int", "string"},
		},
	},
	{
		name: "§3 C# no parameters", section: "3",
		input: "Loomyard.Engine.Layout#Renderer.Draw()",
		want: Glyph{
			Lang: CSharp, Unit: "Loomyard.Engine.Layout",
			Owner: []string{"Renderer"}, Name: "Draw", Params: []string{},
		},
	},
	{
		name: "§3 C# constructor", section: "3",
		input: "Loomyard.Engine.Layout#Renderer.Renderer(Box)",
		want: Glyph{
			Lang: CSharp, Unit: "Loomyard.Engine.Layout",
			Owner: []string{"Renderer"}, Name: "Renderer", Params: []string{"Box"},
		},
	},
	{
		name: "§3 C# parameter modifier", section: "3",
		input: "Loomyard.Engine.Layout#Renderer.Draw(out Box)",
		want: Glyph{
			Lang: CSharp, Unit: "Loomyard.Engine.Layout",
			Owner: []string{"Renderer"}, Name: "Draw", Params: []string{"out Box"},
		},
	},
	{
		name: "§3 C# open generic types", section: "3",
		input: "Loomyard.Engine.Layout#Dictionary<,>.Add<>(T)",
		want: Glyph{
			Lang: CSharp, Unit: "Loomyard.Engine.Layout",
			Owner: []string{"Dictionary<,>"}, Name: "Add<>", Params: []string{"T"},
		},
	},
	{
		name: "§3 C# indexer", section: "3",
		input: "Loomyard.Engine.Layout#Renderer.this[int]",
		want: Glyph{
			Lang: CSharp, Unit: "Loomyard.Engine.Layout",
			Owner: []string{"Renderer"}, Name: "this", Params: []string{"int"},
		},
	},
	{
		name: "§3 C# operator", section: "3",
		input: "Loomyard.Engine.Layout#Box.operator +(Box,Box)",
		want: Glyph{
			Lang: CSharp, Unit: "Loomyard.Engine.Layout",
			Owner: []string{"Box"}, Name: "operator +", Params: []string{"Box", "Box"},
		},
	},
	{
		name: "§3 C# finalizer", section: "3",
		input: "Loomyard.Engine.Layout#Renderer.~Renderer()",
		want: Glyph{
			Lang: CSharp, Unit: "Loomyard.Engine.Layout",
			Owner: []string{"Renderer"}, Name: "~Renderer", Params: []string{},
		},
	},
	{
		name: "§3 C# explicit interface implementation", section: "3",
		input: "Loomyard.Engine.Layout#Renderer.ILayout.Draw(int)",
		want: Glyph{
			Lang: CSharp, Unit: "Loomyard.Engine.Layout",
			Owner: []string{"Renderer", "ILayout"}, Name: "Draw", Params: []string{"int"},
		},
	},

	// §2 — the C# unit: a reserved keyword segment keeps its escape.
	// A .cs file is addressed by the path self glyph the Go unit alphabet spells.
	{
		name: "§2 C# unit segment that is a reserved keyword", section: "2",
		input: "Loomyard.@class#",
		want:  Glyph{Lang: CSharp, Unit: "Loomyard.@class"},
	},
	{
		name: "§2 path self glyph of a .cs file", section: "2",
		input: "Norce.Models/Foo.cs#",
		want:  Glyph{Lang: Go, Unit: "Norce.Models/Foo.cs"},
	},

	// §3 — the C# member spellings: the static constructor, the conversion and checked operators.
	// Also the escape only on a reserved keyword, canonical parameter modifiers and the explicit interface qualifier.
	{
		name: "§3 C# static constructor", section: "3",
		input: "Loomyard.Engine.Layout#Renderer.static Renderer()",
		want: Glyph{
			Lang: CSharp, Unit: "Loomyard.Engine.Layout",
			Owner: []string{"Renderer"}, Name: "static Renderer", Params: []string{},
		},
	},
	{
		name: "§3 C# conversion operator", section: "3",
		input: "Loomyard.Engine.Layout#Box.implicit operator int(Box)",
		want: Glyph{
			Lang: CSharp, Unit: "Loomyard.Engine.Layout",
			Owner: []string{"Box"}, Name: "implicit operator int", Params: []string{"Box"},
		},
	},
	{
		name: "§3 C# conversion operator to a qualified type", section: "3",
		input: "Loomyard.Engine.Layout#Box.implicit operator System.Text.Rune(Box)",
		want: Glyph{
			Lang: CSharp, Unit: "Loomyard.Engine.Layout",
			Owner: []string{"Box"}, Name: "implicit operator System.Text.Rune", Params: []string{"Box"},
		},
	},
	{
		name: "§3 C# checked operator", section: "3",
		input: "Loomyard.Engine.Layout#Box.operator checked +(Box,Box)",
		want: Glyph{
			Lang: CSharp, Unit: "Loomyard.Engine.Layout",
			Owner: []string{"Box"}, Name: "operator checked +", Params: []string{"Box", "Box"},
		},
	},
	{
		name: "§3 C# escape on a reserved keyword", section: "3",
		input: "Loomyard.Engine.Layout#Renderer.@class",
		want:  Glyph{Lang: CSharp, Unit: "Loomyard.Engine.Layout", Owner: []string{"Renderer"}, Name: "@class"},
	},
	{
		name: "§3 C# parameter modifiers", section: "3",
		input: "Loomyard.Engine.Layout#Renderer.Draw(scoped ref Span<int>)",
		want: Glyph{
			Lang: CSharp, Unit: "Loomyard.Engine.Layout",
			Owner: []string{"Renderer"}, Name: "Draw", Params: []string{"scoped ref Span<int>"},
		},
	},
	{
		name: "§3 C# params array", section: "3",
		input: "Loomyard.Engine.Layout#Renderer.Draw(params string[])",
		want: Glyph{
			Lang: CSharp, Unit: "Loomyard.Engine.Layout",
			Owner: []string{"Renderer"}, Name: "Draw", Params: []string{"params string[]"},
		},
	},
	{
		name: "§3 C# explicit interface implementation with a type argument", section: "3",
		input: "Loomyard.Engine.Layout#Renderer.IProvider<A>.Value",
		want: Glyph{
			Lang: CSharp, Unit: "Loomyard.Engine.Layout",
			Owner: []string{"Renderer", "IProvider<A>"}, Name: "Value",
		},
	},
	{
		name: "§3 C# explicit interface implementation of a qualified interface", section: "3",
		input: "Loomyard.Engine.Layout#Renderer.System.IDisposable.Dispose()",
		want: Glyph{
			Lang: CSharp, Unit: "Loomyard.Engine.Layout",
			Owner: []string{"Renderer", "System", "IDisposable"}, Name: "Dispose", Params: []string{},
		},
	},

	// §5 — resolution: a self glyph is well-formed input, exactly like a member glyph.
	{
		name: "§5 a self glyph is well-formed input", section: "5",
		input: "internal/reedengine/render#",
		want:  Glyph{Lang: Go, Unit: "internal/reedengine/render"},
	},
	{
		name: "§5 a glyph without parentheses carries no parameter list", section: "5",
		input: "Loomyard.Engine.Layout#Renderer.Draw",
		want:  Glyph{Lang: CSharp, Unit: "Loomyard.Engine.Layout", Owner: []string{"Renderer"}, Name: "Draw"},
	},
	{
		name: "§5 a namespace-named directory's self glyph under the Go alphabet", section: "5",
		input: "Norce.Models#",
		want:  Glyph{Lang: Go, Unit: "Norce.Models"},
	},
	{
		name: "§5 the same string's self glyph under the C# alphabet", section: "5",
		input: "Norce.Models#",
		want:  Glyph{Lang: CSharp, Unit: "Norce.Models"},
	},
}

// TestDocsAccept drives docsAccept, asserting the whole parsed Glyph with reflect.DeepEqual and,
// for every self-form row (an empty Name), that IsSelf reports true — the property §1 and §3 both
// claim of these rows.
func TestDocsAccept(t *testing.T) {
	for _, tt := range docsAccept {
		t.Run(tt.name+"/"+tt.section, func(t *testing.T) {
			got, err := Parse(tt.want.Lang, tt.input)
			if err != nil {
				t.Fatalf("Parse(%v, %q) error = %v; want nil", tt.want.Lang, tt.input, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse(%v, %q) = %+v; want %+v", tt.want.Lang, tt.input, got, tt.want)
			}
			if tt.want.Name == "" && !got.IsSelf() {
				t.Errorf("Parse(%v, %q).IsSelf() = false; want true", tt.want.Lang, tt.input)
			}
		})
	}
}

// docsReject is every reject-case example from docs/glyph.md §2, §3 and §5.
// Those are the bare-path pre-resolution rejection, a "#" inside a path segment, the two ways the repository root itself cannot be addressed by resolve,
// and the C# spellings the alphabet turns away with their fix.
var docsReject = []rejectCase{
	{
		name: "§5 a bare path is rejected pre-resolution", section: "5",
		lang: Go, input: "internal/logger",
		reason: ReasonNoSeparator, detail: "internal/logger#",
	},
	{
		name: "§5 a \"#\" in a path segment is an explicit error", section: "5",
		lang: Go, input: "internal/lo#gger#run",
		reason: ReasonMultipleSeparators, detail: "internal/lo#gger#run",
	},
	{
		name: "§5 the repository root, a lone \"#\"", section: "5",
		lang: Go, input: "#",
		reason: ReasonUnitEmpty, detail: "",
	},
	{
		name: "§5 the repository root, a dot segment", section: "5",
		lang: Go, input: ".#",
		reason: ReasonUnitDotSegment, detail: ".",
	},
	{
		name: "§3 a C# type outside the canonical spelling", section: "3",
		lang: CSharp, input: "Loomyard.Engine.Layout#Renderer.Draw(Int32)",
		reason: ReasonNonCanonicalType, detail: "int",
	},
	{
		name: "§3 a C# parameter that still carries its name", section: "3",
		lang: CSharp, input: "Loomyard.Engine.Layout#Renderer.Draw(int a)",
		reason: ReasonNonCanonicalType, detail: "int",
	},
	{
		name: "§2 a C# unit segment with a needless escape", section: "2",
		lang: CSharp, input: "@Foo#",
		reason: ReasonNeedlessEscape, detail: "Foo",
	},
}

// TestDocsReject drives docsReject, asserting each row's Reason and Detail via errors.As, and that
// the returned Glyph is always the zero value.
func TestDocsReject(t *testing.T) {
	for _, tt := range docsReject {
		t.Run(tt.name+"/"+tt.section, func(t *testing.T) {
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
