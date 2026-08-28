// Package query implements References, Definition, and Symbol, the engine's public orchestration
// entry points and the sole packages internal/cli calls into.
//
// workspace/symbol is the name → position resolver: given a bare symbol name (no explicit
// position), References/Definition issue workspace/symbol (via resolvePosition, refs.go) and
// require exactly one candidate — zero is quarryengine.ErrSymbolNotFound, more than one is
// quarryengine.ErrAmbiguousSymbol (carrying every candidate as a formatted file:line:col string so
// the caller can disambiguate without a second broader search). A server that does not advertise
// workspaceSymbolProvider in its initialize response fails this path immediately with
// quarryengine.ErrResolverUnsupported rather than attempting the call and getting an empty or
// undefined result. An explicit file:line:col position bypasses this resolver entirely.
//
// A second, narrower resolve mode (--in-file, Query.InFile) exists alongside workspace/symbol for
// the case a caller already knows which file a symbol lives in: resolvePosition issues
// textDocument/documentSymbol for that one file instead and searches its hierarchical result
// exhaustively (collectInFileMatches, descending into every symbol's Children) for exact name
// matches. Unlike workspace/symbol, this resolver does no fuzzy or project-wide matching — it is
// scoped to exactly the one named file — and gates on the server advertising documentSymbolProvider
// rather than workspaceSymbolProvider, failing with quarryengine.ErrResolverUnsupported the same way
// when it does not. The zero/one/many candidate mapping to
// quarryengine.ErrSymbolNotFound/success/quarryengine.ErrAmbiguousSymbol otherwise mirrors
// workspace/symbol's exactly.
//
// refs.go implements References itself, the public orchestration entry point that ties detection
// (registry.DetectLanguage), the language-server registry (the registry package), and the
// generalized LSP client (the lsp package) together: given a target directory and a query (a symbol
// name or an explicit file:line:col position), it launches the right language server, resolves the
// query to a position if needed, and returns the reference list.
// It also defines the shared lookup pipeline (acquireConnection, teardownConnection, lookup) that
// References wraps and that Definition wraps too — both differ only in which single
// LSP call they make once a position is resolved.
// This is the external interface the CLI layer (internal/cli) calls.
package query

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/Knatte18/quarry/internal/quarryengine"
	"github.com/Knatte18/quarry/internal/quarryengine/daemon"
	"github.com/Knatte18/quarry/internal/quarryengine/lsp"
	"github.com/Knatte18/quarry/internal/quarryengine/registry"
)

// Reference reports a symbol reference's file and 1-based line/character position.
type Reference struct {
	File      string
	Line      int
	Character int
}

// InFileQuery resolves a bare symbol name exhaustively within one file via
// textDocument/documentSymbol.
type InFileQuery struct {
	File string
	Name string
}

// Query selects a symbol or position to look up: one of InFile (file-scoped name), Pos (explicit
// position), or Symbol (project-wide name search).
type Query struct {
	InFile *InFileQuery
	Symbol string
	Pos    *quarryengine.Position
}

// Options configures a References call.
type Options struct {
	Registry  registry.Registry
	TargetDir string
	// StateDir is the leaf directory under which the supervised daemon's
	// per-language daemon.json, daemon.lock, and daemon.sock live.
	// It is required and must be a usable absolute path.
	// Populating it is entirely the caller's obligation.
	//
	// Because the supervised daemon is partitioned by StateDir alone, a
	// caller that varies BuildTags while holding StateDir fixed has two tag
	// sets served by one daemon and will get the other tag set's answers.
	// internal/cli discharges this by appending a "tags-<hex>" segment to
	// the resolved leaf; a facade or SDK caller must supply a distinct
	// StateDir per distinct BuildTags value itself — the engine deliberately
	// does not re-key the directory on the caller's behalf.
	StateDir string
	Lang     string
	Query    Query
	Timeout  time.Duration
	// BuildTags is the normalized build-tag set to scope this query's
	// language server to. The engine re-normalizes it defensively on entry
	// (via registry.NormalizeBuildTags, in detectAndRender) rather than
	// trusting the caller's normalization, because Options is public through
	// the facade and an unnormalized value would otherwise fail silently —
	// unlike StateDir, whose misuse fails loudly (the wrong daemon answers,
	// or a stale one is reused) and which stays entirely the caller's
	// obligation.
	//
	// BuildTags carries the same "populating it is entirely the caller's
	// obligation" contract StateDir already does, for the same reason
	// stated on StateDir above: a caller that varies BuildTags without also
	// varying StateDir gets served by whichever daemon StateDir already
	// points at, tag set mismatch and all.
	BuildTags []string
}

