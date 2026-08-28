// Package quarry is a stable, behaviour-free re-export of internal/quarryengine.
// It exists so this module's import path (github.com/Knatte18/quarry/quarry) stays unchanged
// while the engine itself lives at internal/quarryengine, split into the seven-package DAG
// documented there: the root leaf package, lsp, registry, treesitter, daemon, toc, and query.
// This file adds nothing of its own — every declaration below is either a type alias, a
// re-exported sentinel var bound to the identical error value, or a one-line delegating
// function.
// That is the guarantee this file exists to keep, stated as a property rather than as a count
// that would go stale on every addition: whatever the engine exports through this facade,
// grep for a bare struct field, an inline computation, or a multi-line function body anywhere
// below and you will find none — facade_test.go's alias and delegation checks enforce the same
// property mechanically, for every identifier this file re-exports, not just a snapshot count of
// them.
// For the engine's own design — the package DAG, the EnsureServer daemon lifecycle, the
// references/definition/symbol resolution pipeline — see internal/quarryengine's own package
// doc comment, not this file.

package quarry

import (
	"context"

	"github.com/Knatte18/quarry/internal/quarryengine"
	"github.com/Knatte18/quarry/internal/quarryengine/daemon"
	"github.com/Knatte18/quarry/internal/quarryengine/query"
	"github.com/Knatte18/quarry/internal/quarryengine/registry"
	"github.com/Knatte18/quarry/internal/quarryengine/toc"
)

// Entry is registry.Entry, re-exported unchanged.
type Entry = registry.Entry

// Registry is registry.Registry, re-exported unchanged.
type Registry = registry.Registry

// Position is quarryengine.Position, re-exported unchanged.
type Position = quarryengine.Position

// Query is query.Query, re-exported unchanged.
type Query = query.Query

// InFileQuery is query.InFileQuery, re-exported unchanged.
type InFileQuery = query.InFileQuery

// Options is query.Options, re-exported unchanged.
type Options = query.Options

// Reference is query.Reference, re-exported unchanged.
type Reference = query.Reference

// SymbolMatch is query.SymbolMatch, re-exported unchanged.
type SymbolMatch = query.SymbolMatch

// ErrServerNotFound is quarryengine.ErrServerNotFound, re-exported unchanged.
// It is a type alias, not a defined type, so errors.As and type assertions against it still
// match a value the engine actually constructed.
type ErrServerNotFound = quarryengine.ErrServerNotFound

// ErrSymbolNotFound is quarryengine.ErrSymbolNotFound, re-exported unchanged.
type ErrSymbolNotFound = quarryengine.ErrSymbolNotFound

// ErrAmbiguousSymbol is quarryengine.ErrAmbiguousSymbol, re-exported unchanged.
type ErrAmbiguousSymbol = quarryengine.ErrAmbiguousSymbol

// ErrResolverUnsupported is quarryengine.ErrResolverUnsupported, re-exported unchanged.
type ErrResolverUnsupported = quarryengine.ErrResolverUnsupported

// ErrServerTimeout is quarryengine.ErrServerTimeout, re-exported unchanged.
type ErrServerTimeout = quarryengine.ErrServerTimeout

// ErrServerSpawnTimeout is quarryengine.ErrServerSpawnTimeout, re-exported unchanged.
type ErrServerSpawnTimeout = quarryengine.ErrServerSpawnTimeout

// ErrBuildTagsUnsupported is quarryengine.ErrBuildTagsUnsupported, re-exported unchanged.
type ErrBuildTagsUnsupported = quarryengine.ErrBuildTagsUnsupported

// ErrNoLanguage is quarryengine.ErrNoLanguage, the identical sentinel value re-exported for
// errors.Is comparisons against this package's import path.
var ErrNoLanguage = quarryengine.ErrNoLanguage

// ErrServerNotFoundSentinel is quarryengine.ErrServerNotFoundSentinel, re-exported unchanged.
var ErrServerNotFoundSentinel = quarryengine.ErrServerNotFoundSentinel

// ErrSymbolNotFoundSentinel is quarryengine.ErrSymbolNotFoundSentinel, re-exported unchanged.
var ErrSymbolNotFoundSentinel = quarryengine.ErrSymbolNotFoundSentinel

// ErrAmbiguousSymbolSentinel is quarryengine.ErrAmbiguousSymbolSentinel, re-exported unchanged.
var ErrAmbiguousSymbolSentinel = quarryengine.ErrAmbiguousSymbolSentinel

// ErrResolverUnsupportedSentinel is quarryengine.ErrResolverUnsupportedSentinel, re-exported
// unchanged.
var ErrResolverUnsupportedSentinel = quarryengine.ErrResolverUnsupportedSentinel

// ErrServerTimeoutSentinel is quarryengine.ErrServerTimeoutSentinel, re-exported unchanged.
var ErrServerTimeoutSentinel = quarryengine.ErrServerTimeoutSentinel

// ErrServerSpawnTimeoutSentinel is quarryengine.ErrServerSpawnTimeoutSentinel, re-exported
// unchanged.
var ErrServerSpawnTimeoutSentinel = quarryengine.ErrServerSpawnTimeoutSentinel

