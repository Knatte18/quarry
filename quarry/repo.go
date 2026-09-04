// repo.go declares Repo, the facade's entry point, Open, which constructs one, and the TOC query
// that delegates to the engine unchanged.

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