// References resolves a query and returns every reference to it, sorted by file:line:character.
func References(ctx context.Context, opts Options) ([]Reference, error) {
	return lookup(ctx, opts, func(ctx context.Context, client *lsp.Client, fileURI string, pos lsp.Position) ([]lsp.Location, error) {
		return client.References(ctx, fileURI, pos)
	})
}

// detectAndRender detects opts.TargetDir's language and renders its
// initializationOptions map for opts.BuildTags, defensively re-normalizing
// the tag set (see Options.BuildTags's doc comment for why) rather than
// trusting the caller already did. Both a detection failure and
// registry.RenderInitializationOptions's quarryengine.ErrBuildTagsUnsupported
// come back through the returned error unchanged, so a caller of this
// function need not distinguish the two failure modes to propagate them
// correctly.
func detectAndRender(opts Options) (string, registry.Entry, map[string]any, error) {
	lang, entry, err := registry.DetectLanguage(opts.TargetDir, opts.Registry, opts.Lang)
	if err != nil {
		return "", registry.Entry{}, nil, err
	}

	tags := registry.NormalizeBuildTags(opts.BuildTags...)
	initOptions, err := registry.RenderInitializationOptions(lang, entry, tags)
	if err != nil {
		return "", registry.Entry{}, nil, err
	}

	return lang, entry, initOptions, nil
}

// acquireConnection obtains a ready-to-use LSP client and its teardown kind. initOptions is the
// already-rendered initializationOptions map (detectAndRender's result, nil for an untagged
// query) — threaded to daemon.EnsureServer on the native-daemon path and to the legacy path's own
// client.Initialize call below, unchanged either way.
func acquireConnection(ctx context.Context, lang string, entry registry.Entry, opts Options, initOptions map[string]any) (*lsp.Client, daemon.ConnKind, error) {
	if entry.HasNativeDaemon {
		return daemon.EnsureServer(ctx, lang, entry, opts.TargetDir, opts.StateDir, opts.Timeout, initOptions)
	}

	client, err := lsp.NewClient(entry.Command)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, daemon.ConnKindLegacy, &quarryengine.ErrServerNotFound{Language: lang, InstallHint: entry.InstallHint}
		}
		return nil, daemon.ConnKindLegacy, fmt.Errorf("quarry: start language server for %q: %w", lang, err)
	}
	client.Lang = lang

	rootURI, err := daemon.RootURIFor(opts.TargetDir)
	if err != nil {
		client.Kill()
		return nil, daemon.ConnKindLegacy, err
	}

	initCtx, initCancel := context.WithTimeout(ctx, opts.Timeout)
	defer initCancel()
	if err := client.Initialize(initCtx, rootURI, initOptions); err != nil {
		// Mirror the existing timedOut-branching teardown logic exactly,
		// just localized to this one failure instead of spanning the whole
		// call: a timed-out server could re-block on the graceful shutdown
		// handshake close() sends, so a stalled initialize is hard-killed
		// instead.
		if errors.Is(err, quarryengine.ErrServerTimeoutSentinel) {
			client.Kill()
		} else {
			client.Close()
		}
		return nil, daemon.ConnKindLegacy, err
	}

	return client, daemon.ConnKindLegacy, nil
}

