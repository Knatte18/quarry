# Batch: engine-repackage

```yaml
task: Thin quarry/ facade over internal/quarryengine
batch: engine-repackage
number: 2
cards: 8
verify: go test ./... && go test -tags lsp -run "^$" ./...
depends-on: [1]
```

## Rename mechanic

For each `Moves:` pair the implementer MUST:

1. Run `git mv <old> <new>` FIRST, before making any other change to the moved file.
2. Make ONLY surgical edits — touch only the lines that must change after the move (package or module declaration, imports, identifier retargeting, seam splits).
3. Use a full-file `Creates:` entry only for genuinely new files that have no predecessor.
4. Never write the relocated file from scratch and delete the original — that breaks git rename history and inflates review diffs.

## Batch Scope

This batch is the reorganization itself: all 14 production files and 20 test files leave the flat `quarry/` package for the five-package DAG under `internal/quarryengine/`, the identifiers that now cross a package boundary become exported, a test-support package `daemontest` is created to carry the one test seam that cannot survive the split, and `quarry/` is rebuilt as a facade of aliases and delegating functions at its unchanged import path. It is one batch because no smaller unit compiles: removing `errors.go` from `package quarry` breaks every remaining file in it, so the move is atomic by construction. Individual cards may leave the tree non-compiling between commits; only the batch boundary is a green gate.

The external interface batches 3 and 4 consume is the finished package layout: batch 3 writes guards that walk it, batch 4 redistributes prose into it.

Two batch-local notes. First, cards run in the listed order and the order matters — card 3 strips the LSP wire types out of `position.go` and card 4 re-homes them in `lsp/wire.go`; running card 4 before card 3 leaves them duplicated. Second, `quarry/lspclient_guard_test.go` and `quarry/seam_enforcement_test.go` are not deferred to batch 3: both would fail the moment their targets move, so cards 4 and 10 retarget them as part of the move itself.

## Cards

### Card 3: Create the `quarryengine` root package

- **Context:**
  - `quarry/ensureserver.go`
  - `quarry/lspclient.go`
  - `quarry/refs.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/log.go`
- **Deletes:** none
- **Moves:**
  - `quarry/errors.go` -> `internal/quarryengine/errors.go`
  - `quarry/position.go` -> `internal/quarryengine/position.go`
- **Requirements:** Both moved files declare `package quarryengine`. `errors.go` keeps every declaration unchanged except its line-22 doc comment, whose reference `quarry.ErrServerNotFoundSentinel` (set by batch 1 card 2) becomes `quarryengine.ErrServerNotFoundSentinel`. `position.go` keeps ONLY the `Position` struct and its doc comment; delete `lspPosition`, `lspRange`, `lspLocation`, `toLSPPosition`, `utf16Length`, and `formatLocation` from it — card 4 re-homes them in the `lsp` package. After the strip, `position.go` needs no imports at all; remove the now-unused `fmt`, `os`, `strings`, and `unicode/utf16` import block. Create `internal/quarryengine/log.go` declaring `package quarryengine` and one exported package-level var `Logger`, moved verbatim from `quarry/ensureserver.go:32`'s `defaultLogHandler`: `var Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))`. Carry over `ensureserver.go`'s existing doc comment for that var (lines 26–31), adjusting the name. Card 6 deletes the original declaration and retargets its two call sites; card 4 retargets `lspclient.go`'s five.
- **Commit:** `refactor(quarry): move errors, Position and the log handler to internal/quarryengine`

### Card 4: Create the `lsp` package

- **Context:**
  - `internal/quarryengine/errors.go`
  - `internal/quarryengine/log.go`
  - `internal/quarryengine/position.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/lsp/wire.go`
- **Deletes:** none
- **Moves:**
  - `quarry/lspclient.go` -> `internal/quarryengine/lsp/lspclient.go`
  - `quarry/lspclient_test.go` -> `internal/quarryengine/lsp/lspclient_test.go`
  - `quarry/lspclient_guard_test.go` -> `internal/quarryengine/lsp/lspclient_guard_test.go`
  - `quarry/position_test.go` -> `internal/quarryengine/lsp/position_test.go`
