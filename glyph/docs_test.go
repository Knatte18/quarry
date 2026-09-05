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
// asked to check today — every Go-alphabet row. The §2 per-language self-form rows for Python and
// C# are documentation only: there is no extractor to test either alphabet against, so they are
// carried as commented-out placeholders below the table, naming the task that will enable them,
// rather than silently omitted.
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
	// §2, Python and C# self form (documentation only — no extractor to test either alphabet
	// against yet; docs/rewrite-plan.md §9 is the task that enables them):
	// {name: "§2 Python self form, module", section: "2", input: "loomyard.engine.layout#",
	//	want: Glyph{Lang: Python, Unit: "loomyard.engine.layout"}},
	// {name: "§2 C# self form, namespace", section: "2", input: "Loomyard.Engine.Layout#",
	//	want: Glyph{Lang: CSharp, Unit: "Loomyard.Engine.Layout"}},

	// §3 — the member, per language: an empty member is the self form in every alphabet; only Go
	// is testable here, for the same reason as §2's placeholders above.
	{
		name: "§3 empty member is the self form (Go)", section: "3",
		input: "internal/reedengine/render#",
		want:  Glyph{Lang: Go, Unit: "internal/reedengine/render"},
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
			got, err := Parse(Go, tt.input)
			if err != nil {
				t.Fatalf("Parse(Go, %q) error = %v; want nil", tt.input, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse(Go, %q) = %+v; want %+v", tt.input, got, tt.want)
			}
			if tt.want.Name == "" && !got.IsSelf() {
				t.Errorf("Parse(Go, %q).IsSelf() = false; want true", tt.input)
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