// ErrBuildTagsUnsupportedSentinel is quarryengine.ErrBuildTagsUnsupportedSentinel, re-exported
// unchanged.
var ErrBuildTagsUnsupportedSentinel = quarryengine.ErrBuildTagsUnsupportedSentinel

// BuiltinRegistry delegates to registry.BuiltinRegistry.
func BuiltinRegistry() Registry {
	return registry.BuiltinRegistry()
}

// LoadRegistry delegates to registry.LoadRegistry.
func LoadRegistry(path string) (Registry, error) {
	return registry.LoadRegistry(path)
}

// DetectLanguage delegates to registry.DetectLanguage.
func DetectLanguage(targetDir string, reg Registry, langOverride string) (string, Entry, error) {
	return registry.DetectLanguage(targetDir, reg, langOverride)
}

// DaemonStateFile delegates to daemon.DaemonStateFile.
func DaemonStateFile(stateDir string, lang string) string {
	return daemon.DaemonStateFile(stateDir, lang)
}

// DaemonLock delegates to daemon.DaemonLock.
func DaemonLock(stateDir string, lang string) string {
	return daemon.DaemonLock(stateDir, lang)
}

// References delegates to query.References.
func References(ctx context.Context, opts Options) ([]Reference, error) {
	return query.References(ctx, opts)
}

// Definition delegates to query.Definition.
func Definition(ctx context.Context, opts Options) ([]Reference, error) {
	return query.Definition(ctx, opts)
}

// Symbol delegates to query.Symbol.
func Symbol(ctx context.Context, opts Options) ([]SymbolMatch, error) {
	return query.Symbol(ctx, opts)
}

// Callers delegates to query.Callers.
func Callers(ctx context.Context, opts Options) ([]Reference, []Reference, error) {
	return query.Callers(ctx, opts)
}

// NormalizeBuildTags delegates to registry.NormalizeBuildTags.
func NormalizeBuildTags(tags ...string) []string {
	return registry.NormalizeBuildTags(tags...)
}

// TOCSymbol is toc.Symbol, re-exported unchanged.
type TOCSymbol = toc.Symbol

// TOCKind is toc.Kind, re-exported unchanged.
type TOCKind = toc.Kind

// TOCFileResult is toc.FileTOC, re-exported unchanged.
type TOCFileResult = toc.FileTOC

// TOCDirEntry is toc.DirEntry, re-exported unchanged.
type TOCDirEntry = toc.DirEntry

// TOCDirResult is toc.DirTOC, re-exported unchanged.
type TOCDirResult = toc.DirTOC

// TOCOptions is toc.Options, re-exported unchanged.
type TOCOptions = toc.Options

// The three TOCKind values a toc symbol ever carries, re-exported unchanged.
const (
	// TOCKindFunction is toc.KindFunction, re-exported unchanged.
	TOCKindFunction = toc.KindFunction
	// TOCKindMethod is toc.KindMethod, re-exported unchanged.
	TOCKindMethod = toc.KindMethod
	// TOCKindType is toc.KindType, re-exported unchanged.
	TOCKindType = toc.KindType
)

// TOCAllSentences is toc.AllSentences, the TOCOptions.DocSentences sentinel meaning "keep the
// whole docstring, unsplit", re-exported unchanged.
const TOCAllSentences = toc.AllSentences

// ErrLanguageUnsupported is quarryengine.ErrLanguageUnsupported, the identical sentinel value
// re-exported for errors.Is comparisons against this package's import path.
var ErrLanguageUnsupported = quarryengine.ErrLanguageUnsupported

// TOCFile delegates to toc.TOCFile.
func TOCFile(path string, lang string, opts TOCOptions) (TOCFileResult, error) {
	return toc.TOCFile(path, lang, opts)
}

// TOCDir delegates to toc.TOCDir.
func TOCDir(dir string, lang string) (TOCDirResult, error) {
	return toc.TOCDir(dir, lang)
}

// TOCLanguages delegates to registry.ExtensionLanguages, returning the five languages the toc
// survey designed for, regardless of whether each has a registered Strategy yet. internal/cli
// validates --lang against this set, not TOCImplemented's: a designed-but-unimplemented language
// stays a legal request that toc dir can list files under, while toc file surfaces
// ErrLanguageUnsupported for that same request. internal/cli imports nothing under
// internal/quarryengine directly — every engine identifier it needs reaches it through this
// facade file — and this function is why toc's --lang validation does not become the first
// exception.
func TOCLanguages() []string {
	return registry.ExtensionLanguages()
}

// TOCImplemented delegates to toc.Implemented, returning the subset of TOCLanguages that
// currently has a registered Strategy. The unsupported-language error message internal/cli builds
// is worded from this set, not TOCLanguages': that message answers "what can quarry actually
// read", and naming the full designed set there would claim a language like rust is available in
// the very error saying it is not. Like TOCLanguages, this exists so internal/cli never has to
// import internal/quarryengine/toc directly.
func TOCImplemented() []string {
	return toc.Implemented()
}