- **Requirements:** All four moved files declare `package lsp`. Create `internal/quarryengine/lsp/wire.go` holding the six declarations card 3 stripped from `position.go`, renamed on the way in: `lspPosition` -> `Position`, `lspRange` -> `Range`, `lspLocation` -> `Location`, `toLSPPosition` -> `ToPosition`, `formatLocation` -> `FormatLocation`, and `utf16Length` unchanged and still unexported. `ToPosition` takes a `quarryengine.Position` and returns an `lsp.Position`, so `wire.go` imports `github.com/Knatte18/quarry/internal/quarryengine` alongside `fmt`, `os`, `strings`, `unicode/utf16`. In `lspclient.go`, retarget every use of the old names to the new ones, retarget the five `defaultLogHandler` call sites (lines 574, 577, 582, 605, 608) to `quarryengine.Logger`, retarget `ErrServerTimeout` to `quarryengine.ErrServerTimeout`, and export the identifiers `query` will need: type `lspClient` -> `Client`; constructors `newLSPClient` -> `NewClient`, `newLSPClientFromRW` -> `NewClientFromRW`, `newLSPClientDial` -> `NewClientDial`; types `lspDocumentSymbol` -> `DocumentSymbol`, `symbolInformation` -> `SymbolInformation`; methods `call` -> `Call`, `notify` -> `Notify`, `initialize` -> `Initialize`, `references` -> `References`, `definition` -> `Definition`, `workspaceSymbol` -> `WorkspaceSymbol`, `documentSymbol` -> `DocumentSymbol`, `close` -> `Close`, `kill` -> `Kill`, `supportsWorkspaceSymbol` -> `SupportsWorkspaceSymbol`, `supportsDocumentSymbol` -> `SupportsDocumentSymbol`. The method `documentSymbol` and the type `lspDocumentSymbol` both map onto the name `DocumentSymbol`, which is NOT a Go conflict — method names live in the receiver's method set, not the package identifier scope, so `func (c *Client) DocumentSymbol(...) ([]DocumentSymbol, error)` compiles fine. Name the method `DocumentSymbols` anyway, purely as a readability choice because it returns a slice, and say so in its doc comment; do not describe it as avoiding a compile error. Leave `lspError`, `lspMessage`, `lspReadResult`, `capabilities`, `capabilityFlag`, `readLoop`, `readMessage`, `writeMessage`, and `parseDefinitionResult` unexported. Update `lspclient_test.go` and `position_test.go` to the new names throughout. One rename in `position_test.go` is not a substitution but a qualification, and a naive find-and-replace gets it wrong: its two `Position{File: path, Line: 3, Character: N}` literals at lines 31 and 51 mean the CALLER-facing position, which card 3 left in the root package, while the file's other `Position` references now mean the wire type. Those two literals become `quarryengine.Position{...}` and the file gains an `internal/quarryengine` import; leaving them bare yields `unknown field File in struct literal`. In `lspclient_guard_test.go`, retarget the hardcoded `pkgDir`-relative target to this package's own `lspclient.go` and widen its allowed set from "stdlib only" to "stdlib plus the single import path `github.com/Knatte18/quarry/internal/quarryengine`" — every other non-stdlib import, first-party or third-party, still fails. Update the file's own doc comment and the failure message to state the widened rule and why it is one hardcoded path rather than a per-file allowed-set table.
- **Commit:** `refactor(quarry): move the LSP client and wire types to internal/quarryengine/lsp`

### Card 5: Create the `registry` package

- **Context:**
  - `internal/quarryengine/errors.go`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:**
  - `quarry/registry.go` -> `internal/quarryengine/registry/registry.go`
  - `quarry/load.go` -> `internal/quarryengine/registry/load.go`
  - `quarry/detect.go` -> `internal/quarryengine/registry/detect.go`
  - `quarry/registry_test.go` -> `internal/quarryengine/registry/registry_test.go`
  - `quarry/load_test.go` -> `internal/quarryengine/registry/load_test.go`
  - `quarry/detect_test.go` -> `internal/quarryengine/registry/detect_test.go`
