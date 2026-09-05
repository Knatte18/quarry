// quarry.go declares the type aliases, Kind constants, DepthAll, and error sentinels that make the
// engine's answer shape and error identity, and the git layer's own error identity, nameable from
// outside this module.

package quarry

import (
	"github.com/Knatte18/quarry/internal/engine"
	"github.com/Knatte18/quarry/internal/gitsrc"
)

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

// ResolveResult is an alias for engine.ResolveResult, for the same reason DirAnswer is: Go enforces
// the internal rule on import paths, not on types reached through an alias, so an external importer
// can name the resolve verb's answer shape without importing internal/engine.
type ResolveResult = engine.ResolveResult

// ExpandAnswer is an alias for engine.ExpandAnswer, for the same reason DirAnswer is: it makes the
// expand verb's answer shape nameable without an import of internal/engine.
type ExpandAnswer = engine.ExpandAnswer

// Status is an alias for engine.Status, the closed per-entry vocabulary both ResolveResult and
// ExpandAnswer draw from, for the same reason DirAnswer is.
type Status = engine.Status

// NotATypeError is an alias for engine.NotATypeError, not a re-declaration, so that
// errors.As(err, &notType) against *quarry.NotATypeError succeeds for a caller that never imports
// the engine — the same transitivity argument ErrTargetNotFound's and ErrTargetOutsideRepo's own
// doc comments make below for errors.Is.
type NotATypeError = engine.NotATypeError

// SelfGlyphError is an alias for engine.SelfGlyphError, not a re-declaration, so that
// errors.As(err, &selfGlyph) against *quarry.SelfGlyphError succeeds for a caller that never
// imports the engine — the same transitivity argument NotATypeError's own doc comment makes above.
type SelfGlyphError = engine.SelfGlyphError

// The four Status values a resolve or expand query ever emits, aliased from the engine so a caller
// can name them without importing internal/engine.
const (
	// StatusFound marks exactly one matching declaration.
	StatusFound = engine.StatusFound
	// StatusNotFound marks no matching declaration.
	StatusNotFound = engine.StatusNotFound
	// StatusAmbiguous marks several different declarations matching, with nothing chosen between
	// them.
	StatusAmbiguous = engine.StatusAmbiguous
	// StatusMultipart marks one symbol the language lets be declared in several places, with every
	// part returned.
	StatusMultipart = engine.StatusMultipart
)

// ErrTargetNotFound is the engine's own ErrTargetNotFound value, not a copy, so errors.Is stays
// transitive across the facade: a caller checking errors.Is(err, quarry.ErrTargetNotFound) against
// an error returned by (*Repo).TOC succeeds without ever importing internal/engine.
var ErrTargetNotFound = engine.ErrTargetNotFound

// ErrTargetOutsideRepo is the engine's own ErrTargetOutsideRepo value, for the same reason
// ErrTargetNotFound is: it is the same value, not a copy, which is what keeps errors.Is transitive.
var ErrTargetOutsideRepo = engine.ErrTargetOutsideRepo

// ErrTargetHasSeparator is the engine's own ErrTargetHasSeparator value, for the same reason
// ErrTargetNotFound is: it is the same value, not a copy, which is what keeps errors.Is transitive.
var ErrTargetHasSeparator = engine.ErrTargetHasSeparator

// ErrLanguageUnsupported is the engine's own ErrLanguageUnsupported value, for the same reason
// ErrTargetNotFound is: it is the same value, not a copy, which is what keeps errors.Is transitive.
var ErrLanguageUnsupported = engine.ErrLanguageUnsupported

// DeltaEntry is an alias for engine.DeltaEntry, for the same reason DirAnswer is: it lets a caller
// build the batch (*Repo).Delta accepts without an import of internal/engine.
type DeltaEntry = engine.DeltaEntry

// DeltaAnswer is an alias for engine.DeltaAnswer, for the same reason DirAnswer is: it makes the
// pure delta core's answer shape nameable without an import of internal/engine.
type DeltaAnswer = engine.DeltaAnswer

// DeltaFile is an alias for engine.DeltaFile, for the same reason DirAnswer is: it makes one input
// entry's echoed disposition nameable without an import of internal/engine.
type DeltaFile = engine.DeltaFile

// Disposition is an alias for engine.Disposition, the closed vocabulary a DeltaFile's Disposition
// field is drawn from, for the same reason DirAnswer is.
type Disposition = engine.Disposition

