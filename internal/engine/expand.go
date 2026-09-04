// expand.go implements the exported expand verb: what a type consists of, across files. It holds
// Repo.Expand and its unexported worker, and NotATypeError, the one typed failure the verb returns.

package engine

import "fmt"

// NotATypeError is returned by Repo.Expand when target's glyph names a match set holding no
// KindType symbol — a function, a method, a const, a var, or the several-declaration "init" glyph,
// whatever its count. ID is the glyph's own String() form and Kind is the kind of the first match in
// file-then-line order.
//
// It is a struct rather than a bare sentinel because docs/rewrite-plan.md's expand rule — the glyph
// must name a type, and on any other kind the answer names the kind — requires the kind to be
// carried, and a caller mapping engine failures to a status word and an exit code needs it without
// parsing a message. That is the same argument that split ErrTargetOutsideRepo from
// ErrTargetNotFound in internal/engine/repo.go.
//
// NotATypeError is returned as Expand's own error and is never carried inside ExpandAnswer: an
// ok-plus-kind pair inside the payload would duplicate the envelope a later task owns, inside the
// data.
//
// NotATypeError is never returned under a unit collision, because the glyph does not unambiguously
// name anything there and naming a kind would be a claim the answer cannot support — a collision
// answers StatusAmbiguous instead, before the kind gate is ever reached.
type NotATypeError struct {
	// ID is the glyph's own String() form.
	ID string
	// Kind is the kind of the first match in file-then-line order.
	Kind Kind
}

// Error implements the error interface, naming the glyph and the kind it actually named.
func (e *NotATypeError) Error() string {
	return fmt.Sprintf("engine: expand %s: not a type, kind %s", e.ID, e.Kind)
}