- **Requirements:** All six moved files declare `package registry`. `Entry`, `Registry`, `BuiltinRegistry`, `LoadRegistry`, and `DetectLanguage` are already exported and keep their names and signatures. `builtins`, `precedence`, `validateEntry`, `markersMatch`, `markerExists`, and `sortedLanguages` stay unexported — they have no production caller outside this package, and `registry_test.go`/`load_test.go`/`detect_test.go` reach them in-package. Four `//go:build lsp` test files DO call `builtins()` from outside this package once they move: `ensureserver_integration_test.go`, `toolchain_integration_test.go`, `supervised_integration_test.go` (all to `daemon`, card 6) and `refs_integration_test.go` (to `query`, card 8). They retarget to the already-exported `registry.BuiltinRegistry()`, whose body is exactly `return builtins()`, so `builtins` itself does not need exporting. Cards 6 and 8 carry those retargets. Retarget `detect.go`'s `ErrNoLanguage` reference to `quarryengine.ErrNoLanguage` and add the corresponding import. Nothing else changes.
- **Commit:** `refactor(quarry): move the language-server registry to internal/quarryengine/registry`

### Card 6: Create the `daemon` package and export its toolchain seams

- **Context:**
  - `internal/quarryengine/errors.go`
  - `internal/quarryengine/log.go`
  - `internal/quarryengine/lsp/lspclient.go`
  - `internal/quarryengine/lsp/wire.go`
  - `internal/quarryengine/registry/registry.go`
- **Edits:**
  - `internal/quarryengine/lsp/lspclient.go`
- **Creates:** none
- **Deletes:** none
- **Moves:**
  - `quarry/ensureserver.go` -> `internal/quarryengine/daemon/ensureserver.go`
  - `quarry/toolchain.go` -> `internal/quarryengine/daemon/toolchain.go`
  - `quarry/daemonstate.go` -> `internal/quarryengine/daemon/daemonstate.go`
  - `quarry/probe.go` -> `internal/quarryengine/daemon/probe.go`
  - `quarry/ensureserver_test.go` -> `internal/quarryengine/daemon/ensureserver_test.go`
  - `quarry/ensureserver_integration_test.go` -> `internal/quarryengine/daemon/ensureserver_integration_test.go`
  - `quarry/toolchain_test.go` -> `internal/quarryengine/daemon/toolchain_test.go`
  - `quarry/toolchain_integration_test.go` -> `internal/quarryengine/daemon/toolchain_integration_test.go`
  - `quarry/daemonstate_test.go` -> `internal/quarryengine/daemon/daemonstate_test.go`
  - `quarry/quarrydaemon_test.go` -> `internal/quarryengine/daemon/quarrydaemon_test.go`
  - `quarry/supervised_test.go` -> `internal/quarryengine/daemon/supervised_test.go`
  - `quarry/supervised_lsp_test.go` -> `internal/quarryengine/daemon/supervised_lsp_test.go`
  - `quarry/supervised_integration_test.go` -> `internal/quarryengine/daemon/supervised_integration_test.go`
