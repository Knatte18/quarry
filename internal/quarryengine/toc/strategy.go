// strategy.go declares the per-language extraction contract, Strategy, and the registration and
// lookup machinery concrete strategies use to make themselves discoverable: Register, StrategyFor,
// and Implemented. Concrete strategies register themselves from their own file's init, so the set
// Implemented reports is derived from what actually compiled in rather than from a hand-maintained
// slice.

package toc

import (
	"sort"

	ts "github.com/tree-sitter/go-tree-sitter"
)

// Strategy is the per-language extraction contract every toc-supported language implements. It is
// designed to accommodate all five languages the toc survey covers (Go, Python, C#, TypeScript,
// Rust), even though only three register a concrete Strategy in this task.
type Strategy interface {
	// Language returns the canonical language name this Strategy registers under.
	Language() string

	// Symbols returns every listable declaration under root, in source order ascending by Start.
	// It never descends into a function or method body — only container nodes (types, namespaces,
	// classes) are walked for nested declarations. Each returned Symbol's Docstring is the full,
	// untrimmed docstring; sentence trimming is the entry point's job, not the strategy's. SigEnd is
	// set per the language-specific derivation, or left zero for a symbol with no body. Symbols
	// returns an empty, non-nil slice when the file has no listable declaration.
	Symbols(root *ts.Node, src []byte) []Symbol

	// Header returns the untruncated, delimiter-stripped prose of the file's first non-directive
	// comment block, or "" when the file has none. First-paragraph truncation is the entry point's
	// job, not the strategy's, so both toc file and toc dir share one truncation call site
	// (FirstParagraph).
	Header(root *ts.Node, src []byte) string

	// Package returns the file's declared package or namespace name, or "" for a language with no
	// such concept and for a file that declares none. It never derives a name from the filename or
	// the directory — the field reports only what the file itself declares.
	Package(root *ts.Node, src []byte) string

	// Generated reports whether the file's leading comment matches this language's generated-file
	// banner convention. known is false for a language with no reliable rule; the caller must then
	// omit the "generated" key entirely rather than emit a false generated value.
	Generated(root *ts.Node, src []byte) (generated bool, known bool)

	// TestFile reports whether base (a file's base name) names a test file by this language's
	// toolchain or framework convention. known is false for a language with no reliable rule; the
	// caller must then omit the "test" key entirely rather than emit a false isTest value.
	TestFile(base string) (isTest bool, known bool)
}

// strategies is the unexported package-level registry: canonical language name to its registered
// Strategy.
var strategies = make(map[string]Strategy)

// Register adds s to the package registry under s.Language(). Register panics on a duplicate
// registration for the same language name — that can only be a programming error at package-init
// time, since every concrete strategy registers itself exactly once from its own file's init.
func Register(s Strategy) {
	lang := s.Language()
	if _, exists := strategies[lang]; exists {
		panic("toc: duplicate Strategy registration for language " + lang)
	}
	strategies[lang] = s
}

// StrategyFor returns the registered Strategy for lang, and false if no strategy is registered
// under that name.
func StrategyFor(lang string) (Strategy, bool) {
	s, ok := strategies[lang]
	return s, ok
}

// Implemented returns the sorted set of canonical language names with a registered Strategy, so
// callers can distinguish "designed but not implemented" from "unknown extension" without keeping a
// second, hand-maintained list in sync with the registry.
func Implemented() []string {
	names := make([]string, 0, len(strategies))
	for name := range strategies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// swapRegistry replaces the package-level strategy registry with m and returns the map it replaced.
// It exists solely as a _test.go seam: the registry has no unregister path, so a test that exercises
// Register's duplicate-registration panic must register a fake language to trigger it, and that fake
// would otherwise leak into every later test in this package's binary — including
// TestImplemented_MatchesRegisteredStrategies (added in a later batch), which asserts Implemented()
// is exactly csharp, go, python and runs after classify_test.go in file order. A test using this seam
// must take a copy of the current map, install it via swapRegistry, register its fake language into
// the copy, and restore the original with t.Cleanup — otherwise the fake registration survives past
// that test and fails TestImplemented_MatchesRegisteredStrategies instead, the hardest kind of
// failure to attribute back to its cause.
//
// Production code must never call swapRegistry.
func swapRegistry(m map[string]Strategy) map[string]Strategy {
	previous := strategies
	strategies = m
	return previous
}
