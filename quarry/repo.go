// repo.go declares Repo, the facade's entry point, Open, which constructs one, and the TOC query
// that delegates to the engine unchanged.

package quarry

import (
	"fmt"

	"github.com/Knatte18/quarry/internal/engine"
)

// Repo is an opened repository handle wrapping the engine's own, plus the repository root the
// constructor was given. It is safe for concurrent use by multiple goroutines: the engine field
// holds only the repository root string and reads the filesystem fresh on every query, and root is
// read-only after construction, so neither field changes once Open returns.
type Repo struct {
	engine *engine.Repo
	root   string
}

// Open returns a Repo rooted at root. root must be an absolute path naming an existing directory.
// Open performs no git discovery and no cwd resolution, because the engine performs neither and the
// facade adds no behaviour of its own — with one exception, DeltaGit: a caller-facing convenience
// over the git layer, not query behaviour, which exists so the primary Go consumer is not forced to
// reimplement the one thing that layer exists to hold. root discovery is the CLI's job
// (internal/cli, batch 3).
func Open(root string) (*Repo, error) {
	er, err := engine.Open(root)
	if err != nil {
		return nil, fmt.Errorf("quarry: open %q: %w", root, err)
	}
	return &Repo{engine: er, root: root}, nil
}

// TOC answers a table-of-contents query for target, a repository-relative path with "" and "." both
// meaning the root, per opts. TOC applies no filtering, no re-shaping and no defaulting beyond what
// TOCOptions already encodes: it returns the engine's own answer and the engine's own error
// unchanged, so errors.Is(err, ErrTargetNotFound) and errors.Is(err, ErrTargetOutsideRepo) succeed
// against the facade's sentinels for a caller that never imports the engine.
func (r *Repo) TOC(target string, opts TOCOptions) (DirAnswer, error) {
	return r.engine.TOC(target, opts)
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