- **Requirements:** All thirteen moved files declare `package daemon`. Delete the `defaultLogHandler` declaration and its doc comment from `ensureserver.go` (card 3 re-homed it) and retarget its two call sites at lines 447 and 543 to `quarryengine.Logger`. Export the identifiers `query` needs: `ensureServer` -> `EnsureServer`, `connKind` -> `ConnKind` with `connKindNative`/`connKindSupervised`/`connKindLegacy` -> `ConnKindNative`/`ConnKindSupervised`/`ConnKindLegacy`, and `rootURIFor` -> `RootURIFor`. Export the two toolchain test seams and the type one of them names: `installGoToolchain` -> `InstallGoToolchain`, `userCacheDir` -> `UserCacheDir`, `toolchainInstaller` -> `ToolchainInstaller`; give each exported var a doc comment stating it is an injection point that exists so tests can substitute it, and that production code never reassigns it. Export two more identifiers that `daemontest` needs in card 7 so `query`'s integration test can inspect and clean up a recorded daemon: `daemonState` -> `State` and `readDaemonState` -> `ReadState`. Leave `finalizeConnection`, `nativeArgv`, `supervisedArgv`, `reconnectUnderLock`, `ensureNative`, `ensureSupervised`, `runGoInstall`, `resolveGoToolchain`, `goToolchainCacheDir`, `goToolchainInstallLock`, `probe`, `writeDaemonState`, `daemonStale`, `supervisedProtocolVersion`, `daemonIdleTimeout`, and `spawnRacePollInterval` unexported. `daemon` keeps its own in-package `killRecordedDaemon` helper in `supervised_integration_test.go` unchanged — it cannot import `daemontest` (import cycle in test), exactly as with `withFakeInstaller` and `repoRoot`. `DaemonStateFile` and `DaemonLock` are already exported and keep their names. Retarget every `*lspClient` to `*lsp.Client` and its methods to the exported names from card 4, every `lspPosition`/`lspLocation` to `lsp.Position`/`lsp.Location`, every `Entry` to `registry.Entry`, and every `Err*` reference to `quarryengine.Err*`, adding the three imports. `ensureNative` and `ensureSupervised` both assign `client.lang = lang` after construction, a same-package field write in the pre-move code that does not compile once `daemon` and `lsp` are separate packages; card 4 did not export this field because nothing outside `lsp` needed it at the time. Export it here as `Lang` on `lsp.Client` (struct field only, in `lspclient.go`), retarget its two internal uses in `lspclient.go`'s `Close`/`Kill` diagnostic `Warn` calls from `c.lang` to `c.Lang`, and retarget both assignment sites in `ensureserver.go` from `client.lang` to `client.Lang`. Apply the same retargeting inside all nine moved test files. Two further retargets apply to the `//go:build lsp` files only. First, `builtins()` is not visible from `package daemon`: replace all ten occurrences with `registry.BuiltinRegistry()` — nine code call sites plus one doc-comment mention, namely `ensureserver_integration_test.go` lines 44, 50, 68, 72, 132, 145, 176 and its line-6 doc-comment mention, `toolchain_integration_test.go` line 41, and `supervised_integration_test.go` line 46. Second, `repoRoot(t)` is defined in `refs_integration_test.go`, which card 8 moves to `package query`; `daemon` cannot import `query`, so add a package-local `repoRoot(t *testing.T) string` to `ensureserver_integration_test.go`, copied from that definition and serving both its own three call sites (lines 50, 71, 135) and `supervised_integration_test.go`'s one (line 50). It must walk FOUR `filepath.Dir` levels up from `runtime.Caller(0)`, not the original's two — the file now sits at `internal/quarryengine/daemon/` rather than `quarry/` — and its doc comment must say so, since silently keeping two levels yields a path that exists but is the wrong directory. Duplicating this seven-line stdlib helper across `daemon` and `query` is deliberate; see the overview's *test-support helpers* Decision.
- **Commit:** `refactor(quarry): move the daemon lifecycle to internal/quarryengine/daemon`

### Card 7: Create the `daemontest` test-support package

- **Context:**
  - `internal/quarryengine/daemon/toolchain.go`
  - `internal/quarryengine/daemon/toolchain_test.go`
  - `internal/quarryengine/daemon/daemonstate.go`
  - `internal/quarryengine/daemon/supervised_integration_test.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/daemon/daemontest/daemontest.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create `internal/quarryengine/daemon/daemontest/daemontest.go` declaring `package daemontest`, holding two exported helpers modelled on the ones in `toolchain_test.go`: `WithFakeInstaller(t *testing.T, fake daemon.ToolchainInstaller)`, which replaces `daemon.InstallGoToolchain` and restores the previous value via `t.Cleanup`; and `WithTempUserCacheDir(t *testing.T) string`, which replaces `daemon.UserCacheDir` with a closure returning a fresh `t.TempDir()`, restores via `t.Cleanup`, and returns that directory. Copy their doc comments from `toolchain_test.go`, adjusted for the new names. Do NOT delete or modify `withFakeInstaller` and `withTempUserCacheDir` in `toolchain_test.go`, and do NOT rewire `toolchain_test.go`, `toolchain_integration_test.go`, or `ensureserver_test.go` to call the `daemontest` versions: all three are in-package `package daemon` test files, and Go rejects an in-package test that imports a package which imports the package under test — `daemon [test] -> daemontest -> daemon` fails to build with `import cycle not allowed in test`. This was confirmed against a scratch module before the plan was written, not assumed. `daemon`'s own tests therefore keep using their in-package helpers, unchanged, and `daemontest` exists exclusively for callers outside `package daemon` — today that is `query`'s `refs_test.go` in card 8. Add two further exported helpers that `query`'s `//go:build lsp` test needs, for the same reason and by the same route: `StateFile(stateDir, lang string) string`, a one-line delegation to `daemon.DaemonStateFile`; and `KillRecordedDaemon(t *testing.T, statePath string)`, ported from `supervised_integration_test.go:29`'s `killRecordedDaemon` and rewritten against the names card 6 exports — `daemon.ReadState(statePath)` instead of `readDaemonState`, and `daemon.State`'s `PID` field — with the same silently-return-on-error-or-not-found behaviour, since it runs from `t.Cleanup` where a failure to find a dead daemon is not an error. Give the package a doc comment stating exactly this: no production code, exists so tests outside `package daemon` can drive `daemon`'s exported injection points and inspect its recorded state, deliberately duplicates helpers that `daemon`'s own in-package tests keep because those copies cannot be replaced by an import, and is imported only from `_test.go` files.
