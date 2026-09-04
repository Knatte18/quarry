// match.go is the single place the harness matches a giveaway token against transcript, prompt, or
// answer content. It defines exactly two matching classes: bare identifier tokens (a tool name, the
// literal "quarry", the server name) match as whole words, case-insensitively, because a short token
// under substring matching would trip on ordinary prose; composed strings that already carry their own
// structure (an MCP prefix, a repository-root path, a worktree path) match as a case-sensitive
// substring, because word-boundary matching would not apply cleanly to a path or a prefix. The gates
// and the scorer's redactor both import this file so the definition of "the giveaway token appeared"
// cannot drift between them.
//
// The startup assertion on the resolved worktree root path is deliberately NOT one of these two
// classes and does not route through this file: it tests a filesystem path the harness is about to
// use, not content it is searching, it is deliberately case-insensitive where the composed-string
// form here is case-sensitive, and it has exactly one caller.

package ladder

import (
	"regexp"
	"strings"
)

// BareTokenPattern returns a case-insensitive, word-bounded regular expression matching token as a
// whole word. The token is regexp-quoted first, so a token that itself contains regular-expression
// metacharacters is matched literally rather than interpreted.
func BareTokenPattern(token string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(token) + `\b`)
}

// MatchesBareToken reports whether text contains token as a case-insensitive, word-bounded match. Use
// this for bare identifier tokens: tool names, the literal "quarry", and the server name.
func MatchesBareToken(text, token string) bool {
	return BareTokenPattern(token).MatchString(text)
}

// MatchesComposedString reports whether text contains s as a case-sensitive substring. Use this for
// composed strings that already carry internal structure, such as an MCP prefix or a repository-root
// or worktree path, where word-boundary matching does not apply.
func MatchesComposedString(text, s string) bool {
	return strings.Contains(text, s)
}

// BareTokenAlternation builds a single case-insensitive, word-bounded regular expression matching any
// one of tokens. Empty entries are skipped. It returns nil when tokens is empty or every entry is
// empty, since a regular expression that never matches would fail silently. The scorer's redactor and
// gate check (d) both build their tool-name alternation through this function; it is the only place in
// the package an alternation over tool names is constructed.
func BareTokenAlternation(tokens []string) *regexp.Regexp {
	quoted := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token == "" {
			continue
		}
		quoted = append(quoted, regexp.QuoteMeta(token))
	}
	if len(quoted) == 0 {
		return nil
	}
	return regexp.MustCompile(`(?i)\b(` + strings.Join(quoted, "|") + `)\b`)
}
