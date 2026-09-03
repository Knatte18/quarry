// golang_test.go covers the Go alphabet: the accept table (goAccept), the reject table (goReject),
// a completeness test that every Reason has at least one reject row across this file and
// parse_test.go, a ParseError field test for Lang and Input, the Error() smoke tests, and a
// case-sensitivity test.

package glyph

import (
	"errors"
	"reflect"
	"testing"
)

// acceptCase is one row of the Go accept table: an input Parse must accept, and the exact Glyph it
// must produce, sourced from a named docs/glyph.md section.
type acceptCase struct {
	name    string
	input   string
	want    Glyph
	section string
}

// goAccept is the Go alphabet's accept table. Every row is driven both by TestParse_GoAccept below
// and by string_test.go's round-trip tests.
var goAccept = []acceptCase{
	{
		name: "package-level function", input: "internal/logger#stderrHandlerSnapshot", section: "1",
		want: Glyph{Lang: Go, Unit: "internal/logger", Name: "stderrHandlerSnapshot"},
	},
	{
		name: "method with one-element owner", input: "internal/logger#dualHandler.stderr", section: "1",
		want: Glyph{Lang: Go, Unit: "internal/logger", Owner: []string{"dualHandler"}, Name: "stderr"},
	},
	{
		name: "method in a deeper unit", input: "internal/reedengine/render#Renderer.Draw", section: "1",
		want: Glyph{Lang: Go, Unit: "internal/reedengine/render", Owner: []string{"Renderer"}, Name: "Draw"},
	},
	{
		name: "function in a package main", input: "cmd/lyx#run", section: "1",
		want: Glyph{Lang: Go, Unit: "cmd/lyx", Name: "run"},
	},
	{
		name: "package-level function, deep unit", input: "internal/shedrecipe#Lookup", section: "7",
		want: Glyph{Lang: Go, Unit: "internal/shedrecipe", Name: "Lookup"},
	},
	{
		name: "init, a plain identifier to the parser", input: "internal/logger#init", section: "3",
		want: Glyph{Lang: Go, Unit: "internal/logger", Name: "init"},
	},
	{
		name: "external test package unit", input: "internal/logger_test#SomeName", section: "2",
		want: Glyph{Lang: Go, Unit: "internal/logger_test", Name: "SomeName"},
	},
	{
		name: "canonical type-parameter spelling", input: "internal/logger#Box", section: "3",
		want: Glyph{Lang: Go, Unit: "internal/logger", Name: "Box"},
	},
	{
		name: "method on dualHandler", input: "internal/logger#dualHandler.Handle", section: "3",
		want: Glyph{Lang: Go, Unit: "internal/logger", Owner: []string{"dualHandler"}, Name: "Handle"},
	},
	{
		name: "method on durableHandler", input: "internal/logger#durableHandler.Handle", section: "3",
		want: Glyph{Lang: Go, Unit: "internal/logger", Owner: []string{"durableHandler"}, Name: "Handle"},
	},
	{
		name: "single-segment unit", input: "glyph#Parse", section: "",
		want: Glyph{Lang: Go, Unit: "glyph", Name: "Parse"},
	},
	{
		name: "deep unit", input: "a/b/c/d/e#Name", section: "",
		want: Glyph{Lang: Go, Unit: "a/b/c/d/e", Name: "Name"},
	},
	{
		name: "Unicode member identifier", input: "internal/logger#Ærlig", section: "",
		want: Glyph{Lang: Go, Unit: "internal/logger", Name: "Ærlig"},
	},
	{
		name: "underscore as a member name", input: "internal/logger#_", section: "",
		want: Glyph{Lang: Go, Unit: "internal/logger", Name: "_"},
	},
	{
		name: "unit segment with ., -, + and ~", input: "internal/go-lib.v2+x~1#Name", section: "",
		want: Glyph{Lang: Go, Unit: "internal/go-lib.v2+x~1", Name: "Name"},
	},
}