- **Commit:** `test(quarry): add the daemontest package for out-of-package toolchain seam access`

### Card 8: Create the `query` package

- **Context:**
  - `internal/quarryengine/errors.go`
  - `internal/quarryengine/position.go`
  - `internal/quarryengine/lsp/lspclient.go`
  - `internal/quarryengine/lsp/wire.go`
  - `internal/quarryengine/registry/registry.go`
  - `internal/quarryengine/registry/load.go`
  - `internal/quarryengine/registry/detect.go`
  - `internal/quarryengine/daemon/ensureserver.go`
  - `internal/quarryengine/daemon/daemontest/daemontest.go`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:**
  - `quarry/definition.go` -> `internal/quarryengine/query/definition.go`
  - `quarry/refs.go` -> `internal/quarryengine/query/refs.go`
  - `quarry/symbol.go` -> `internal/quarryengine/query/symbol.go`
  - `quarry/definition_test.go` -> `internal/quarryengine/query/definition_test.go`
  - `quarry/refs_test.go` -> `internal/quarryengine/query/refs_test.go`
  - `quarry/refs_integration_test.go` -> `internal/quarryengine/query/refs_integration_test.go`
  - `quarry/symbol_test.go` -> `internal/quarryengine/query/symbol_test.go`
- **Requirements:** All seven moved files declare `package query`. `Reference`, `InFileQuery`, `Query`, `Options`, `SymbolMatch`, `References`, `Definition`, and `Symbol` keep their names and signatures exactly; `Options.Registry` keeps its type, now written `registry.Registry`. Retarget `acquireConnection`, `teardownConnection`, `lookup`, `resolvePosition`, `collectInFileMatches`, `symbolFromClient`, and `toSortedReferences` to the exported names from cards 4, 5 and 6 — types AND method calls alike. Every `*lspClient` method invocation across `refs.go`, `definition.go`, and `symbol.go` retargets to card 4's exported name: `.references(` -> `.References(`, `.definition(` -> `.Definition(`, `.initialize(` -> `.Initialize(`, `.workspaceSymbol(` -> `.WorkspaceSymbol(`, `.close()` -> `.Close()`, `.kill()` -> `.Kill()`, `.supportsWorkspaceSymbol()` -> `.SupportsWorkspaceSymbol()`, `.supportsDocumentSymbol()` -> `.SupportsDocumentSymbol()`, and — the one that is not a plain capitalization — `.documentSymbol(` -> `.DocumentSymbols(`, plural, per card 4's readability choice. The type substitutions are: `*lspClient` -> `*lsp.Client`, `newLSPClient` -> `lsp.NewClient`, `lspPosition`/`lspLocation`/`lspDocumentSymbol` -> `lsp.Position`/`lsp.Location`/`lsp.DocumentSymbol`, `toLSPPosition` -> `lsp.ToPosition`, `formatLocation` -> `lsp.FormatLocation`, `ensureServer` -> `daemon.EnsureServer`, `connKind*` -> `daemon.ConnKind*`, `rootURIFor` -> `daemon.RootURIFor`, `Entry`/`Registry`/`DetectLanguage` -> `registry.Entry`/`registry.Registry`/`registry.DetectLanguage`, and every `Err*` -> `quarryengine.Err*`. The `Position` a caller supplies via `Query.Pos` stays `quarryengine.Position`. In `refs_test.go`, replace the calls to `withTempUserCacheDir(t)` and `withFakeInstaller(t, …)` at lines 391 and 394 with `daemontest.WithTempUserCacheDir(t)` and `daemontest.WithFakeInstaller(t, …)`; this is the cross-package seam the `daemontest` Decision exists for and is the one place `query`'s tests may reach into `daemon`. Three further changes land in `refs_integration_test.go`, all `//go:build lsp` and therefore only caught by the tagged compile pass. Replace all six occurrences of `builtins()` with `registry.BuiltinRegistry()`, which is not visible from `package query` otherwise — five code call sites at lines 71, 93, 188, 207, 245, plus the line-180 doc-comment mention. Line 188 is `t.Skip(builtins()["go"].InstallHint)` inside `TestReferences_InFile_Integration`; it is easy to miss because it sits in a subtest whose skip guard mirrors line 71's. Keep its local `repoRoot(t)` definition where it is, but change it to walk FOUR `filepath.Dir` levels up from `runtime.Caller(0)` instead of two — the file now sits at `internal/quarryengine/query/` rather than `quarry/` — and update its doc comment accordingly; card 6 adds an independent copy for `daemon`, since `daemon` cannot import `query`. Retarget the daemon-state calls, which `package query` cannot reach directly under the layering rule card 11 pins: `DaemonStateFile(stateDir, "go")` at lines 86, 200 and 238 becomes `daemontest.StateFile(stateDir, "go")`, and `killRecordedDaemon(t, statePath)` at lines 87, 201 and 239 becomes `daemontest.KillRecordedDaemon(t, statePath)` — the latter is defined only in `daemon`'s `supervised_integration_test.go`, which stays in `daemon`, so the `daemontest` copy is the only reachable one. Retarget the two fixture paths at lines 75 and 193 from `filepath.Join(root, "quarry", "detect.go")` to `filepath.Join(root, "internal", "quarryengine", "registry", "detect.go")`, since `detect.go` moved in card 5 and `findFuncPosition` would otherwise fail to read it. Keep the historical `internal/scoutengine` comment at `refs_integration_test.go:55` exactly as it is.
- **Commit:** `refactor(quarry): move the query verbs to internal/quarryengine/query`

