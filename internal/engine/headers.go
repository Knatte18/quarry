// headers.go holds the per-format rules that turn a non-code file's raw bytes into the prose that
// becomes its header: one leading heading-plus-paragraph rule for Markdown, and one leading-comment
// rule per remaining comment syntax the walk recognizes. Every rule is pure: bytes in, prose out, no
// I/O, no tree-sitter node in its signature. A Go file's header comes from the Go strategy, never
// from here.

package engine

import "strings"

// headerRule extracts the header prose from a non-code file's raw source bytes. Every concrete rule
// returns the first paragraph only, via FirstParagraph, so a long leading comment does not become
// the header; leading blank lines are skipped before a rule looks for its delimiter, and a comment
// that does not start the file is not a header.
type headerRule func(src []byte) string

// markdownHeader returns the file's first ATX ("#"-prefixed) or setext (underlined with "=" or "-")
// heading, then the first paragraph that follows it, joined by a newline. A file with no heading
// returns the empty string.
func markdownHeader(src []byte) string {
	lines := strings.Split(string(src), "\n")

	i := skipBlankLines(lines, 0)
	if i >= len(lines) {
		return ""
	}

	var heading string
	trimmed := strings.TrimSpace(lines[i])
	switch {
	case strings.HasPrefix(trimmed, "#"):
		heading = strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		i++
	case i+1 < len(lines) && isSetextUnderline(lines[i+1]):
		heading = trimmed
		i += 2
	default:
		// Neither an ATX nor a setext heading opens the file: there is no heading to report, and
		// this rule never falls back to treating an ordinary paragraph as one.
		return ""
	}

	i = skipBlankLines(lines, i)
	if i >= len(lines) {
		return heading
	}

	paragraph := FirstParagraph(strings.Join(lines[i:], "\n"))
	if paragraph == "" {
		return heading
	}
	return heading + "\n" + paragraph
}

// skipBlankLines returns the index of the first line at or after i that is not all whitespace, or
// len(lines) when every remaining line is blank.
func skipBlankLines(lines []string, i int) int {
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	return i
}

// isSetextUnderline reports whether line, once trimmed, is non-empty and consists entirely of "="
// characters or entirely of "-" characters — the two setext-heading underline forms.
func isSetextUnderline(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	return strings.Count(trimmed, "=") == len(trimmed) || strings.Count(trimmed, "-") == len(trimmed)
}

// htmlCommentHeader returns the prose of a leading "<!-- ... -->" comment, delimiters stripped. A
// file that does not open with that delimiter has no header.
func htmlCommentHeader(src []byte) string {
	inner, ok := leadingDelimitedComment(string(src), "<!--", "-->")
	if !ok {
		return ""
	}
	return FirstParagraph(StripLineComment(inner, ""))
}

// cssCommentHeader returns the prose of a leading "/* ... */" comment.
func cssCommentHeader(src []byte) string {
	inner, ok := leadingDelimitedComment(string(src), "/*", "*/")
	if !ok {
		return ""
	}
	return FirstParagraph(stripBlockCommentBody(inner))
}

// scriptCommentHeader returns the prose of a leading run of "//" lines, or of a leading
// "/* ... */" comment, whichever the file starts with.
func scriptCommentHeader(src []byte) string {
	text := strings.TrimLeft(string(src), " \t\r\n")
	if strings.HasPrefix(text, "/*") {
		return cssCommentHeader([]byte(text))
	}

	var run []string
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "//") {
			break
		}
		run = append(run, trimmed)
	}
	if len(run) == 0 {
		return ""
	}
	return FirstParagraph(StripLineComment(strings.Join(run, "\n"), "//"))
}

// hashBlockHeader returns the prose of a leading run of "#"-prefixed lines, skipping a shebang line
// ("#!") when it is the first line.
func hashBlockHeader(src []byte) string {
	lines := strings.Split(strings.TrimLeft(string(src), " \t\r\n"), "\n")
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "#!") {
		lines = lines[1:]
	}

	var run []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			break
		}
		run = append(run, trimmed)
	}
	if len(run) == 0 {
		return ""
	}
	return FirstParagraph(StripLineComment(strings.Join(run, "\n"), "#"))
}

// leadingDelimitedComment reports the content between a leading open/close delimiter pair — e.g.
// "<!--"/"-->" or "/*"/"*/" — after skipping leading blank characters. ok is false when text does
// not open with openDelim, or when no closeDelim follows it.
func leadingDelimitedComment(text, openDelim, closeDelim string) (inner string, ok bool) {
	trimmed := strings.TrimLeft(text, " \t\r\n")
	if !strings.HasPrefix(trimmed, openDelim) {
		return "", false
	}
	rest := trimmed[len(openDelim):]
	idx := strings.Index(rest, closeDelim)
	if idx < 0 {
		return "", false
	}
	return rest[:idx], true
}

// stripBlockCommentBody trims each line of a "/* ... */" comment's already-delimiter-stripped body,
// removing one leading "*" per line the way a conventionally aligned block comment indents its
// continuation lines, then trims the whole.
func stripBlockCommentBody(body string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimPrefix(trimmed, "*")
		lines[i] = strings.TrimSpace(trimmed)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
