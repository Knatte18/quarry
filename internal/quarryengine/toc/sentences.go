// sentences.go implements FirstSentences, the sentence-boundary docstring trimming rule the toc
// package doc comment documents in full. This file imports only strings and unicode.

package toc

import (
	"strings"
	"unicode"
)

// sentenceAbbreviations is the closed, explicit list of abbreviations whose trailing "." never ends
// a sentence. Matched case-insensitively against the token ending at the candidate terminator. See
// the package doc comment for the rule's rationale — "e.g." and "i.e." are common in Go doc
// comments, so without this exception the rule would split mid-sentence.
var sentenceAbbreviations = []string{
	"e.g.", "i.e.", "cf.", "vs.", "etc.", "resp.", "approx.",
}

// FirstSentences returns the first n sentences of text, an already-delimiter-stripped docstring.
//
// n == AllSentences returns text unchanged. n <= 0 returns "" — the caller is responsible for then
// omitting the docstring key rather than emitting an empty string. n greater than the number of
// sentences in text returns the whole of text; that is not an error.
//
// A sentence ends at a '.', '!', or '?' that is followed by whitespace or end-of-string, except when
// that terminator belongs to one of exactly three excluded shapes: a known abbreviation from
// sentenceAbbreviations; a single-letter initial, i.e. a '.' whose preceding token is one letter
// (e.g. "A."); or a terminator inside a backtick-quoted span, tracked by counting backticks while
// scanning — an unpaired trailing backtick leaves the remainder of text inside the span, which is
// the safe direction, since it under-splits rather than splitting mid-identifier.
//
// FirstSentences preserves the original inter-sentence spacing exactly: it returns text's own prefix
// up to and including the nth terminator, then strings.TrimSpace's the result. It never re-joins
// split pieces with a synthesized separator, which would rewrite a docstring's newlines into spaces.
func FirstSentences(text string, n int) string {
	if n == AllSentences {
		return text
	}
	if n <= 0 {
		return ""
	}

	runes := []rune(text)
	inBackticks := false
	sentenceCount := 0
	cutAt := -1

	for i, r := range runes {
		switch r {
		case '`':
			inBackticks = !inBackticks
			continue
		case '.', '!', '?':
			// fall through to the terminator check below
		default:
			continue
		}

		if inBackticks {
			continue
		}
		if !terminatorFollowedByBoundary(runes, i) {
			continue
		}
		if r == '.' && isAbbreviationEnd(runes, i) {
			continue
		}
		if r == '.' && isSingleLetterInitial(runes, i) {
			continue
		}

		sentenceCount++
		if sentenceCount == n {
			cutAt = i
			break
		}
	}

	if cutAt == -1 {
		// Fewer than n sentences in the whole text: return it whole, per the "not an error" rule.
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(string(runes[:cutAt+1]))
}

// terminatorFollowedByBoundary reports whether the rune at index i in runes is followed by
// whitespace or end-of-string, the shape a sentence-ending terminator must have.
func terminatorFollowedByBoundary(runes []rune, i int) bool {
	if i+1 >= len(runes) {
		return true
	}
	return unicode.IsSpace(runes[i+1])
}

// isAbbreviationEnd reports whether the text ending at the '.' at index i in runes ends with one of
// sentenceAbbreviations, case-insensitively, at a word boundary (the character preceding the
// abbreviation, if any, is neither a letter nor a digit).
func isAbbreviationEnd(runes []rune, i int) bool {
	lower := []rune(strings.ToLower(string(runes[:i+1])))
	for _, abbr := range sentenceAbbreviations {
		abbrRunes := []rune(abbr)
		if len(lower) < len(abbrRunes) {
			continue
		}
		if string(lower[len(lower)-len(abbrRunes):]) != abbr {
			continue
		}
		boundaryIdx := len(lower) - len(abbrRunes) - 1
		if boundaryIdx < 0 {
			return true
		}
		before := lower[boundaryIdx]
		if !unicode.IsLetter(before) && !unicode.IsDigit(before) {
			return true
		}
	}
	return false
}

// isSingleLetterInitial reports whether the '.' at index i in runes is preceded by exactly one
// letter that is itself preceded by whitespace, start-of-string, or another '.', the shape of a
// single-letter initial such as "A." in "A. Smith wrote this.".
func isSingleLetterInitial(runes []rune, i int) bool {
	if i == 0 || !unicode.IsLetter(runes[i-1]) {
		return false
	}
	if i-1 > 0 {
		before := runes[i-2]
		if unicode.IsLetter(before) || unicode.IsDigit(before) {
			return false
		}
	}
	return true
}
