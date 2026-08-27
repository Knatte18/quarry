// Package quarryengine finds every reference to a symbol name or an explicit
// source position, shows a symbol's definition, and searches workspace
// symbols by name (`quarry refs|definition|symbol <symbol|file:line:col>`)
// in a target project, across whichever of five languages (Go, Python, C#,
// TypeScript, Rust) the project is written in. It generalizes the Go-only,
// in-process go/packages/go/types approach the scout spike
// (docs/scout-spike.md) recommended for Go alone into a uniform LSP
// ("Language Server Protocol") path that works for every supported
// language, including Go, trading the spike's sub-millisecond in-process
// query cost for one LSP round trip per query — a deliberate scope trade
// the scout multilang research (docs/scout-multilang.md) records in full.
// V1 (this tree's current shape) goes further for Go specifically: it wires
// a full `daemon.EnsureServer` daemon lifecycle — toolchain-managed
// install, spawn-or-reuse, health probing — so Go's language server is warm
// across calls rather than cold-spawned every time. The other four
// languages (Python, C#, TypeScript, Rust) keep the original
// cold-spawn-per-call design entirely unchanged: each call still launches
// its server fresh, initializes it, and tears it down at the end of that
// one call. This comment covers only the resulting design, not the
// alternatives considered.
//
// # The engine/CLI split
//
// The whole internal/quarryengine tree — this package and its lsp,
// registry, daemon, and query subpackages — plus the quarry/ facade that
// re-exports them, is the engine half of an engine/CLI seam: it returns
// typed Go results and typed errors and never imports internal/output,
// cobra, or internal/cli — no io.Writer, no exit codes, no output envelope.
// internal/cli is the sole consumer that maps engine results/errors onto
// the internal/output JSON envelope (output.Ok/output.Err). Beyond that
// negative rule there is no import allowlist: the engine draws on the
// shared infrastructure layer as freely as any other engine module, which
// keeps it cycle-free and importable by any future consumer without
// charging rent on each new dependency. This tree's own seam enforcement
// test (internal/quarryengine/seam_enforcement_test.go) enforces the rule
// across both trees: it walks internal/quarryengine/ recursively — this
// package, lsp, registry, daemon, daemon/daemontest, and query — plus
// quarry/, and fails if any non-test file in either tree imports the CLI
// package, cobra, or the output-envelope package. internal/cli is the sole
// place engine results become JSON.
//
// # The package layout
//
// The engine is a five-package DAG under internal/quarryengine, plus one
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
//   - daemon (ensureserver.go, toolchain.go, daemonstate.go, probe.go,
//     doc.go) is the EnsureServer seam, the Go toolchain manager, and the
//     supervised daemon's runtime state. See daemon/doc.go's own package
//     doc comment. It imports the root, registry, and lsp.
//   - query (definition.go, refs.go, symbol.go) is the public orchestration
//     layer: References, Definition, and Symbol, the entry points
//     internal/cli calls. It imports all four packages above.
//   - daemon/daemontest sits outside the production DAG: it exports test
//     seams (WithFakeInstaller, WithTempUserCacheDir, StateFile,
//     KillRecordedDaemon) for callers outside package daemon — today only
//     query's tests — and is imported only from _test.go files, never from
//     production code.
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
// ErrServerNotFound (the entry's Command[0] binary is absent on $PATH, or —
// for Go — the toolchain manager could not resolve/install the pinned
// binary; carries InstallHint), ErrSymbolNotFound (workspace/symbol
// resolved the queried name to zero candidates), ErrAmbiguousSymbol
// (resolved to more than one candidate; carries every candidate as
// file:line:col), ErrResolverUnsupported (the launched server does not
// advertise workspaceSymbolProvider), ErrServerTimeout (a phase's deadline
// expired; names the stalled phase and the timeout), and
// ErrServerSpawnTimeout (the supervised strategy's bounded spawn-race retry
// gave up without ever observing a healthy daemon). ErrSymbolNotFound and
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
//   - No call hierarchy, no implementation. Only textDocument/references,
//     textDocument/definition, and the workspace/symbol resolver are wired.
//     The spike's call-hierarchy fix (TypesInfo.Uses/Defs-based, not
//     AST-pattern-based) does not generalize to a language-agnostic LSP
//     client, and implementation was never in this engine's rubric.
//   - Symbol does not share query's resolvePosition's (refs.go)
//     ambiguity-collapsing behavior. Unlike References/Definition, Symbol
//     (query/symbol.go) never collapses multiple workspace/symbol
//     candidates into an ErrAmbiguousSymbol failure — returning every match
//     is the whole point of a symbol search, not an error state needing
//     disambiguation. See query/symbol.go's own doc comment for the full
//     rationale.
package quarryengine