// teardownConnection tears down client per the rule its daemon.ConnKind demands —
// the two EnsureServer strategies wrap fundamentally different process
// lifetimes and must not be torn down the same way (see the plan's daemon.ConnKind
// teardown Shared Decision).
func teardownConnection(client *lsp.Client, kind daemon.ConnKind, timedOut bool) {
	switch kind {
	case daemon.ConnKindSupervised:
		// A supervised connection is a dial into a daemon quarry spawned to
		// outlive this call. Never run the LSP shutdown handshake or kill
		// it — the daemon is meant to keep serving other callers, and this
		// process's exit reclaims the dialed socket's fd on its own.
		return
	default:
		// daemon.ConnKindNative and daemon.ConnKindLegacy share identical teardown: both
		// wrap a connection this call owns outright (native's disposable
		// proxy subprocess, or the legacy path's own directly-spawned
		// server), so hard-killing on a timeout and gracefully closing
		// otherwise is safe for both.
		if timedOut {
			client.Kill()
		} else {
			client.Close()
		}
	}
}

// runOnConnection performs the connection-acquisition portion of the lookup pipeline every
// multi-call entry point shares: detectAndRender, acquireConnection, a deferred teardownConnection
// closing over a timedOut local, and resolvePosition. Once a position is resolved, it calls fn with
// the resolved client, the file URI, the position, and a pointer to that same timedOut local, so fn
// can issue as many further LSP calls on this one connection as its pipeline needs, sequentially,
// before the deferred teardown runs.
//
// timedOut is captured by reference — fn receives a pointer to it — because a phase fn runs after
// this function's defer is registered can still set it to true; this mirrors the invariant lookup
// (below) relied on directly before this call was factored out. teardownConnection's
// ConnKindSupervised branch must keep returning with no shutdown handshake and no kill regardless of
// what fn does to *timedOut, since a supervised daemon connection outlives this call by design.
func runOnConnection(ctx context.Context, opts Options, fn func(ctx context.Context, client *lsp.Client, fileURI string, pos lsp.Position, timedOut *bool) error) error {
	lang, entry, initOptions, err := detectAndRender(opts)
	if err != nil {
		return err
	}

	client, kind, err := acquireConnection(ctx, lang, entry, opts, initOptions)
	if err != nil {
		// No deferred teardown needed here: acquireConnection already tears
		// down any partial connection itself on its own error path.
		return err
	}

	// timedOut is captured by reference via the pointer fn receives, since
	// resolvePosition below or any phase fn itself runs may still set it to
	// true after this defer is registered.
	timedOut := false
	defer func() {
		teardownConnection(client, kind, timedOut)
	}()

	fileURI, lspPos, err := resolvePosition(ctx, client, opts, lang, entry)
	if err != nil {
		if errors.Is(err, quarryengine.ErrServerTimeoutSentinel) {
			timedOut = true
		}
		return err
	}

	return fn(ctx, client, fileURI, lspPos, &timedOut)
}

// lookup is the thin single-LSP-call wrapper both References and Definition run over
// runOnConnection: once runOnConnection resolves a position, lookup's fn issues exactly one LSP
// call (lspCall, the one step that varies between callers) under a fresh
// context.WithTimeout(ctx, opts.Timeout) deadline, sets *timedOut when the call's error satisfies
// errors.Is(err, quarryengine.ErrServerTimeoutSentinel), and stores the resulting locations for
// conversion with toSortedReferences once runOnConnection returns.
func lookup(ctx context.Context, opts Options, lspCall func(ctx context.Context, client *lsp.Client, fileURI string, pos lsp.Position) ([]lsp.Location, error)) ([]Reference, error) {
	var locations []lsp.Location
	err := runOnConnection(ctx, opts, func(ctx context.Context, client *lsp.Client, fileURI string, pos lsp.Position, timedOut *bool) error {
		callCtx, callCancel := context.WithTimeout(ctx, opts.Timeout)
		defer callCancel()
		locs, err := lspCall(callCtx, client, fileURI, pos)
		if err != nil {
			if errors.Is(err, quarryengine.ErrServerTimeoutSentinel) {
				*timedOut = true
			}
			return err
		}
		locations = locs
		return nil
	})
	if err != nil {
		return nil, err
	}

	return toSortedReferences(locations), nil
}