// ChangedDimension is an alias for engine.ChangedDimension, the closed vocabulary a ModifiedSymbol's
// Changed array draws its elements from, for the same reason DirAnswer is.
type ChangedDimension = engine.ChangedDimension

// SymbolLocation is an alias for engine.SymbolLocation, for the same reason DirAnswer is: it makes a
// modified symbol's before-side location block nameable without an import of internal/engine.
type SymbolLocation = engine.SymbolLocation

// ModifiedSymbol is an alias for engine.ModifiedSymbol, for the same reason DirAnswer is: it makes
// one symbol table key's before/after divergence nameable without an import of internal/engine.
type ModifiedSymbol = engine.ModifiedSymbol

// RenamedPair is an alias for engine.RenamedPair, for the same reason DirAnswer is: it makes one
// exact-tier rename pairing nameable without an import of internal/engine.
type RenamedPair = engine.RenamedPair

// RenameCandidateEntry is an alias for engine.RenameCandidateEntry, for the same reason DirAnswer
// is: it makes one deleted symbol's evidence-tier candidate block nameable without an import of
// internal/engine.
type RenameCandidateEntry = engine.RenameCandidateEntry

// RenameCandidate is an alias for engine.RenameCandidate, for the same reason DirAnswer is: it makes
// one evidence-tier candidate pairing nameable without an import of internal/engine.
type RenameCandidate = engine.RenameCandidate

// RenameSignals is an alias for engine.RenameSignals, for the same reason DirAnswer is: it makes one
// candidate pair's mechanical evidence nameable without an import of internal/engine.
type RenameSignals = engine.RenameSignals

// The five Disposition values a delta batch ever emits, aliased from the engine so a caller can name
// them without importing internal/engine.
const (
	// DispositionAdded marks a file present only on the after side.
	DispositionAdded = engine.DispositionAdded
	// DispositionRemoved marks a file present only on the before side.
	DispositionRemoved = engine.DispositionRemoved
	// DispositionChanged marks a file present on both sides and extracted on each.
	DispositionChanged = engine.DispositionChanged
	// DispositionUnsupported marks a file whose extension resolves to no registered strategy.
	DispositionUnsupported = engine.DispositionUnsupported
	// DispositionError marks an entry that was refused before extraction, or that failed extraction.
	DispositionError = engine.DispositionError
)

// The four ChangedDimension values, in the order a changed array reports them, aliased from the
// engine so a caller can name them without importing internal/engine.
const (
	// ChangedBody marks a difference in the body token stream.
	ChangedBody = engine.ChangedBody
	// ChangedSignature marks a difference in the signature token stream.
	ChangedSignature = engine.ChangedSignature
	// ChangedDoc marks a difference in the docstring text.
	ChangedDoc = engine.ChangedDoc
	// ChangedFile marks a difference in the file the symbol was extracted from.
	ChangedFile = engine.ChangedFile
)

// ErrNotARepository is the git layer's own ErrNotARepository value, not a copy, so errors.Is stays
// transitive across the facade: a caller checking errors.Is(err, quarry.ErrNotARepository) against
// an error returned by (*Repo).DeltaGit succeeds without ever importing internal/gitsrc.
var ErrNotARepository = gitsrc.ErrNotARepository

// ErrRootNotTopLevel is the sentinel *RootNotTopLevelError wraps, aliased from the git layer for the
// same reason ErrNotARepository is: it is the same value, not a copy, which is what keeps errors.Is
// transitive.
var ErrRootNotTopLevel = gitsrc.ErrRootNotTopLevel

// ErrUnknownRevision is the sentinel *UnknownRevisionError wraps, aliased from the git layer for the
// same reason ErrNotARepository is: it is the same value, not a copy, which is what keeps errors.Is
// transitive.
var ErrUnknownRevision = gitsrc.ErrUnknownRevision

// RootNotTopLevelError is an alias for gitsrc.RootNotTopLevelError, not a re-declaration, so that
// errors.As(err, &rootErr) against *quarry.RootNotTopLevelError succeeds for a caller that never
// imports internal/gitsrc — the same transitivity argument NotATypeError's own doc comment makes
// above for the engine's error identity.
type RootNotTopLevelError = gitsrc.RootNotTopLevelError

// UnknownRevisionError is an alias for gitsrc.UnknownRevisionError, not a re-declaration, so that
// errors.As(err, &revErr) against *quarry.UnknownRevisionError succeeds for a caller that never
// imports internal/gitsrc — the same transitivity argument RootNotTopLevelError's own doc comment
// makes above.
type UnknownRevisionError = gitsrc.UnknownRevisionError
