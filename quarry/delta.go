// delta.go holds the two delta methods — the pure Delta and the git convenience DeltaGit — and the
// one new type this facade declares rather than aliases: GitDeltaAnswer. Every other type in this
// package is an alias for an engine type; GitDeltaAnswer is not, because it carries the two revision
// strings DeltaGit computed its answer from, and the engine's own DeltaAnswer must stay ignorant of
// git — see GitDeltaAnswer's own doc comment for why.

package quarry

// GitDeltaAnswer wraps a DeltaAnswer with the two revision strings DeltaGit computed it from. The
// core's own DeltaAnswer carries no revision information at all: the engine knows nothing about git,
// and a field it can never populate would be a lie in its own type, so the echo lives here, exactly
// where the revisions are known.
//
// From and To are declared before the embedded DeltaAnswer so the emitted key order puts them
// first. To is a pointer so an absent revision — the after side is the working tree — marshals as
// an explicit JSON null rather than being omitted: the key's presence, with a null value, is itself
// the statement that the after side is the working tree.
type GitDeltaAnswer struct {
	// From is the before-side revision string, exactly as the caller gave it.
	From string `json:"from"`
	// To is the after-side revision string, exactly as the caller gave it, or nil for the working
	// tree.
	To *string `json:"to"`
	DeltaAnswer
}

// Delta compares two versions of a batch of files, exactly as the engine's own Delta does: it
// returns the engine's own answer and the engine's own error unchanged — no filtering, no
// re-shaping, no defaulting, exactly as the existing three query methods on this facade do.
func (r *Repo) Delta(entries []DeltaEntry) (DeltaAnswer, error) {
	return r.engine.Delta(entries)
}
