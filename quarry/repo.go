// repo.go declares Repo, the facade's entry point, Open, which constructs one, the TOC query that
// delegates to the engine unchanged, the Glyphs query composing TOC with a projection, and
// GlyphsOptions, the frozen option value Glyphs queries under and internal/cli's own preset
// compares itself against.

package quarry

import (
	"fmt"

	"github.com/Knatte18/quarry/internal/engine"
)

// Repo is an opened repository handle wrapping the engine's own. It is safe for concurrent use by
// multiple goroutines, because it holds only the engine handle, which itself holds only the
// repository root string and reads the filesystem fresh on every query.
type Repo struct {
	engine *engine.Repo
}

// Open returns a Repo rooted at root. root must be an absolute path naming an existing directory.
// Open performs no git discovery and no cwd resolution, because the engine performs neither and the
// facade adds no behaviour of its own — root discovery is the CLI's job (internal/cli, batch 3).
func Open(root string) (*Repo, error) {
	er, err := engine.Open(root)
	if err != nil {
		return nil, fmt.Errorf("quarry: open %q: %w", root, err)
	}
	return &Repo{engine: er}, nil
}

// TOC answers a table-of-contents query for target, a repository-relative path with "" and "." both
// meaning the root, per opts. TOC applies no filtering, no re-shaping and no defaulting beyond what
// TOCOptions already encodes: it returns the engine's own answer and the engine's own error
// unchanged, so errors.Is(err, ErrTargetNotFound) and errors.Is(err, ErrTargetOutsideRepo) succeed
// against the facade's sentinels for a caller that never imports the engine.
func (r *Repo) TOC(target string, opts TOCOptions) (DirAnswer, error) {
	return r.engine.TOC(target, opts)
}

// GlyphsOptions returns the frozen options the glyphs preset expands to: Depth: DepthAll and
// Symbols pointing to a true. It returns a fresh value on each call — including a fresh *bool for
// Symbols — so a caller cannot mutate a shared one through the returned pointer.
//
// GlyphsOptions is exported rather than unexported because internal/cli's own drift test parses
// the CLI's preset tokens and compares that parse against this value, and that test cannot live in
// this package: internal/cli imports quarry, and the reverse import would be a cycle. See the
// overview's glyphs-options-is-exported Shared Decision.
func GlyphsOptions() TOCOptions {
	symbols := true
	return TOCOptions{Depth: DepthAll, Symbols: &symbols}
}

// Glyphs answers a glyphs query for target, a repository-relative path with "" and "." both
// meaning the root: it maps an empty target to ".", queries TOC under GlyphsOptions, and projects
// the result through GlyphView. On a non-nil error it returns the zero GlyphsAnswer and that error
// unchanged, so errors.Is(err, ErrTargetNotFound) keeps working through it exactly as it does
// through TOC.
//
// The normalisation happens here, not in GlyphView, because TOC accepts "" and "." as the same
// query: without normalising here, one query would have two spellings in its own answer's Target
// field, and GlyphView is deliberately a verbatim echo of what it is handed.
func (r *Repo) Glyphs(target string) (GlyphsAnswer, error) {
	normalised := target
	if normalised == "" {
		normalised = "."
	}
	a, err := r.TOC(normalised, GlyphsOptions())
	if err != nil {
		return GlyphsAnswer{}, err
	}
	return GlyphView(normalised, a), nil
}

// Resolve answers every target in targets, positionally, exactly as the engine's own Resolve does:
// it returns the engine's own result slice and the engine's own error unchanged — no filtering, no
// re-shaping, no defaulting.
//
// Resolve keeps the engine's multi-target signature rather than the command line's one-target rule:
// a Go caller batches many glyphs in one call and pays one parse per unit, which is the performance
// property this facade exists to preserve.
func (r *Repo) Resolve(targets []string) ([]ResolveResult, error) {
	return r.engine.Resolve(targets)
}

// Expand answers target — the target type's own head, plus every member whose owner chain begins
// with it — exactly as the engine's own Expand does: it returns the engine's own answer and the
// engine's own error unchanged. errors.As(err, &notType) against *NotATypeError therefore succeeds
// for a caller that never imports the engine.
func (r *Repo) Expand(target string) (ExpandAnswer, error) {
	return r.engine.Expand(target)
}