// resolvePosition returns the file:// URI and LSP wire position
// textDocument/references should query, checking opts.Query's three forms in
// a fixed precedence — InFile first, then Pos, then Symbol — even though
// callers are expected to set only one.
//
// When opts.Query.InFile is set, resolvePosition gates on
// client.SupportsDocumentSymbol() (quarryengine.ErrResolverUnsupported if unadvertised),
// then issues textDocument/documentSymbol for InFile.File and searches the
// result recursively (collectInFileMatches) for every symbol whose Name
// exactly equals InFile.Name: zero matches is quarryengine.ErrSymbolNotFound, more than
// one is quarryengine.ErrAmbiguousSymbol (each candidate formatted as file:line:col from
// its SelectionRange), and exactly one match's SelectionRange.Start is used
// as-is as the LSP position — no fuzzy matching, no column math, mirroring
// the no-round-trip discipline described below for the Symbol path.
//
// When opts.Query.Pos is set, it is converted directly via lsp.ToPosition.
//
// Otherwise resolvePosition resolves opts.Query.Symbol via workspace/symbol:
// zero candidates is quarryengine.ErrSymbolNotFound, more than one is quarryengine.ErrAmbiguousSymbol
// (each candidate formatted as file:line:col), and exactly one candidate's
// own location is used as-is — its Range.Start is already the
// 0-based-line/UTF-16-character LSP position the wire format needs, so no
// round trip through the byte-column Position type happens on this path
// (that round trip would misconvert the offset on any line with a
// multi-byte rune before the symbol, exactly the hazard lsp.ToPosition exists
// to avoid on the Query.Pos path). A server that does not advertise
// workspaceSymbolProvider yields quarryengine.ErrResolverUnsupported rather than
// attempting the call.
func resolvePosition(ctx context.Context, client *lsp.Client, opts Options, lang string, entry registry.Entry) (fileURI string, pos lsp.Position, err error) {
	if opts.Query.InFile != nil {
		if !client.SupportsDocumentSymbol() {
			return "", lsp.Position{}, &quarryengine.ErrResolverUnsupported{Language: lang, Server: entry.Command[0]}
		}

		symCtx, symCancel := context.WithTimeout(ctx, opts.Timeout)
		defer symCancel()
		symbols, err := client.DocumentSymbols(symCtx, "file://"+opts.Query.InFile.File)
		if err != nil {
			return "", lsp.Position{}, err
		}

		matches := collectInFileMatches(symbols, opts.Query.InFile.Name)
		switch len(matches) {
		case 0:
			return "", lsp.Position{}, &quarryengine.ErrSymbolNotFound{Symbol: opts.Query.InFile.Name, TargetDir: opts.TargetDir}
		case 1:
			return "file://" + opts.Query.InFile.File, matches[0].SelectionRange.Start, nil
		default:
			formatted := make([]string, len(matches))
			for i, m := range matches {
				formatted[i] = lsp.FormatLocation(lsp.Location{URI: "file://" + opts.Query.InFile.File, Range: m.SelectionRange})
			}
			return "", lsp.Position{}, &quarryengine.ErrAmbiguousSymbol{Symbol: opts.Query.InFile.Name, Candidates: formatted}
		}
	}

	if opts.Query.Pos != nil {
		lspPos, err := lsp.ToPosition(*opts.Query.Pos)
		if err != nil {
			return "", lsp.Position{}, fmt.Errorf("quarry: convert position %+v: %w", *opts.Query.Pos, err)
		}
		return "file://" + opts.Query.Pos.File, lspPos, nil
	}

	if !client.SupportsWorkspaceSymbol() {
		return "", lsp.Position{}, &quarryengine.ErrResolverUnsupported{Language: lang, Server: entry.Command[0]}
	}

	symbolCtx, symbolCancel := context.WithTimeout(ctx, opts.Timeout)
	defer symbolCancel()
	candidates, err := client.WorkspaceSymbol(symbolCtx, opts.Query.Symbol)
	if err != nil {
		return "", lsp.Position{}, err
	}

	switch len(candidates) {
	case 0:
		return "", lsp.Position{}, &quarryengine.ErrSymbolNotFound{Symbol: opts.Query.Symbol, TargetDir: opts.TargetDir}
	case 1:
		loc := candidates[0].Location
		return loc.URI, loc.Range.Start, nil
	default:
		formatted := make([]string, len(candidates))
		for i, c := range candidates {
			formatted[i] = lsp.FormatLocation(c.Location)
		}
		return "", lsp.Position{}, &quarryengine.ErrAmbiguousSymbol{Symbol: opts.Query.Symbol, Candidates: formatted}
	}
}

