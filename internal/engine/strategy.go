// strategy.go declares the per-language extraction contract, Strategy, and the registration and
// lookup machinery concrete strategies use to make themselves discoverable: Register, StrategyFor,
// and Implemented. Concrete strategies register themselves from their own file's init, so the set
// Implemented reports is derived from what actually compiled in rather than from a hand-maintained
// slice.

package engine

import (
	"sort"

	ts "github.com/tree-sitter/go-tree-sitter"
)

// Strategy is the per-language extraction contract a supported language implements.
type Strategy interface {
	// Language returns the canonical language name this Strategy registers under.
	Language() string

	// Symbols returns every listable declaration under root, in source order ascending by Start.
	// unit is the glyph unit every symbol in this file belongs to — the Go package directory, or its
	// "_test"-suffixed external-test counterpart — and the strategy builds each symbol's Glyph and
	// ID from it. A strategy never derives the unit itself: the unit is a directory-level fact,
	// established once by the walk over every file in a directory, not a file-level one a single
	// file's own content could establish. Symbols never descends into a function or method body —
	// only container nodes (types, namespaces, classes) are walked for nested declarations. SigEnd
	// is set per the language-specific derivation, or left zero for a symbol with no body. Symbols
	// returns an empty, non-nil slice when the file has no listable declaration.
	Symbols(unit string, root *ts.Node, src []byte) []Symbol

	// Header returns the untruncated, delimiter-stripped prose of the file's first non-directive
	// comment block, or "" when the file has none. First-paragraph truncation is the entry point's
	// job, not the strategy's, so both toc file and toc dir share one truncation call site
	// (FirstParagraph).
	Header(root *ts.Node, src []byte) string

	// Package returns the file's declared package or namespace name, or "" for a language with no
	// such concept and for a file that declares none. It never derives a name from the filename or
	// the directory — the field reports only what the file itself declares.
	Package(root *ts.Node, src []byte) string

	// PackageDoc returns the file's package documentation, or "" when this file carries none. This
	// is a different rule from Header: Header returns the first non-directive leading comment
	// block, wherever it sits, while PackageDoc returns only a block that is both immediately above
	// the package clause and recognisable as package documentation by the language's own
	// convention. Both are needed because a file can carry one, the other, or both — a file header
	// and a package doc comment are two separate leading blocks, and only PackageDoc's stricter
	// rule tells them apart.
	PackageDoc(root *ts.Node, src []byte) string

	// Generated reports whether the file's leading comment matches this language's generated-file
	// banner convention. With Go the only language and its own generated-file rule always known,
	// there is no caller left for a "rule not known for this language" signal; a second language
	// with no reliable rule reintroduces that signal and its return value together, rather than
	// carrying an always-true known flag no one reads.
	Generated(root *ts.Node, src []byte) bool

	// TestFile reports whether base (a file's base name) names a test file by this language's
	// toolchain or framework convention. See Generated's doc comment for why this has one return
	// value rather than the (isTest, known) pair a language with no reliable rule would need.
	TestFile(base string) bool
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
		panic("engine: duplicate Strategy registration for language " + lang)
	}
	strategies[lang] = s
}

// StrategyFor returns the registered Strategy for lang, and false if no strategy is registered
// under that name.
func StrategyFor(lang string) (Strategy, bool) {
	s, ok := strategies[lang]
	return s, ok
}

// Implemented returns the sorted set of canonical language names with a registered Strategy.
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
// TestImplemented_MatchesRegisteredStrategies, which asserts Implemented()
// is exactly go and runs after classify_test.go in file order. A test using this seam
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
