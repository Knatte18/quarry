// parse.go declares Parse and the language-free split it runs before dispatching to a language's
// own alphabet.

package glyph

import (
	"strings"
	"unicode/utf8"
)

// Parse checks s against lang's alphabet and returns the Glyph it names. On any failure Parse
// returns the zero Glyph and a *ParseError describing why. Parse reads no source: it is a
// syntactic check only.
func Parse(lang Language, s string) (Glyph, error) {
	if lang != Go {
		return Glyph{}, &ParseError{Lang: lang, Input: s, Reason: ReasonUnsupportedLanguage, Detail: string(lang)}
	}

	if !utf8.ValidString(s) {
		return Glyph{}, &ParseError{Lang: lang, Input: s, Reason: ReasonInvalidUTF8}
	}

	if strings.Count(s, "#") > 1 {
		return Glyph{}, &ParseError{Lang: lang, Input: s, Reason: ReasonMultipleSeparators, Detail: s}
	}

	unit, member, ok := splitGlyph(s)
	if !ok {
		return Glyph{}, &ParseError{Lang: lang, Input: s, Reason: ReasonNoSeparator}
	}

	switch lang {
	case Go:
		return parseGo(s, unit, member)
	default:
		// Unreachable: the lang != Go check above has already rejected every non-Go value. This arm
		// exists only to satisfy Go's terminating-statement rule for the switch.
		return Glyph{}, &ParseError{Lang: lang, Input: s, Reason: ReasonUnsupportedLanguage, Detail: string(lang)}
	}
}

// splitGlyph divides s at its first "#" into unit and member. ok is false when s has no "#" at
// all. splitGlyph is language-free: it is what every alphabet's Parse dispatch reuses. Parse calls
// splitGlyph only after its own count check has confirmed s carries exactly one "#", so ok is false
// here only in the zero-separator case.
func splitGlyph(s string) (unit, member string, ok bool) {
	return strings.Cut(s, "#")
}