// collectInFileMatches recursively searches syms — a
// textDocument/documentSymbol result, descending into every node's Children
// — for symbols whose Name exactly equals name, and returns every match
// found in encounter order. It is factored out of resolvePosition's InFile
// branch as a pure, transport-free helper specifically so the recursive
// exact-name collection is unit-testable without a fake LSP server. No fuzzy
// matching happens here: a name that differs by case, substring, or any
// other transformation is never collected.
func collectInFileMatches(syms []lsp.DocumentSymbol, name string) []lsp.DocumentSymbol {
	var matches []lsp.DocumentSymbol
	for _, sym := range syms {
		if bareInFileName(sym.Name) == name {
			matches = append(matches, sym)
		}
		matches = append(matches, collectInFileMatches(sym.Children, name)...)
	}
	return matches
}

// bareInFileName strips gopls's "(Receiver).Method" qualifier from a method
// DocumentSymbol's Name. gopls does not nest a method as a bare-named Child
// under its receiver type's DocumentSymbol the way it nests, e.g., an
// interface's own methods or a struct's fields — every concrete method comes
// back as its own entry (top-level, or nested only as deep as its enclosing
// declaration requires) named "(Receiver).Method", not "Method". A name
// without a leading "(...)"  qualifier is returned unchanged, so plain
// functions and any symbol kind that genuinely does nest under bare names
// are unaffected.
func bareInFileName(name string) string {
	if !strings.HasPrefix(name, "(") {
		return name
	}
	if i := strings.Index(name, ")."); i != -1 {
		return name[i+2:]
	}
	return name
}

// toSortedReferences maps raw LSP locations to the public Reference type
// (file:// URIs trimmed back to plain paths, 0-based positions promoted to
// 1-based for display) and sorts them by file, then line, then character —
// a stable, portable display order independent of whatever order the
// server returned results in.
func toSortedReferences(locations []lsp.Location) []Reference {
	refs := make([]Reference, len(locations))
	for i, loc := range locations {
		refs[i] = Reference{
			File:      trimFileURI(loc.URI),
			Line:      loc.Range.Start.Line + 1,
			Character: loc.Range.Start.Character + 1,
		}
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].File != refs[j].File {
			return refs[i].File < refs[j].File
		}
		if refs[i].Line != refs[j].Line {
			return refs[i].Line < refs[j].Line
		}
		return refs[i].Character < refs[j].Character
	})
	return refs
}

// trimFileURI strips the "file://" scheme from an LSP document URI, the
// same conversion lsp.FormatLocation (wire.go) applies.
func trimFileURI(uri string) string {
	return strings.TrimPrefix(uri, "file://")
}
