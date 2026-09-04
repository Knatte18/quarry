// quarry.go declares the type aliases, Kind constants, DepthAll, and error sentinels that make the
// engine's answer shape and error identity nameable from outside this module.

package quarry

import "github.com/Knatte18/quarry/internal/engine"

// DirAnswer is an alias for engine.DirAnswer, not a defined type. Go enforces the internal rule on
// import paths, not on types reached through an alias, so an external importer can write
// var a quarry.DirAnswer without importing internal/engine.
type DirAnswer = engine.DirAnswer

// FileEntry is an alias for engine.FileEntry, for the same reason DirAnswer is: it makes the
// engine's file-entry shape nameable without an import of internal/engine.
type FileEntry = engine.FileEntry

// Symbol is an alias for engine.Symbol, for the same reason DirAnswer is: it makes the engine's
// symbol shape nameable without an import of internal/engine.
type Symbol = engine.Symbol

// Kind is an alias for engine.Kind, the closed vocabulary a Symbol's Kind field is drawn from, for
// the same reason DirAnswer is.
type Kind = engine.Kind

// TOCOptions is an alias for engine.TOCOptions, for the same reason DirAnswer is: it lets a caller
// build the options Repo.TOC accepts without an import of internal/engine.
type TOCOptions = engine.TOCOptions

// The five Kind values a toc query ever emits, aliased from the engine so a caller can name them
// without importing internal/engine.
const (
	// KindFunction marks a free function: a func with no receiver.
	KindFunction = engine.KindFunction
	// KindMethod marks a function bound to a receiver, including an interface method.
	KindMethod = engine.KindMethod
	// KindType marks a type-level declaration.
	KindType = engine.KindType
	// KindConst marks a package-level const declaration.
	KindConst = engine.KindConst
	// KindVar marks a package-level var declaration.
	KindVar = engine.KindVar
)

// DepthAll requests a TOC query recurse to the bottom of the tree, aliased from the engine so a
// caller can name it without importing internal/engine.
const DepthAll = engine.DepthAll

// ErrTargetNotFound is the engine's own ErrTargetNotFound value, not a copy, so errors.Is stays
// transitive across the facade: a caller checking errors.Is(err, quarry.ErrTargetNotFound) against
// an error returned by (*Repo).TOC succeeds without ever importing internal/engine.
var ErrTargetNotFound = engine.ErrTargetNotFound

// ErrTargetOutsideRepo is the engine's own ErrTargetOutsideRepo value, for the same reason
// ErrTargetNotFound is: it is the same value, not a copy, which is what keeps errors.Is transitive.
var ErrTargetOutsideRepo = engine.ErrTargetOutsideRepo

// ErrLanguageUnsupported is the engine's own ErrLanguageUnsupported value, for the same reason
// ErrTargetNotFound is: it is the same value, not a copy, which is what keeps errors.Is transitive.
var ErrLanguageUnsupported = engine.ErrLanguageUnsupported
