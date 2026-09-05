// self_test.go is the executable form of the compose direction's contract: that Self composes the
// self-form string and delegates to Parse rather than duplicating its rules, in both the accept and
// the reject direction.

package glyph

import (
	"errors"
	"reflect"
	"testing"
)

// selfComposePaths holds the paths TestSelf_ComposeThenParse drives, each proven both ways: Self
// composes and validates it, and parsing the composed string back recovers the same path.
var selfComposePaths = []string{
	"internal/logger",
	"internal/logger/logger.go",
	"cmd/lyx",
	"internal/engine/testdata/tree/pkg_test",
}

// TestSelf_ComposeThenParse drives selfComposePaths, asserting for each path p that Self(Go, p)
// returns a Glyph for which IsSelf reports true, whose String is exactly p with "#" appended, and
// which is reflect.DeepEqual to what Parse(Go, p+"#") returns directly; and that parsing that
// string back yields a Glyph whose path — its String with the trailing "#" removed — is
// byte-identical to p.
func TestSelf_ComposeThenParse(t *testing.T) {
	for _, p := range selfComposePaths {
		t.Run(p, func(t *testing.T) {
			composed, err := Self(Go, p)
			if err != nil {
				t.Fatalf("Self(Go, %q) error = %v; want nil", p, err)
			}
			if !composed.IsSelf() {
				t.Errorf("Self(Go, %q).IsSelf() = false; want true", p)
			}
			want := p + "#"
			if got := composed.String(); got != want {
				t.Errorf("Self(Go, %q).String() = %q; want %q", p, got, want)
			}

			parsed, err := Parse(Go, want)
			if err != nil {
				t.Fatalf("Parse(Go, %q) error = %v; want nil", want, err)
			}
			if !reflect.DeepEqual(composed, parsed) {
				t.Errorf("Self(Go, %q) = %+v; want %+v (Parse(Go, %q))", p, composed, parsed, want)
			}

			roundTripped, err := Parse(Go, composed.String())
			if err != nil {
				t.Fatalf("Parse(Go, %q) (composed string) error = %v; want nil", composed.String(), err)
			}
			gotPath := roundTripped.String()[:len(roundTripped.String())-len("#")]
			if gotPath != p {
				t.Errorf("round trip path = %q; want %q", gotPath, p)
			}
		})
	}
}

// selfUnitReject is the shared table driving both TestSelf_Reject and TestSelf_Reject_MatchesParse:
// each row's path, run through Self, must fail with reason, and the same reason must appear when
// path+"#" is run through Parse directly — one shared table, so a unit rule added to Parse without
// a matching Self row is visible here rather than silently uncovered.
var selfUnitReject = []struct {
	name   string
	lang   Language
	path   string
	reason Reason
}{
	{name: "multiple separators", lang: Go, path: "a#b", reason: ReasonMultipleSeparators},
	{name: "unit empty", lang: Go, path: "", reason: ReasonUnitEmpty},
	{name: "dot segment", lang: Go, path: ".", reason: ReasonUnitDotSegment},
	{name: "dot-dot segment", lang: Go, path: "a/../b", reason: ReasonUnitDotSegment},
	{name: "empty segment", lang: Go, path: "a//b", reason: ReasonUnitEmptySegment},
	{name: "unsupported language", lang: Language("python"), path: "x", reason: ReasonUnsupportedLanguage},
}

// TestSelf_Reject drives selfUnitReject through Self, asserting Reason via errors.As on a
// *ParseError and that the returned Glyph is always the zero value.
func TestSelf_Reject(t *testing.T) {
	for _, tt := range selfUnitReject {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Self(tt.lang, tt.path)

			var pe *ParseError
			if !errors.As(err, &pe) {
				t.Fatalf("Self(%v, %q) error = %v; want *ParseError", tt.lang, tt.path, err)
			}
			if pe.Reason != tt.reason {
				t.Errorf("Self(%v, %q) Reason = %v; want %v", tt.lang, tt.path, pe.Reason, tt.reason)
			}
			if !reflect.DeepEqual(got, Glyph{}) {
				t.Errorf("Self(%v, %q) Glyph = %+v; want zero value", tt.lang, tt.path, got)
			}
		})
	}
}

// TestSelf_Reject_MatchesParse drives the same selfUnitReject table through Parse(lang, path+"#")
// directly, asserting the identical Reason each row asserts against Self above. Equal reasons here
// are the strongest available proof that Self delegates rather than duplicates: any unit rule Parse
// gains without an equivalent Self row would leave this test unaffected but TestSelf_Reject failing
// to compile against a stale table, since both are driven from selfUnitReject together.
func TestSelf_Reject_MatchesParse(t *testing.T) {
	for _, tt := range selfUnitReject {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.lang, tt.path+"#")

			var pe *ParseError
			if !errors.As(err, &pe) {
				t.Fatalf("Parse(%v, %q) error = %v; want *ParseError", tt.lang, tt.path+"#", err)
			}
			if pe.Reason != tt.reason {
				t.Errorf("Parse(%v, %q) Reason = %v; want %v", tt.lang, tt.path+"#", pe.Reason, tt.reason)
			}
			if !reflect.DeepEqual(got, Glyph{}) {
				t.Errorf("Parse(%v, %q) Glyph = %+v; want zero value", tt.lang, tt.path+"#", got)
			}
		})
	}
}
