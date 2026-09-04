// treesitter.go is the parsing backend for the engine package: grammar loading, parser
// construction, and nothing else. It is the only place in the tree that constructs a *ts.Parser or
// a *ts.Tree — every other package that needs a parse tree goes through WithTree.

// Package treesitter wraps the tree-sitter C bindings behind a single parse-and-release seam,
// WithTree. It resolves a canonical language name to its compiled grammar, builds the parser and
// tree, hands the caller the parsed root node, and releases both underlying C allocations before
// returning — on the success route, the route where the caller's callback errors, and the route
// where the parse is merely partial (HasError()) alike. Callers must never retain the *ts.Node
// WithTree hands them past their own callback's return: the node is invalidated the moment
// WithTree closes the tree that owns it.
package treesitter

import (
	"fmt"
	"sort"

	ts "github.com/tree-sitter/go-tree-sitter"
	tsgo "github.com/tree-sitter/tree-sitter-go/bindings/go"

	// Blank-imported so cgoguard is a build-graph dependency of every package that links
	// tree-sitter: this makes the CGO_ENABLED=0 guard fire before this package's own cgo imports
	// would otherwise hit the raw linker error.
	_ "github.com/Knatte18/quarry/internal/cgoguard"
)

// grammars maps each canonical language name this package wires to the unsafe.Pointer its grammar
// module exports. NewLanguage is called lazily, once per WithTree call, rather than once at
// package init.
var grammars = map[string]func() *ts.Language{
	"go": func() *ts.Language { return ts.NewLanguage(tsgo.Language()) },
}

// onRelease is an unexported test seam: nil in production, and invoked from WithTree after both
// the parser's and the tree's Close calls have run. It exists solely so a test can observe that
// both C allocations were released on every route; production code must never assign it.
var onRelease func()

// Supported reports whether a grammar is wired for the canonical language name lang.
func Supported(lang string) bool {
	_, ok := grammars[lang]
	return ok
}

// Languages returns the sorted canonical names this package has a wired grammar for.
func Languages() []string {
	names := make([]string, 0, len(grammars))
	for name := range grammars {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// WithTree parses src as lang and calls fn with the resulting tree's root node and whether the
// parse was partial (tree.RootNode().HasError()).
//
// WithTree owns the whole parse lifecycle: it resolves the grammar, constructs a *ts.Parser,
// releases it with a deferred Close, sets the language, parses src, constructs the resulting
// *ts.Tree, and releases it with a deferred Close. Both defers are registered immediately after
// their allocation succeeds, so both C allocations are released on every route out of this
// function — fn returning nil, fn returning an error, and a partial parse alike. fn's error is
// returned unchanged.
//
// fn must never retain root beyond its own return: root is invalidated the moment WithTree closes
// the tree that owns it.
//
// An unknown lang returns a plain, unwrapped error naming lang and the wired set.
func WithTree(lang string, src []byte, fn func(root *ts.Node, partial bool) error) error {
	newLanguage, ok := grammars[lang]
	if !ok {
		return fmt.Errorf("treesitter: no grammar wired for language %q; wired languages: %v", lang, Languages())
	}

	// Registered before either Close defer below, so it runs last — LIFO defer ordering is what
	// guarantees onRelease observes both allocations already released, on every route out of this
	// function.
	defer func() {
		if onRelease != nil {
			onRelease()
		}
	}()

	parser := ts.NewParser()
	defer parser.Close()

	if err := parser.SetLanguage(newLanguage()); err != nil {
		return fmt.Errorf("treesitter: set language %q: %w", lang, err)
	}

	tree := parser.Parse(src, nil)
	if tree == nil {
		// Parse can return nil on a genuine parser failure; report it rather than let the caller
		// dereference a nil *ts.Tree.
		return fmt.Errorf("treesitter: parse produced no tree for language %q", lang)
	}
	defer tree.Close()

	root := tree.RootNode()
	return fn(root, root.HasError())
}
