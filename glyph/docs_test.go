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

	// §5 — resolution: a self glyph is well-formed input, exactly like a member glyph.
	{
		name: "§5 a self glyph is well-formed input", section: "5",
		input: "internal/reedengine/render#",
		want:  Glyph{Lang: Go, Unit: "internal/reedengine/render"},
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

// docsReject is every reject-case example from docs/glyph.md §5: the bare-path pre-resolution
// rejection, a "#" inside a path segment, and the two ways the repository root itself cannot be
// addressed by resolve.
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