// TestParse_GoAccept drives goAccept, asserting the whole parsed Glyph with reflect.DeepEqual.
func TestParse_GoAccept(t *testing.T) {
	for _, tt := range goAccept {
		t.Run(tt.name+"/"+tt.section, func(t *testing.T) {
			got, err := Parse(Go, tt.input)
			if err != nil {
				t.Fatalf("Parse(Go, %q) error = %v; want nil", tt.input, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse(Go, %q) = %+v; want %+v", tt.input, got, tt.want)
			}
		})
	}
}

// TestParse_GoAccept_ReceiverIsHalfTheKey asserts that dualHandler.Handle and durableHandler.Handle
// parse to Glyph values differing in Owner alone, which is docs/glyph.md §3's point that the
// receiver type is half the key.
func TestParse_GoAccept_ReceiverIsHalfTheKey(t *testing.T) {
	dual, err := Parse(Go, "internal/logger#dualHandler.Handle")
	if err != nil {
		t.Fatalf("Parse(Go, dualHandler.Handle) error = %v; want nil", err)
	}
	durable, err := Parse(Go, "internal/logger#durableHandler.Handle")
	if err != nil {
		t.Fatalf("Parse(Go, durableHandler.Handle) error = %v; want nil", err)
	}
	if reflect.DeepEqual(dual, durable) {
		t.Fatalf("dualHandler.Handle and durableHandler.Handle parsed equal: %+v", dual)
	}
	if dual.Unit != durable.Unit || dual.Name != durable.Name {
		t.Errorf("Unit/Name differ unexpectedly: dual = %+v, durable = %+v", dual, durable)
	}
	if reflect.DeepEqual(dual.Owner, durable.Owner) {
		t.Errorf("Owner did not differ: dual = %v, durable = %v", dual.Owner, durable.Owner)
	}
}

// goReject is the Go alphabet's reject table, reusing parse_test.go's rejectCase type so the
// completeness test below can range over goReject and parseReject in one loop. Every row's lang
// field is Go.
//
// The doubly-invalid unit rows report the leftmost failing segment's reason, since the walk is
// left-to-right and first-failure-wins. "(*dualHandler).Handle" carries both "*" and parentheses
// and reports ReasonMemberPointer, because the pointer check runs before the parens check. The §7
// dotted spelling ("internal/reedengine/render.Renderer.Draw") never reaches the member checks at
// all: it has no "#", so it fails the split.
var goReject = []rejectCase{
	{name: "no separator", lang: Go, input: "internal/logger",
		reason: ReasonNoSeparator, detail: ""},
	{name: "empty string", lang: Go, input: "",
		reason: ReasonNoSeparator, detail: ""},
	{name: "dotted spelling has no #", lang: Go, input: "internal/reedengine/render.Renderer.Draw", section: "7",
		reason: ReasonNoSeparator, detail: ""},
	{name: "unit empty", lang: Go, input: "#run",
		reason: ReasonUnitEmpty, detail: ""},
	{name: "leading slash", lang: Go, input: "/internal/logger#run",
		reason: ReasonUnitEmptySegment, detail: ""},
	{name: "trailing slash", lang: Go, input: "internal/logger/#run",
		reason: ReasonUnitEmptySegment, detail: ""},
	{name: "doubled slash", lang: Go, input: "internal//logger#run",
		reason: ReasonUnitEmptySegment, detail: ""},
	{name: "leading dot segment", lang: Go, input: "./internal/logger#run",
		reason: ReasonUnitDotSegment, detail: "."},
	{name: "dot-dot segment", lang: Go, input: "internal/../logger#run",
		reason: ReasonUnitDotSegment, detail: ".."},
	{name: "dot-dot segment before a space", lang: Go, input: "internal/../lo gger#run",
		reason: ReasonUnitDotSegment, detail: ".."},
	{name: "empty segment before a space", lang: Go, input: "internal//lo gger#run",
		reason: ReasonUnitEmptySegment, detail: ""},
	{name: "backslash in unit", lang: Go, input: `internal\logger#run`,
		reason: ReasonUnitBadRune, detail: `'\\'`},
	{name: "space in unit segment", lang: Go, input: "internal/my logger#run",
		reason: ReasonUnitBadRune, detail: `' '`},
	{name: "leading space", lang: Go, input: " internal/logger#run",
		reason: ReasonUnitBadRune, detail: `' '`},
	{name: "tab in second unit segment", lang: Go, input: "internal/lo\tgger#run",
		reason: ReasonUnitBadRune, detail: `'\t'`},
	{name: "member empty", lang: Go, input: "internal/logger#",
		reason: ReasonMemberEmpty, detail: ""},
	{name: "leading dot component", lang: Go, input: "internal/logger#.Handle",
		reason: ReasonMemberEmptyComponent, detail: ""},
	{name: "trailing dot component", lang: Go, input: "internal/logger#Handle.",
		reason: ReasonMemberEmptyComponent, detail: ""},
	{name: "doubled dot component", lang: Go, input: "internal/logger#A..b",
		reason: ReasonMemberEmptyComponent, detail: ""},
	{name: "too deep", lang: Go, input: "internal/logger#A.B.c",
		reason: ReasonMemberTooDeep, detail: "A.B.c"},
	{name: "not identifier, leading digit", lang: Go, input: "internal/logger#1abc",
		reason: ReasonMemberNotIdentifier, detail: "1abc"},
	{name: "not identifier, hyphen", lang: Go, input: "internal/logger#a-b",
		reason: ReasonMemberNotIdentifier, detail: "a-b"},
	{name: "keyword func", lang: Go, input: "internal/logger#func",
		reason: ReasonMemberKeyword, detail: "func"},
	{name: "keyword range", lang: Go, input: "internal/logger#range",
		reason: ReasonMemberKeyword, detail: "range"},
	{name: "type params", lang: Go, input: "internal/logger#Box[T]", section: "3",
		reason: ReasonMemberTypeParams, detail: `'['`},
	{name: "parens", lang: Go, input: "internal/logger#Renderer.Draw(int)",
		reason: ReasonMemberParens, detail: `'('`},
	{name: "pointer receiver, parenthesized", lang: Go, input: "internal/logger#(*dualHandler).Handle", section: "3",
		reason: ReasonMemberPointer, detail: `'*'`},
	{name: "pointer receiver, bare", lang: Go, input: "internal/logger#*dualHandler.Handle", section: "3",
		reason: ReasonMemberPointer, detail: `'*'`},
	{name: "second hash reaches the member validator", lang: Go, input: "internal/logger#a#b",
		reason: ReasonMemberBadRune, detail: `'#'`},
	{name: "space inside a member component", lang: Go, input: "internal/logger#A .b",
		reason: ReasonMemberBadRune, detail: `' '`},
	{name: "trailing space", lang: Go, input: "internal/logger#run ",
		reason: ReasonMemberBadRune, detail: `' '`},
	{name: "invalid UTF-8 in the unit half", lang: Go, input: "internal\xff/logger#run",
		reason: ReasonInvalidUTF8, detail: ""},
	{name: "invalid UTF-8 in the member half", lang: Go, input: "internal/logger#ru\xffn",
		reason: ReasonInvalidUTF8, detail: ""},
}

// TestParse_GoReject drives goReject, asserting each row's Reason and Detail via errors.As, and
// that the returned Glyph is always the zero value.
func TestParse_GoReject(t *testing.T) {
	for _, tt := range goReject {
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

// TestReasons_Completeness ranges over Reasons and fails for any element that no row of goReject or
// parseReject names. This guarantees that adding a seventeenth Reason constant and listing it in
// Reasons fails until a reject case exists; it does not guarantee that a constant added without
// being listed in Reasons is ever caught — that is caught by review, not by this test.
func TestReasons_Completeness(t *testing.T) {
	covered := make(map[Reason]bool, len(Reasons))
	for _, tt := range goReject {
		covered[tt.reason] = true
	}
	for _, tt := range parseReject {
		covered[tt.reason] = true
	}
	for _, r := range Reasons {
		if !covered[r] {
			t.Errorf("Reason %v has no row in goReject or parseReject", r)
		}
	}
}

// TestParseError_Fields asserts Lang and Input on the *ParseError recovered from one unit reject,
// one member reject, and one unsupported_language reject.
func TestParseError_Fields(t *testing.T) {
	tests := []struct {
		name     string
		lang     Language
		input    string
		wantLang Language
	}{
		{name: "unit reject", lang: Go, input: "internal/../logger#run", wantLang: Go},
		{name: "member reject", lang: Go, input: "internal/logger#func", wantLang: Go},
		{name: "unsupported language reject", lang: Language("python"), input: "internal/logger#run", wantLang: Language("python")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.lang, tt.input)
			var pe *ParseError
			if !errors.As(err, &pe) {
				t.Fatalf("Parse(%v, %q) error = %v; want *ParseError", tt.lang, tt.input, err)
			}
			if pe.Lang != tt.wantLang {
				t.Errorf("Lang = %v; want %v", pe.Lang, tt.wantLang)
			}
			if pe.Input != tt.input {
				t.Errorf("Input = %q; want %q", pe.Input, tt.input)
			}
		})
	}
}

// TestParseError_Error_NonEmpty asserts that every Reason in Reasons produces a non-empty Error()
// message.
func TestParseError_Error_NonEmpty(t *testing.T) {
	for _, r := range Reasons {
		pe := &ParseError{Lang: Go, Input: "internal/logger#run", Reason: r, Detail: "x"}
		if pe.Error() == "" {
			t.Errorf("Error() for Reason %v is empty", r)
		}
	}
}

// TestParseError_Error_Distinct asserts that, for one fixed Lang, Input and Detail, every Reason in
// Reasons produces a distinct Error() message.
func TestParseError_Error_Distinct(t *testing.T) {
	seen := make(map[string]Reason, len(Reasons))
	for _, r := range Reasons {
		pe := &ParseError{Lang: Go, Input: "internal/logger#run", Reason: r, Detail: "x"}
		msg := pe.Error()
		if prior, ok := seen[msg]; ok {
			t.Errorf("Reason %v and %v both produce Error() = %q", prior, r, msg)
		}
		seen[msg] = r
	}
}

// TestParse_GoAccept_CaseSensitivity asserts that internal/Logger#Foo and internal/logger#foo both
// parse, and that their Glyph values differ: neither folds into the other.
func TestParse_GoAccept_CaseSensitivity(t *testing.T) {
	upper, err := Parse(Go, "internal/Logger#Foo")
	if err != nil {
		t.Fatalf("Parse(Go, internal/Logger#Foo) error = %v; want nil", err)
	}
	lower, err := Parse(Go, "internal/logger#foo")
	if err != nil {
		t.Fatalf("Parse(Go, internal/logger#foo) error = %v; want nil", err)
	}
	if reflect.DeepEqual(upper, lower) {
		t.Errorf("internal/Logger#Foo and internal/logger#foo parsed equal: %+v", upper)
	}
}
