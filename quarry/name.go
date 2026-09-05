// name.go declares Name, the package-level batch entry point over the engine's glyph maker.

package quarry

import "github.com/Knatte18/quarry/internal/engine"

// Name predicts the id and kind for every declaration in decls, positionally, delegating to
// engine.Name and returning its result unchanged: no filtering, no re-shaping, no defaulting,
// matching the delegation posture every other query in this package already keeps.
//
// Name is a package-level function rather than a Repo method because the maker performs no I/O, and
// a receiver would claim a repository dependency the query does not have — a caller holding only a
// declaration fragment should not have to first open a directory that has nothing to do with the
// answer.
//
// Name returns no error: with no I/O nothing can fail batch-wide, every failure is a property of one
// entry and is carried in that entry's own NameResult, and a returned error would have no value to
// carry while inviting a caller to abandon results that are perfectly good.
//
// The result is positional and always the same length as decls; an empty input returns an empty,
// non-nil slice.
func Name(decls []Declaration) []NameResult {
	return engine.Name(decls)
}
