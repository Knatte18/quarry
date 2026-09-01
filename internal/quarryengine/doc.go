// Package quarryengine finds every reference to a symbol name or an explicit
// source position, shows a symbol's definition, searches workspace symbols
// by name (`quarry refs|definition|symbol <symbol|file:line:col>`), shows
// every caller of a symbol paired with its enclosing declaration (`quarry
// impact <symbol|file:line:col>`), and extracts a file's or a directory's
// table of contents (`quarry toc file|dir <path>`), in a target project,
// across whichever of five
// languages (Go, Python, C#, TypeScript, Rust) the project is written in.
// The first three questions are answered over one uniform LSP ("Language
// Server Protocol") path that works for every supported language,
// generalizing the Go-only, in-process go/packages/go/types approach the
// scout spike (docs/research/scout-spike.md) recommended for Go alone, trading the
// spike's sub-millisecond in-process query cost for one LSP round trip per
// query — a deliberate scope trade the scout multilang research
// (docs/research/scout-multilang.md) records in full. V1 (this tree's current
// shape) goes further for Go specifically: it wires a full
// `daemon.EnsureServer` daemon lifecycle — toolchain-managed install,
// spawn-or-reuse, health probing — so Go's language server is warm across
// calls rather than cold-spawned every time. The other four languages
// (Python, C#, TypeScript, Rust) keep the original cold-spawn-per-call
// design entirely unchanged: each call still launches its server fresh,
// initializes it, and tears it down at the end of that one call. The toc
// verbs answer "what is in this file" and "what is in this directory" over
// a second, entirely separate backend: no language server, no daemon —
// package toc parses source directly with tree-sitter (package
// treesitter) and walks the resulting parse tree. This comment covers
// only the resulting design, not the alternatives considered.
//
// # The engine/CLI split
//
// The whole internal/quarryengine tree — this package and its lsp,
// registry, daemon, query, treesitter, toc, and impact subpackages — plus the
// quarry/ facade that re-exports them, is the engine half of an engine/CLI
// seam: it returns typed Go results and typed errors and never imports
// internal/output, cobra, or internal/cli — no io.Writer, no exit codes,
// no output envelope. internal/cli is the sole consumer that maps engine
// results/errors onto the internal/output JSON envelope (output.Ok/
// output.Err). Beyond that negative rule there is no import allowlist: the
// engine draws on the shared infrastructure layer as freely as any other
// engine module, which keeps it cycle-free and importable by any future
// consumer without charging rent on each new dependency. This tree's own
// seam enforcement test (internal/quarryengine/seam_enforcement_test.go)
// enforces the rule across both trees: it walks internal/quarryengine/
// recursively — this package, lsp, registry, daemon, daemon/daemontest,
// query, treesitter, toc, and impact — plus quarry/, and fails if any non-test
// file in either tree imports the CLI package, cobra, or the
// output-envelope package. internal/cli is the sole place engine results
// become JSON.
//
// # The package layout
//
// The engine is an eight-package DAG under internal/quarryengine, plus one
// test-support-only package that sits outside the production DAG:
//
//   - quarryengine (this package; errors.go, position.go, log.go, doc.go)
//     is the shared leaf every other package in the DAG depends on: the
//     typed error vocabulary (see "The typed error vocabulary" below), the
//     caller-facing Position type, and Logger, the shared package-level
//     slog handler every other package logs through. It imports no
//     subpackage of the DAG.
//   - lsp (lspclient.go, wire.go) is the generalized LSP client. See
//     lsp/lspclient.go's own package doc comment for the wire protocol it
//     speaks and the position-conversion rules. It imports only the root.
//   - registry (registry.go, load.go, detect.go) is the language-server
//     registry. See registry/registry.go's own package doc comment for the
//     built-in set, the servers.yaml overlay, and the detection
//     precedence. It imports only the root.
//   - treesitter (treesitter.go) is the toc verbs' parsing backend: it
//     resolves a canonical language name to its compiled tree-sitter
//     grammar and wraps parser/tree construction and release behind one
//     seam, WithTree. See treesitter/treesitter.go's own package doc
//     comment. It imports only the root.
//   - daemon (ensureserver.go, toolchain.go, daemonstate.go, probe.go,
//     doc.go) is the EnsureServer seam, the Go toolchain manager, and the
//     supervised daemon's runtime state. See daemon/doc.go's own package
//     doc comment. It imports the root, registry, and lsp.
//   - toc (strategy.go, toc.go, golang.go, python.go, csharp.go, and the
//     other per-language and shared-helper files) is the toc orchestration
//     layer: the per-language extraction strategies and the TOCFile/TOCDir
//     entry points internal/cli calls for `quarry toc file|dir`. See
//     toc/doc.go's own package doc comment. It imports the root, registry
//     (for language detection), and treesitter (for the parse-and-release
//     seam).
//   - query (definition.go, refs.go, symbol.go, callers.go, verify.go) is
//     the public orchestration layer for the LSP-backed verbs: References,
//     Definition, Symbol, and Callers, the entry points internal/cli calls
//     for `quarry refs|definition|symbol|assert-no-callers`. It imports the
//     root, lsp, registry, and daemon.
//   - impact (impact.go, enclosing.go, types.go) composes query's verified
//     caller set with toc's declaration ranges, answering "who calls this,
//     and what declaration is each call site inside" for `quarry impact`.
//     It sits above query in the DAG. It imports the root, query, and toc.
//   - daemon/daemontest sits outside the production DAG: it exports test
//     seams (WithFakeInstaller, WithTempUserCacheDir, StateFile,
//     KillRecordedDaemon, and the ConnKindNative/ConnKindSupervised/
//     ConnKindLegacy re-export constants) for callers outside package daemon
//     — today only query's tests — and is imported only from _test.go
//     files, never from production code.
//
// # The generalized LSP client
//
// See lsp/lspclient.go's own package doc comment.
//
// # The language-server registry
//
// See registry/registry.go's own package doc comment.
//
// # The EnsureServer seam
//
// See daemon/doc.go's own package doc comment.
//
// # Go toolchain manager
//
// See daemon/doc.go's own package doc comment.
//
// # Daemon state and concurrency
//
// See daemon/doc.go's own package doc comment.
//
// # The typed error vocabulary
//
// Every engine failure mode is a distinct sentinel or data-carrying error
// type (errors.go), each satisfying errors.Is against a package-level
// sentinel regardless of its concrete field values: ErrNoLanguage (no
// registry entry's markers matched under the target directory),
// ErrLanguageUnsupported (a resolved extension maps to no language, or to
// a language the toc strategies do not yet implement — the toc path's only
// sentinel; every other toc failure mode is a CLI-side concern with no
// engine sentinel of its own), ErrServerNotFound (the entry's Command[0]
// binary is absent on $PATH, or — for Go — the toolchain manager could not
// resolve/install the pinned binary; carries InstallHint), ErrSymbolNotFound
// (workspace/symbol resolved the queried name to zero candidates),
// ErrAmbiguousSymbol (resolved to more than one candidate; carries every
// candidate as file:line:col), ErrResolverUnsupported (the launched server
// does not advertise workspaceSymbolProvider), ErrServerTimeout (a phase's
// deadline expired; names the stalled phase and the timeout), and
// ErrServerSpawnTimeout (the supervised strategy's bounded spawn-race retry
// gave up without ever observing a healthy daemon), and ErrBuildTagsUnsupported (a
// non-empty --build-tags set was requested for a language whose registry entry has
// no {{tags}} placeholder to render it into). ErrSymbolNotFound and
// ErrAmbiguousSymbol exist as their own distinct types specifically so
// internal/cli can tell "confirmed absent" apart from "found, but
// ambiguous" without parsing error strings — exit codes and the rest of
// that CLI-level contract are internal/cli's concern, not this package's.
//
// # What this engine deliberately does not do
//
//   - No in-process go/packages arm. The spike's recommended
//     sub-millisecond, zero-false-positive Go-only path is not wired here;
//     this engine always goes through LSP, including for Go (gopls),
//     trading peak Go-only precision/speed for uniform multi-language
//     coverage.
//   - No call hierarchy. textDocument/references, textDocument/definition,
//     the workspace/symbol resolver, and textDocument/implementation are
//     wired, but call hierarchy is not. The spike's call-hierarchy fix
//     (TypesInfo.Uses/Defs-based, not AST-pattern-based) does not generalize
//     to a language-agnostic LSP client. textDocument/implementation is not
//     a general capability exposed to a caller — it is wired for exactly one
//     purpose, widening the declaration match set Callers (query/callers.go)
//     verifies each candidate reference against, so an interface-method
//     query's declaration set includes its concrete implementers.
//   - Symbol does not share query's resolvePosition's (refs.go)
//     ambiguity-collapsing behavior. Unlike References/Definition, Symbol
//     (query/symbol.go) never collapses multiple workspace/symbol
//     candidates into an ErrAmbiguousSymbol failure — returning every match
//     is the whole point of a symbol search, not an error state needing
//     disambiguation. See query/symbol.go's own doc comment for the full
//     rationale.
//   - Exactly one parsing backend ships for toc, never a cgo-and-pure-Go
//     pair selected by build tag. Two independent tree-sitter
//     implementations do not produce identical parse trees, so a
//     build-tag-selected pair would make the same file yield different toc
//     answers depending on how the binary happened to be compiled — a
//     correctness hazard this engine avoids by shipping only the cgo
//     binding.
//   - toc spawns no daemon and caches nothing. Unlike the LSP-backed verbs,
//     which warm a language server across calls, tree-sitter has no
//     project-wide index and no cross-file state for a daemon to keep
//     warm: every toc call parses exactly the file(s) it is asked about,
//     fresh, and nothing is retained between calls.
package quarryengine
