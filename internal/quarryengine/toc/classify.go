// classify.go holds the filename- and banner-based classification rules: IsDirectiveBlock (which
// leading comment blocks are directives, not a file header), TestFileByName, and
// GeneratedByBanner.

package toc

import "strings"

// IsDirectiveBlock reports whether stripped — an already-delimiter-stripped leading comment block
// starting at the 1-based line startLine — is a directive block for lang, and must therefore be
// skipped when looking for the file header.
//
// It returns true only when every non-empty line in stripped matches a known directive form for
// lang. A block mixing a directive line with a prose line is not a directive block. Empty lines
// inside the block are ignored when deciding.
//
// For Go, a directive line begins with "go:build", "+build", "go:generate", "go:embed", or
// "nolint", or matches the generated-file banner (the "Code generated " prefix together with the
// "DO NOT EDIT." suffix, per the toolchain's own convention). A language with no directive rule
// never has a directive block.
func IsDirectiveBlock(lang string, startLine int, stripped string) bool {
	lines := strings.Split(stripped, "\n")
	sawDirective := false
	for offset, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !isDirectiveLine(lang, startLine+offset, trimmed) {
			return false
		}
		sawDirective = true
	}
	return sawDirective
}

// isDirectiveLine reports whether one already-stripped, non-empty line matches a known directive
// form for lang, given the line's own 1-based lineNumber within the file.
func isDirectiveLine(lang string, lineNumber int, trimmed string) bool {
	switch lang {
	case "go":
		switch {
		case strings.HasPrefix(trimmed, "go:build"),
			strings.HasPrefix(trimmed, "+build"),
			strings.HasPrefix(trimmed, "go:generate"),
			strings.HasPrefix(trimmed, "go:embed"),
			strings.HasPrefix(trimmed, "nolint"):
			return true
		default:
			return isGeneratedBanner(trimmed)
		}
	default:
		return false
	}
}

// isGeneratedBanner reports whether trimmed matches the Go toolchain's generated-file banner
// convention: a line carrying the "Code generated " prefix together with the "DO NOT EDIT." suffix.
func isGeneratedBanner(trimmed string) bool {
	return strings.HasPrefix(trimmed, "Code generated ") && strings.HasSuffix(trimmed, "DO NOT EDIT.")
}

// TestFileByName reports whether base (a file's base name) names a test file for lang, by that
// language's toolchain convention rather than by mere style.
//
// A caller must never emit false for a language whose known is false — a false test flag is a lie
// a consumer cannot distinguish from a fact.
//
// For Go, known is true and isTest is the "_test.go" suffix, which the toolchain itself defines.
// A language with no rule reports known false.
func TestFileByName(lang, base string) (isTest, known bool) {
	switch lang {
	case "go":
		return strings.HasSuffix(base, "_test.go"), true
	default:
		return false, false
	}
}

// GeneratedByBanner reports whether leadingComment — a file's already-delimiter-stripped leading
// comment block — marks the file as generated, by lang's own banner convention.
//
// A caller must never emit false for a language whose known is false, for the same reason
// documented on TestFileByName. Note that being skipped as a header candidate (IsDirectiveBlock) and
// being consumed here as a generated-file marker are independent readings of the same leading
// comment block — the banner is both.
//
// For Go, known is true and generated is whether leadingComment matches the
// "Code generated ... DO NOT EDIT." banner. A language with no rule reports known false.
func GeneratedByBanner(lang, leadingComment string) (generated, known bool) {
	switch lang {
	case "go":
		for _, line := range strings.Split(leadingComment, "\n") {
			if isGeneratedBanner(strings.TrimSpace(line)) {
				return true, true
			}
		}
		return false, true
	default:
		return false, false
	}
}