### Card 9: Rebuild `quarry/` as a thin facade

- **Context:**
  - `internal/cli/cli.go`
  - `internal/quarryengine/errors.go`
  - `internal/quarryengine/position.go`
  - `internal/quarryengine/registry/registry.go`
  - `internal/quarryengine/registry/load.go`
  - `internal/quarryengine/registry/detect.go`
  - `internal/quarryengine/daemon/daemonstate.go`
  - `internal/quarryengine/query/definition.go`
  - `internal/quarryengine/query/refs.go`
  - `internal/quarryengine/query/symbol.go`
- **Edits:** none
- **Creates:**
  - `quarry/facade.go`
- **Deletes:** none
- **Moves:**
  - `quarry/doc.go` -> `internal/quarryengine/doc.go`
- **Requirements:** `internal/quarryengine/doc.go` declares `package quarryengine`; move it with `git mv` and change only the package clause in this card — batch 4 rewrites its prose. Create `quarry/facade.go` declaring `package quarry`, headed by a short package doc comment (roughly 20 lines) stating that this package is a stable, behaviour-free re-export of `internal/quarryengine`, that it adds nothing of its own, and pointing at `internal/quarryengine`'s own documentation for the engine's design. The file re-exports exactly the 29 identifiers `quarry/` exported before this task, and nothing else. Fourteen type aliases, written with `=` so `errors.As` and type assertions in `internal/cli` keep matching: `Entry = registry.Entry`, `Registry = registry.Registry`, `Position = quarryengine.Position`, `Query = query.Query`, `InFileQuery = query.InFileQuery`, `Options = query.Options`, `Reference = query.Reference`, `SymbolMatch = query.SymbolMatch`, `ErrServerNotFound = quarryengine.ErrServerNotFound`, `ErrSymbolNotFound = quarryengine.ErrSymbolNotFound`, `ErrAmbiguousSymbol = quarryengine.ErrAmbiguousSymbol`, `ErrResolverUnsupported = quarryengine.ErrResolverUnsupported`, `ErrServerTimeout = quarryengine.ErrServerTimeout`, `ErrServerSpawnTimeout = quarryengine.ErrServerSpawnTimeout`. Seven sentinel re-export vars bound to the identical `error` value, never re-created with `errors.New`: `ErrNoLanguage`, `ErrServerNotFoundSentinel`, `ErrSymbolNotFoundSentinel`, `ErrAmbiguousSymbolSentinel`, `ErrResolverUnsupportedSentinel`, `ErrServerTimeoutSentinel`, `ErrServerSpawnTimeoutSentinel`. Eight delegating functions whose signatures match the originals byte-for-byte and whose bodies are a single call: `BuiltinRegistry`, `LoadRegistry`, `DetectLanguage` to `registry`; `DaemonStateFile`, `DaemonLock` to `daemon`; `References`, `Definition`, `Symbol` to `query`. Do not add, remove, or re-sign any exported identifier. Do not edit `internal/cli/cli.go` — it is listed as read-only context so the implementer can confirm the facade covers every call site it makes.
- **Commit:** `refactor(quarry): rebuild quarry as a thin facade over internal/quarryengine`

