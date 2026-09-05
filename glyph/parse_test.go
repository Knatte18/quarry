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

// TestSplitGlyph_FirstHash asserts that a string carrying two "#" still splits at the first,
// leaving the second in the member half — splitGlyph itself has no count rule — but that Parse
// never reaches that split for such an input: its own count check catches the second "#" first and
// rejects with ReasonMultipleSeparators, before the Go member validator ever sees it.
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
	if pe.Reason != ReasonMultipleSeparators {
		t.Errorf("Parse(Go, %q) Reason = %v; want %v", "internal/logger#a#b", pe.Reason, ReasonMultipleSeparators)
	}
}

// selfAccept is the self-form accept table: every row is a glyph string with an empty member,
// which Parse must accept as the self form rather than reject with ReasonNoSeparator's predecessor
// (the retired ReasonMemberEmpty).
var selfAccept = []struct {
	name  string
	input string
	unit  string
}{
	{name: "package unit", input: "internal/reedengine/render#", unit: "internal/reedengine/render"},
	{name: "file unit", input: "internal/reedengine/render/focus.go#", unit: "internal/reedengine/render/focus.go"},
	{name: "package main unit", input: "cmd/lyx#", unit: "cmd/lyx"},
	{name: "testdata unit", input: "internal/engine/testdata/tree/pkg_test#", unit: "internal/engine/testdata/tree/pkg_test"},
}

// TestParse_SelfAccept drives selfAccept, asserting Unit, a nil Owner, an empty Name, a nil Params,
// and that IsSelf reports true for each row.
func TestParse_SelfAccept(t *testing.T) {
	for _, tt := range selfAccept {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(Go, tt.input)
			if err != nil {
				t.Fatalf("Parse(Go, %q) error = %v; want nil", tt.input, err)
			}
			if got.Unit != tt.unit {
				t.Errorf("Parse(Go, %q) Unit = %q; want %q", tt.input, got.Unit, tt.unit)
			}
			if got.Owner != nil {
				t.Errorf("Parse(Go, %q) Owner = %v; want nil", tt.input, got.Owner)
			}
			if got.Name != "" {
				t.Errorf("Parse(Go, %q) Name = %q; want empty", tt.input, got.Name)
			}
			if got.Params != nil {
				t.Errorf("Parse(Go, %q) Params = %v; want nil", tt.input, got.Params)
			}
			if !got.IsSelf() {
				t.Errorf("Parse(Go, %q).IsSelf() = false; want true", tt.input)
			}
		})
	}
}

// selfReject holds the reject cases the grammar change introduces or moves: the bare-path
// no-separator cases whose Detail now names the fix, the multiple-"#" cases, and the unit-rule
// cases applied to a self-form input (an empty member).
var selfReject = []rejectCase{
	{name: "bare path, unit only", lang: Go, input: "internal/logger",
		reason: ReasonNoSeparator, detail: "internal/logger#"},
	{name: "bare path, cwd-relative file", lang: Go, input: "logger.go",
		reason: ReasonNoSeparator, detail: "logger.go#"},
	{name: "three hashes", lang: Go, input: "a#b#c",
		reason: ReasonMultipleSeparators, detail: "a#b#c"},
	{name: "trailing empty member after a second hash", lang: Go, input: "a#b#",
		reason: ReasonMultipleSeparators, detail: "a#b#"},
	{name: "empty unit, self form", lang: Go, input: "#",
		reason: ReasonUnitEmpty, detail: ""},
	{name: "dot unit, self form", lang: Go, input: ".#",
		reason: ReasonUnitDotSegment, detail: "."},
	{name: "dot-dot segment, self form", lang: Go, input: "a/../b#",
		reason: ReasonUnitDotSegment, detail: ".."},
	{name: "empty segment, self form", lang: Go, input: "a//b#",
		reason: ReasonUnitEmptySegment, detail: ""},
}

// TestParse_SelfReject drives selfReject, asserting each row's Reason and Detail via errors.As —
// never the message text — and that the returned Glyph is always the zero value.
func TestParse_SelfReject(t *testing.T) {
	for _, tt := range selfReject {
		t.Run(tt.name, func(t *testing.T) {
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

// TestParseError_Error_NoSeparator pins card 4's authoritative rendered message verbatim, including
// the repository-relative clause and the parenthesised Detail, for the exact input this hint is
// aimed at.
func TestParseError_Error_NoSeparator(t *testing.T) {
	_, err := Parse(Go, "internal/logger")
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("Parse(Go, %q) error = %v; want *ParseError", "internal/logger", err)
	}
	want := `glyph: parse "internal/logger" as go: a glyph needs a "#"; a path is addressed as its own glyph by appending one to its repository-relative form (internal/logger#)`
	if got := pe.Error(); got != want {
		t.Errorf("Error() = %q; want %q", got, want)
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
