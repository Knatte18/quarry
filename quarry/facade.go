// Package quarry is a stable, behaviour-free re-export of internal/quarryengine.
// It exists so this module's import path (github.com/Knatte18/quarry/quarry) stays unchanged
// while the engine itself lives at internal/quarryengine, split into the five-package DAG
// documented there: the root leaf package, lsp, registry, daemon, and query.
// This file adds nothing of its own — every declaration below is either a type alias, a
// re-exported sentinel var bound to the identical error value, or a one-line delegating
// function.
// It re-exports exactly the 29 identifiers this package exported before the engine-repackage
// move: no more, no less.
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
