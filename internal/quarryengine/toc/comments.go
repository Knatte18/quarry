// comments.go holds the shared text rules every strategy and entry point uses to turn raw comment
// source into prose: comment-delimiter stripping (StripLineComment, StripComment) and
// first-paragraph truncation (FirstParagraph). Every function here is a pure text-in/text-out
// transform with no I/O and no tree-sitter node in its signature.

package toc

import "strings"

// StripLineComment removes one leading occurrence of prefix (e.g. "//") from each line of text,
// after trimming that line's leading whitespace, then trims the result and rejoins the lines with
// "\n", and trims the whole. A line that is exactly the bare prefix becomes an empty line — that is
// what lets FirstParagraph's blank-line rule work uniformly across comment forms.
//
// An empty prefix is a supported call: StripLineComment then performs only the per-line trim, join,
// and whole trim, with no prefix removal.
func StripLineComment(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if prefix != "" {
			trimmed = strings.TrimPrefix(trimmed, prefix)
		}
		lines[i] = strings.TrimSpace(trimmed)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// StripComment is the delimiter-stripping entry point the Go strategy calls. It dispatches on the
// comment's form: when text's first non-whitespace characters are "/*", it removes the opening "/*"
// (and a "/**" variant's extra "*"), the closing "*/", and one leading "*" from each intermediate
// line, then applies the same per-line trim, join, and whole trim StripLineComment applies.
// Otherwise it delegates to StripLineComment(text, prefix) unchanged.
//
// This dispatch exists because tree-sitter emits a "/* ... */" block as a comment node exactly like
// a "//" line comment, so a block-form file header reaches the strip path with its delimiters
// intact. A prefix-only rule would leave "/*" and "*/" sitting in the emitted docstring and header
// text. Only the delimiters are removed here — no reflowing, no de-indentation beyond the shared
// per-line trim.
func StripComment(text, prefix string) string {
	trimmedStart := strings.TrimLeft(text, " \t\r\n")
	if !strings.HasPrefix(trimmedStart, "/*") {
		return StripLineComment(text, prefix)
	}

	body := trimmedStart
	body = strings.TrimPrefix(body, "/**")
	body = strings.TrimPrefix(body, "/*")
	body = strings.TrimSuffix(strings.TrimRight(body, " \t\r\n"), "*/")

	lines := strings.Split(body, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimPrefix(trimmed, "*")
		lines[i] = strings.TrimSpace(trimmed)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// FirstParagraph returns everything in text before the first empty line, or the whole of text when
// it contains no empty line, trimmed.
//
// FirstParagraph is applied to text that has already been delimiter-stripped, never to raw comment
// source — that ordering is the whole reason this one rule covers a "//" block and a "/* */" block
// without a per-form special case.
//
// Both TOCFile and TOCDir call FirstParagraph on the file header: the truncation is symmetric across
// the two verbs, and an "optimization" that skips it for one of them is a regression, not a
// simplification.
func FirstParagraph(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			return strings.TrimSpace(strings.Join(lines[:i], "\n"))
		}
	}
	return strings.TrimSpace(text)
}
