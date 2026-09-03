// parse_test.go covers the language-free layer: splitGlyph over every docs/glyph.md §1 example, the
// first-"#" rule, the language check, and the two precedence orders (unsupported language over no
// separator, invalid UTF-8 over no separator).

package glyph

import (
	"errors"
	"reflect"
	"testing"
)

// TestSplitGlyph asserts the unit/member split for every whole-glyph example in docs/glyph.md §1.
func TestSplitGlyph(t *testing.T) {
	tests := []struct {
		name       string
		lang       string
		input      string
		wantUnit   string
		wantMember string
	}{
		{
			name: "Go package-level function", lang: "Go",
			input:    "internal/logger#stderrHandlerSnapshot",
			wantUnit: "internal/logger", wantMember: "stderrHandlerSnapshot",
		},
		{
			name: "Go method", lang: "Go",
			input:    "internal/logger#dualHandler.stderr",
			wantUnit: "internal/logger", wantMember: "dualHandler.stderr",
		},
		{
			name: "Go method in a deeper unit", lang: "Go",
			input:    "internal/reedengine/render#Renderer.Draw",
			wantUnit: "internal/reedengine/render", wantMember: "Renderer.Draw",
		},
		{
			name: "Go function in a package main", lang: "Go",
			input:    "cmd/lyx#run",
			wantUnit: "cmd/lyx", wantMember: "run",
		},
		{
			name: "Python method on a nested class", lang: "Python",
			input:    "loomyard.engine.layout#Beta.Inner.handle",
			wantUnit: "loomyard.engine.layout", wantMember: "Beta.Inner.handle",
		},
		{
			name: "C# method overload", lang: "C#",
			input:    "Loomyard.Engine.Layout#Renderer.Draw(int)",
			wantUnit: "Loomyard.Engine.Layout", wantMember: "Renderer.Draw(int)",
		},
		{
			name: "C# property", lang: "C#",
			input:    "Loomyard.Engine.Layout#Renderer.Title",
			wantUnit: "Loomyard.Engine.Layout", wantMember: "Renderer.Title",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name+"/"+tt.lang, func(t *testing.T) {
			gotUnit, gotMember, ok := splitGlyph(tt.input)
			if !ok {
				t.Fatalf("splitGlyph(%q) ok = false; want true", tt.input)
			}
			if gotUnit != tt.wantUnit {
				t.Errorf("splitGlyph(%q) unit = %q; want %q", tt.input, gotUnit, tt.wantUnit)
			}
			if gotMember != tt.wantMember {
				t.Errorf("splitGlyph(%q) member = %q; want %q", tt.input, gotMember, tt.wantMember)
			}
		})
	}
}

// TestSplitGlyph_FirstHash asserts that a string carrying two "#" splits at the first, leaving the
// second in the member half, and that the second "#" reaches the Go member validator rather than a
// pre-split check.
func TestSplitGlyph_FirstHash(t *testing.T) {
	unit, member, ok := splitGlyph("internal/logger#a#b")
	if !ok {
		t.Fatalf("splitGlyph(%q) ok = false; want true", "internal/logger#a#b")
	}
	if unit != "internal/logger" {
		t.Errorf("splitGlyph unit = %q; want %q", unit, "internal/logger")
	}
	if member != "a#b" {
		t.Errorf("splitGlyph member = %q; want %q", member, "a#b")
	}

	_, err := Parse(Go, "internal/logger#a#b")
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("Parse(Go, %q) error = %v; want *ParseError", "internal/logger#a#b", err)
	}
	if pe.Reason != ReasonMemberBadRune {
		t.Errorf("Parse(Go, %q) Reason = %v; want %v", "internal/logger#a#b", pe.Reason, ReasonMemberBadRune)
	}
}

// rejectCase is one row of a reject table: an input Parse must reject with a given Reason and
// Detail, sourced from a named docs/glyph.md section where the spec writes the case down. It is
// declared once here and reused by golang_test.go's goReject table and its completeness test.
type rejectCase struct {
	name    string
	lang    Language
	input   string
	reason  Reason
	detail  string
	section string
}

// parseReject holds every language-free reject case: the unsupported_language group, the
// reject-precedence case (unsupported language over no separator), and the invalid_utf8-precedes-
// the-split case.
var parseReject = []rejectCase{
	{
		name: "unsupported language: python", lang: Language("python"), input: "internal/logger#run",
		reason: ReasonUnsupportedLanguage, detail: "python",
	},
	{
		name: "unsupported language: csharp", lang: Language("csharp"), input: "internal/logger#run",
		reason: ReasonUnsupportedLanguage, detail: "csharp",
	},
	{
		name: "unsupported language: empty", lang: Language(""), input: "internal/logger#run",
		reason: ReasonUnsupportedLanguage, detail: "",
	},
	{
		name: "unsupported language: arbitrary value", lang: Language("rust"), input: "internal/logger#run",
		reason: ReasonUnsupportedLanguage, detail: "rust",
	},
	{
		name: "reject precedence: unsupported language over no separator",
		lang: Language("python"), input: "no-hash",
		reason: ReasonUnsupportedLanguage, detail: "python",
	},
	{
		name: "reject precedence: invalid UTF-8 over no separator",
		lang: Go, input: "no-hash-\xff",
		reason: ReasonInvalidUTF8, detail: "",
	},
}

// TestParse_LanguageFreeReject drives parseReject, asserting each row's Reason and Detail via
// errors.As, and that the returned Glyph is always the zero value.
func TestParse_LanguageFreeReject(t *testing.T) {
	for _, tt := range parseReject {
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
