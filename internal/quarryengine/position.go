// position.go implements the caller-facing Position type.
// It is ported from the recovered tools/scout-poc/gopls.go harness (git show 3b4dcf86) but
// decoupled from go/token: the engine never parses Go source itself, so a position here is whatever
// a caller supplies via a "file:line:col" CLI argument, not a go/token.Position derived from
// loading a package graph.

package quarryengine

// Position is a caller-supplied source location: a 1-based line and a 1-based byte column into
// File, exactly as parsed from a "file:line:col" CLI argument.
// It is the engine's language-agnostic stand-in for go/token.Position — no package graph is loaded
// to produce one.
type Position struct {
	File      string
	Line      int
	Character int
}