### Card 10: Relocate the engine/CLI seam guard as a subtree walker

- **Context:**
  - `internal/quarryengine/errors.go`
  - `internal/quarryengine/lsp/lspclient_guard_test.go`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:**
  - `quarry/seam_enforcement_test.go` -> `internal/quarryengine/seam_enforcement_test.go`
- **Requirements:** The moved file declares `package quarryengine`. Rewrite `TestEngineSeamInvariant_BannedImports` so that instead of reading one directory it walks two trees rooted at the module root, resolved relative to this file via `runtime.Caller(0)`: `internal/quarryengine/` recursively (every subpackage, including `lsp`, `registry`, `daemon`, `daemon/daemontest`, `query`) and `quarry/`. It visits non-test `.go` files only — a banned import matters when it ships in production code. The banned list is unchanged and stays a banned list, not an allowlist: `github.com/Knatte18/quarry/internal/output`, any path containing `spf13/cobra`, and any path containing `/internal/` that ends in `cli`. Keep the existing `parsedCount == 0` fatal. Add one further assertion: the walk must have visited at least six distinct package directories, so a package added later and silently skipped cannot let the guard go green by finding nothing to check. Update the file's own doc comment to describe the widened scope.
- **Commit:** `test(quarry): widen the engine seam guard to walk the whole engine tree`

## Batch Tests

`verify:` runs `go test ./... && go test -tags lsp -run "^$" ./...`, which is the whole point of this batch's gate: nothing narrower proves that a 34-file repackaging left the import graph intact, and the full suite completes in about two seconds here. The second invocation compiles the five `//go:build lsp` files without running them, since no `gopls` is present — `supervised_lsp_test.go`, `supervised_integration_test.go`, `ensureserver_integration_test.go`, `toolchain_integration_test.go` (all now under `daemon/`) and `refs_integration_test.go` (under `query/`) all undergo heavy identifier retargeting in cards 6 and 8 and would otherwise go unchecked.

No new test scenario is introduced. The bar is that every existing subtest still exists, still runs, and still passes under its new package — count subtests before and after and compare. Two suites carry specific weight. The untouched `internal/cli` suite is the equivalence proof for the facade: it exercises `quarry.References`, `quarry.Definition`, `quarry.Symbol`, `quarry.BuiltinRegistry`, `quarry.LoadRegistry`, and both `errors.As` paths without being adapted to the new layout, so if it passes unmodified the facade is faithful. `refs_test.go`'s `TestReferences_HasNativeDaemonRoutesThroughEnsureServer` is the subtest the `daemontest` package exists for — it must still run and still pass from `package query`, and its passing is what proves the exported seam works across the package boundary.

Two guards are adapted here rather than in batch 3 because both fail the instant their targets move: `lspclient_guard_test.go` (card 4) must be shown to still fail on a planted third-party import and on a planted first-party import other than `internal/quarryengine`, and `seam_enforcement_test.go` (card 10) must be shown to still fail on a planted `internal/output` import in any of the five engine packages. Verify both by planting, running, and reverting before accepting them as green.
