# Discussion: Thin quarry/ facade over internal/quarryengine

```yaml
task: Thin quarry/ facade over internal/quarryengine
slug: quarry-thin-facade
status: discussing
parent: main
```

## Problem

The `quarry/` directory at the module root holds the entire engine as one flat Go package: 20 production files (~2,900 LOC) plus 21 test files (~4,100 LOC). Those 20 files mix two clearly distinct concerns — LSP daemon lifecycle (`registry.go`, `ensureserver.go`, `lspclient.go`, `toolchain.go`, `daemonstate.go`, `probe.go`) and the query verbs (`definition.go`, `refs.go`, `symbol.go`, `detect.go`, `position.go`, `load.go`) — with `errors.go` and a 289-line `doc.go` on top.

The original justification for keeping this at the module root rather than under `internal/` was importability from outside the repo. That justification does not hold: the one documented future consumer, `cmd/quarry-mcp`, lives in this same module, and Go's `internal/` rule only blocks packages outside the tree containing the `internal/` segment — `cmd/quarry`, `internal/cli`, and a future `cmd/quarry-mcp` can all import `internal/quarryengine/...` as freely as they import `quarry/` today. So the package is public in a way nothing outside the repo consumes, and flat in a way that hides the structure the code already has.

Why now: no external forcing function — this is deliberate structural cleanup taken while the engine's public surface is still small (15 exported identifiers) and before `cmd/quarry-mcp` lands and doubles the number of consumers that would have to be migrated later. A second, smaller reason rides along: every error and log message in the engine still carries a `scoutengine: ` prefix inherited from the Loomyard port, naming a package that no longer exists anywhere in this repo.

## Scope

**In:**

- Move all 20 production files out of `quarry/` into a five-package DAG under `internal/quarryengine/` (layout fixed below under Decisions → *Package layout*).
- Export the identifiers that must now cross a package boundary (`lspClient` → `lsp.Client` and its methods, `ensureServer` → `daemon.EnsureServer`, `connKind` → `daemon.ConnKind`, the LSP wire types, `rootURIFor`, etc.).
- Replace `quarry/`'s contents with a thin facade: type aliases + delegating wrapper functions covering exactly today's exported surface, at the unchanged import path `github.com/Knatte18/quarry/quarry`.
- Move all 21 test files alongside the production files they cover, keeping them in-package (white-box), splitting any file whose subtests straddle a new package boundary.
- Rewrite `quarry/seam_enforcement_test.go` as a subtree-walking guard at `internal/quarryengine/seam_enforcement_test.go`.
- Retarget and widen `lspclient_guard_test.go` (stdlib-only guard) to the new `lsp` package.
- Add a layering guard test that pins the new package DAG.
- Split `quarry/doc.go`: architecture overview to `internal/quarryengine/doc.go`, per-package doc comments in each subpackage, short facade doc in `quarry/doc.go`.
- Replace every `scoutengine: ` prefix in engine error and log messages with `quarry: ` (60 occurrences across 9 files), and update `docs/port-equivalence.md`'s note that documents the old prefix.

**Out:**

- Any behavioural change beyond the `scoutengine: ` → `quarry: ` message-prefix rename. No control-flow, timeout, teardown, retry, or protocol changes.
- `internal/cli`, `internal/output`, `internal/lock`, `internal/proc`, `cmd/quarry` — all untouched. `internal/cli/cli.go`'s import line stays byte-for-byte identical.
- The public exported surface of `quarry/`: nothing added, nothing removed, no signature changes.
- `go.mod`, module path, binary name, CLI flags, JSON output envelope.
- Historical documents in `docs/` that describe Loomyard's `internal/scoutengine` as a past state (`scout-multilang.md`, `scout-vs-grep.md`, `scout-agent-usage-findings.md`, and `port-equivalence.md`'s provenance tables). Only `port-equivalence.md`'s live claim about the current prefix (lines ~105–106) is corrected.
- Creating `cmd/quarry-mcp` or any second consumer.

## Decisions

### Package layout — five packages, not two

- **Decision:** the engine becomes a five-package DAG:

  | Package | Import path | Files |
  |---|---|---|
  | `quarryengine` (root) | `internal/quarryengine` | `errors.go`, `position.go` (caller-facing `Position` only), `log.go` (`defaultLogHandler`), `doc.go` |
  | `lsp` | `internal/quarryengine/lsp` | `lspclient.go`, `wire.go` (LSP wire types + conversion, from the rest of `position.go`) |
  | `registry` | `internal/quarryengine/registry` | `registry.go`, `load.go`, `detect.go` |
  | `daemon` | `internal/quarryengine/daemon` | `ensureserver.go`, `toolchain.go`, `daemonstate.go`, `probe.go` |
  | `query` | `internal/quarryengine/query` | `definition.go`, `refs.go`, `symbol.go` |

  Allowed import directions: `quarryengine` imports no subpackage; `lsp` and `registry` import `quarryengine` only; `daemon` imports `quarryengine`, `registry`, `lsp`; `query` imports all four.

- **Rationale:** the literal daemon/query split named in the task brief does not compile. Measured cross-file dependencies (comments stripped) show a mutual dependency between the two proposed halves: `lspclient.go` (daemon half) uses `lspPosition` ×2, `lspRange` ×4, `lspLocation` ×10, and `formatLocation` ×2 — all declared in `position.go` (query half). Both halves also depend on `errors.go` (daemon: 5 identifiers; query: 7), and `lspclient.go` uses `defaultLogHandler`, which is declared in `ensureserver.go`. A shared leaf package is therefore forced, not optional — this settles the brief's step-7 open question in favour of "yes, a shared package", though it lands as the `quarryengine` root package rather than a sibling named `shared`.

  Given a shared leaf is mandatory anyway, the remaining boundaries follow the dependency graph rather than being invented: `registry.go`/`load.go`/`detect.go` form a self-contained cluster (`Entry`, `Registry`, `builtins`, `precedence`, `validateEntry` are used by `load.go` and `detect.go` internally, and only `Entry`/`Registry`/`DetectLanguage` leak outward), and `lspclient.go` + the wire types form another (the JSON-RPC transport, independent of who spawns the process). What remains in `daemon` is exactly the process-lifecycle concern: spawn-or-reuse, toolchain resolution, state file, health probe.

- **Rejected:**
  - *Two packages (`daemon` + `query`) as literally described in the brief* — requires dumping `errors.go`, `position.go`, `registry.go`, `load.go`, `detect.go` and the log handler into `daemon` to break the cycle, leaving `daemon` with 17 files and `query` with 3. The name `daemon` would then cover the registry, language detection, and the caller-facing position type, none of which are daemon lifecycle.
  - *One package `internal/quarryengine`, flat move only* — zero export churn and zero cycle risk, but delivers none of the structural benefit the task exists for; the 20-file flat directory just changes address.
  - *A sibling `internal/quarryengine/shared` package alongside `daemon`/`query`* — same content as the root package here, but `shared` is a name that describes provenance rather than purpose, and it leaves `internal/quarryengine` itself as an empty directory with no package.

### `Position` lives in the root package; the wire types live in `lsp`

- **Decision:** `position.go` splits. The caller-facing `Position` struct (1-based line, 1-based byte column, as parsed from a `file:line:col` CLI argument) moves to `internal/quarryengine/position.go` in the root package. The LSP wire shapes and the conversion move to `internal/quarryengine/lsp/wire.go`, exported as `lsp.Position`, `lsp.Range`, `lsp.Location`, plus `lsp.ToPosition(quarryengine.Position) (lsp.Position, error)`, `lsp.FormatLocation(lsp.Location) string`, and unexported `utf16Length`.
- **Rationale:** `Position` is part of the engine's caller-facing value vocabulary, alongside the typed error set — both are things every layer names and neither depends on anything. The wire shapes are transport detail owned by the client that serialises them. Two types named `Position` in different packages is idiomatic Go and reads correctly at the one call site that bridges them (`lsp.ToPosition(quarryengine.Position)`); the alternative of renaming one of them produces a worse name on one side or the other.
- **Rejected:** keeping all of `position.go` in `lsp` (drags the caller-facing type behind a transport package that `registry` would then have to import for no reason); putting the wire types in the root package (makes the leaf package carry LSP protocol knowledge that only one subpackage uses).

### `internal/cli` keeps importing the facade

- **Decision:** `internal/cli/cli.go` line 44 stays `"github.com/Knatte18/quarry/quarry"` and no other line in `internal/cli` changes. The facade must therefore cover the full surface the CLI uses.
- **Rationale:** this settles the brief's step-5 open question. Keeping the import gives the facade a real consumer, so the whole `internal/cli` test suite exercises it — a facade nothing imports is a facade that silently rots. It also makes the diff for this task provably scoped: `internal/cli` unchanged is a one-line assertion a reviewer can check.
- **Rejected:** pointing `internal/cli` at `internal/quarryengine/...` directly — leaves the facade dead code on day one, and turns a zero-line `internal/cli` diff into a ~15-call-site rewrite mixed into an already large reorganization.

### Facade surface — exactly today's exports, error types as aliases

- **Decision:** `quarry/` re-exports exactly the 15 identifiers it exports today and nothing more:
  - Type aliases: `Entry`, `Registry` (→ `registry`); `Position` (→ `quarryengine`); `Query`, `InFileQuery`, `Options`, `Reference`, `SymbolMatch` (→ `query`); `ErrServerNotFound`, `ErrSymbolNotFound`, `ErrAmbiguousSymbol`, `ErrResolverUnsupported`, `ErrServerTimeout`, `ErrServerSpawnTimeout` (→ `quarryengine`).
  - Sentinel re-export vars: `ErrNoLanguage`, `ErrServerNotFoundSentinel`, `ErrSymbolNotFoundSentinel`, `ErrAmbiguousSymbolSentinel`, `ErrResolverUnsupportedSentinel`, `ErrServerTimeoutSentinel`, `ErrServerSpawnTimeoutSentinel`.
  - Delegating funcs: `BuiltinRegistry`, `LoadRegistry`, `DetectLanguage`, `DaemonStateFile`, `DaemonLock`, `References`, `Definition`, `Symbol`.
- **Rationale:** the error types must be `type X = quarryengine.X` **aliases**, never new defined types wrapping them — `internal/cli` does `errors.As` / type assertions against `quarry.ErrAmbiguousSymbol` and `quarry.ErrSymbolNotFound` (5 and 1 call sites), and a defined type would silently stop matching an error constructed inside the engine. Same reason the sentinels are re-exported vars pointing at the identical `error` value rather than fresh `errors.New` calls. Data-carrying struct types must be aliases for the same reason; the value types (`Reference`, `Query`, …) are aliases too, for uniformity and so a caller can pass a `query.Options` where a `quarry.Options` is wanted.
  `DetectLanguage`, `DaemonStateFile`, and `DaemonLock` are exported today but unused by `internal/cli`; they are carried over anyway because the task is a reorganization, not a surface audit.
- **Rejected:** narrowing the facade to only what `internal/cli` uses (a surface change smuggled into a reorganization); wrapping instead of aliasing (breaks `errors.As` at CLI call sites).

### Seam enforcement becomes one subtree-walking guard

- **Decision:** delete `quarry/seam_enforcement_test.go`; add `internal/quarryengine/seam_enforcement_test.go` that walks every non-test `.go` file under both `internal/quarryengine/` (recursively) and `quarry/`, applying the same banned list (`internal/output`, `spf13/cobra`, any `/internal/...cli` path). It keeps the existing "scanned zero files → `t.Fatal`" guard and additionally fails if it visited fewer than the expected number of packages, so a future package that is added and then silently skipped cannot go green.
- **Rationale:** the invariant being protected — engine code never turns results into JSON, cobra, or exit codes — is a property of the whole engine tree plus the facade, not of one directory. Five per-package copies of the same 70-line test would drift; one walker cannot.
- **Rejected:** one seam test per package (duplication, drift); leaving the guard in `quarry/` only (would cover the facade and nothing else, i.e. almost nothing).

### `lspclient.go`'s stdlib-only guard is kept and widened by exactly one path

- **Decision:** `lspclient_guard_test.go` moves to `internal/quarryengine/lsp/`, retargets `lspclient.go` at its new path, and its allowed set becomes "the standard library, plus `github.com/Knatte18/quarry/internal/quarryengine` (the root package) — nothing else". Any third-party import still fails, as does any other first-party import.
- **Rationale:** this is a hard constraint on the layout, not a nicety: the existing guard would fail the moment `lspclient.go` imports the root package for `ErrServerTimeout` (2 uses) and `defaultLogHandler` (5 uses). The guard's stated intent is "carries no dependency outside the standard library", aimed at third-party weight, and the root package is a dependency-free leaf of value types — admitting exactly it preserves the intent. The widening is a single hardcoded path, not a per-file allowed-set table, which the guard's own doc comment explicitly forbids.
- **Rejected:** keeping the guard byte-for-byte strict — forces `errors.go` and the log handler into the `lsp` package, which then becomes the package every other package imports for the error vocabulary, i.e. the shared leaf under a misleading name. Deleting the guard — loses a real invariant for a mechanical reason.

### A new layering guard pins the DAG

- **Decision:** add `internal/quarryengine/layering_test.go` asserting the allowed import directions from the *Package layout* table above: the root package imports no `internal/quarryengine/...` path; `lsp` and `registry` import at most the root; `daemon` imports at most root + `registry` + `lsp`; `query` may import all four. Table-driven off the same walk the seam test uses.
- **Rationale:** the layering is the entire deliverable of this task, and it is exactly the kind of property that decays under the next feature commit — one `registry` → `daemon` import would recreate the flat package with extra directories. ~40 lines, no runtime cost, and it fails loudly with the offending file and import path.
- **Rejected:** relying on review discipline; relying on Go's own cycle detection (catches cycles, not layering violations — `registry` importing `daemon` is a legal DAG edge and would pass the compiler while destroying the design).

### `doc.go` splits three ways

- **Decision:** the 289-line `quarry/doc.go` splits into: `internal/quarryengine/doc.go` carrying the architecture overview (the engine/CLI split, the typed error vocabulary, the scope boundaries — rewritten to describe the five-package layout and to name each package's role); a package doc comment at the head of each subpackage's primary file covering that package's own section (`lsp` ← "The generalized LSP client" + the position-conversion paragraph; `registry` ← "The language-server registry"; `daemon` ← "The EnsureServer seam" + "Go toolchain manager" + "Daemon state and concurrency"; `query` ← the resolver and ambiguity paragraphs); and a new ~20-line `quarry/doc.go` stating that this package is a stable thin re-export of `internal/quarryengine`, that it adds no behaviour, and pointing at the engine's own docs.
- **Rationale:** the existing text is accurate and expensive to reproduce — it should be relocated section by section, not rewritten from scratch. Every section already maps cleanly onto exactly one of the new packages, which is itself corroboration that the boundaries are the real ones. References inside the moved prose to files and unexported identifiers (`ensureServer`, `lspClient`, `close()`, `kill()`, `resolvePosition`) must be updated to the new package-qualified, exported names.
- **Rejected:** leaving the full overview in `quarry/doc.go` (documents an implementation the facade no longer contains); duplicating the overview into every package.

### `scoutengine: ` message prefixes become `quarry: `

- **Decision:** replace the literal `scoutengine: ` prefix with `quarry: ` in every engine error and log message — 60 occurrences across `errors.go` (13), `ensureserver.go` (18), `lspclient.go` (8), `daemonstate.go` (6), `toolchain.go` (5), `registry.go` (4), `detect.go` (2), `load.go` (2), `refs.go` (2). Also correct the live claim in `docs/port-equivalence.md` (~lines 105–106) that documents quarry's errors as still carrying the old prefix, including the sample JSON error string quoted there.
- **Rationale:** `scoutengine` is a package name from Loomyard that exists nowhere in this repo; the prefix is a port artifact that leaks a dead name into every CLI error the user sees. The prefix is kept (rather than dropped entirely) because it marks the origin of an error once `internal/cli` has wrapped it into the JSON envelope, and `quarry: ` is the name of both the module and the binary.
- **Rejected:** dropping the prefix entirely (loses origin marking in wrapped errors); deferring to a follow-up task (the operator explicitly pulled it into this task); using `quarryengine: ` (names an internal package that no caller can import — the user-facing name is `quarry`).
- **Note:** this is the one intentional observable change in an otherwise pure reorganization. No existing test asserts the literal prefix — the only other in-repo hits are a historical comment in `refs_integration_test.go:55` and provenance tables in `docs/` describing Loomyard's package, all of which stay as they are.

### Tests move with their files and stay white-box

- **Decision:** every `_test.go` file keeps its `package <pkg>` (in-package) form and moves to the package holding the code it covers; identifiers it references that are now exported get the mechanical rename. Placement:
  - `lsp/`: `lspclient_test.go`, `lspclient_guard_test.go`, `position_test.go`
  - `registry/`: `registry_test.go`, `load_test.go`, `detect_test.go`
  - `daemon/`: `ensureserver_test.go`, `ensureserver_integration_test.go`, `toolchain_test.go`, `toolchain_integration_test.go`, `daemonstate_test.go`, `quarrydaemon_test.go`, `supervised_test.go`, `supervised_lsp_test.go`, `supervised_integration_test.go`
  - `query/`: `definition_test.go`, `refs_test.go`, `refs_integration_test.go`, `symbol_test.go`
  - `internal/quarryengine/`: `seam_enforcement_test.go`, new `layering_test.go`
  - `quarry/`: one new `facade_test.go`
- **Rationale:** these are white-box tests against unexported state by design (fake `io.ReadWriteCloser` transports, injected `installGoToolchain`/`userCacheDir`, hand-built `daemonState` fixtures). Converting them to external `_test` packages would either lose coverage or force further exports that exist only for tests. Keeping them in-package and co-located preserves both.
- **Rejected:** external test packages; a single `internal/quarryengine/alltests` package.

### The facade gets one compile-level surface test

- **Decision:** `quarry/facade_test.go` asserts, in-package, that the facade is a *pure* re-export: for each aliased type, a value of the engine type is assignable to the facade type and back without conversion (which only compiles for a true alias); for each sentinel, `quarry.ErrX == quarryengine.ErrX` by identity; and each delegating func is referenced so a signature drift fails the build.
- **Rationale:** the one way this facade can break silently is by drifting into defined types or re-created sentinels, which compiles fine in `quarry/` and only fails later at an `errors.As` in `internal/cli`. An identity test catches it at the seam.
- **Rejected:** no facade test (the failure mode is silent and lands in a different package); a reflect-based surface-diff test (over-built for 15 identifiers, and the compiler already does the work).

## Technical context

- Module `github.com/Knatte18/quarry`, Go 1.26. Deps: `gofrs/flock`, `spf13/cobra`, `yaml.v3`.
- Current layout: `cmd/quarry/` (main, imports `internal/cli` only), `internal/cli/` (sole engine consumer, imports `quarry` at `cli.go:44`), `internal/output/`, `internal/lock/`, `internal/proc/`, `quarry/`.
- `internal/cli` uses exactly these engine identifiers: `Reference` ×37, `Query` ×14, `BuiltinRegistry` ×8, `Position` ×7, `ErrAmbiguousSymbol` ×5, `SymbolMatch` ×3, `References` ×3, `Options` ×3, `Definition` ×3, `Symbol` ×2, `Registry` ×2, `LoadRegistry` ×2, `InFileQuery` ×2, `ErrSymbolNotFoundSentinel` ×2, `ErrSymbolNotFound` ×1.
- **Identifiers that must become exported** (measured, comments stripped — these are the query-half → daemon-half edges plus the wire types):
  - from `lspclient.go`: `lspClient` → `lsp.Client`; methods `call`, `notify`, `initialize`, `references`, `definition`, `workspaceSymbol`, `documentSymbol`, `close`, `kill`, `supportsWorkspaceSymbol`, `supportsDocumentSymbol` → exported method names; types `lspDocumentSymbol` → `lsp.DocumentSymbol`, `symbolInformation` → `lsp.SymbolInformation`; constructors `newLSPClient` → `lsp.NewClient`, `newLSPClientFromRW`, `newLSPClientDial`. `parseDefinitionResult`, `readLoop`, `readMessage`, `writeMessage`, `capabilities`, `capabilityFlag`, `lspMessage`, `lspError`, `lspReadResult` stay unexported inside `lsp`.
  - from `position.go`: `lspPosition`/`lspRange`/`lspLocation` → `lsp.Position`/`lsp.Range`/`lsp.Location`; `toLSPPosition` → `lsp.ToPosition`; `formatLocation` → `lsp.FormatLocation`. `utf16Length` stays unexported.
  - from `ensureserver.go`: `ensureServer` → `daemon.EnsureServer`; `connKind` → `daemon.ConnKind` with `connKindNative`/`connKindSupervised`/`connKindLegacy` → `daemon.ConnKindNative`/`…Supervised`/`…Legacy`; `rootURIFor` → `daemon.RootURIFor` (used by `refs.go`'s legacy path). `finalizeConnection`, `nativeArgv`, `supervisedArgv`, `reconnectUnderLock`, `ensureNative`, `ensureSupervised`, `daemonIdleTimeout`, `spawnRacePollInterval` stay unexported inside `daemon`.
  - from `registry.go`: `Entry`, `Registry` already exported and stay; `builtins`, `precedence`, `validateEntry` stay unexported (only `load.go`/`detect.go` use them, both landing in the same package).
  - from `ensureserver.go` → root: `defaultLogHandler` moves to `internal/quarryengine/log.go`, exported as `quarryengine.Logger` (or kept as an exported package-level `*slog.Logger`); `lspclient.go`, `ensureserver.go` are its only users (5 + 2 call sites).
- **`refs.go` is the file that touches everything.** It reaches into `lspClient` (×5) and its methods, `connKind` (×2), `ensureServer`, `newLSPClient`, `rootURIFor`, `lspDocumentSymbol` (×3), `Entry`/`Registry`, `DetectLanguage`, and 5 error types. Its `acquireConnection`/`teardownConnection`/`lookup`/`resolvePosition` are the pipeline every verb runs. Expect this file to carry the largest diff.
- **Test seams stay inside one package.** The only package-level vars reassigned by tests are `installGoToolchain` and `userCacheDir` (both `toolchain.go`), reassigned only from `toolchain_test.go` and `toolchain_integration_test.go` — all three files land in `daemon/`, so no test seam has to cross a package boundary. `precedence` and `defaultLogHandler` are never reassigned by tests.
- **`TestProbe_*` lives in `daemonstate_test.go`, not a `probe_test.go`.** This is why `probe.go` is placed in `daemon/` rather than `lsp/`: it keeps those subtests in the same package as their fixtures without splitting a file. `probe` itself only needs `Client.WorkspaceSymbol`, which is exported anyway.
- **`TestFinalizeConnection_*`, `TestReconnectUnderLock_*`, `TestNativeArgv_*`, `TestSupervisedArgv_*` all live in `ensureserver_test.go`** — all target `daemon`-internal functions, so that file moves whole with no split.
- **Build-tagged files:** `supervised_lsp_test.go`, `supervised_integration_test.go`, `refs_integration_test.go`, `ensureserver_integration_test.go`, `toolchain_integration_test.go` carry `//go:build lsp` and need a real `gopls` on `$PATH`. They are excluded from a plain `go test ./...` and must still compile — verify with `go vet -tags lsp ./...` even where they cannot be run.
- Existing guards to preserve: `quarry/seam_enforcement_test.go` (banned-list, not allowlist; direct imports only, never the transitive closure) and `quarry/lspclient_guard_test.go` (its doc comment explicitly forbids generalizing it into a per-file allowed-set table — the widening decided above adds one hardcoded path to one file's check, which stays within that rule).
- No `CONSTRAINTS.md` and no `_codeguide/` in this repo.

## Constraints

- `internal/cli` and `cmd/quarry` must not change, except that `internal/cli`'s expectations about error *message* text shift with the `scoutengine: ` → `quarry: ` rename (no test asserts on it today; confirm during implementation).
- The import path `github.com/Knatte18/quarry/quarry` and every identifier it exports must survive unchanged.
- The engine tree must never import `internal/output`, `spf13/cobra`, or any `internal/*cli` package.
- `internal/quarryengine/lsp/lspclient.go` must import only the standard library plus `internal/quarryengine`.
- The package DAG must stay acyclic and respect the declared layering directions.
- No behaviour change other than the message prefix.

## Testing

The reorganization's safety net is that the existing suite must pass unmodified in behaviour — only file location, package clause, and identifier casing change.

- **Regression baseline (do this first):** capture `go test ./... 2>&1` output and `go build ./...` before touching anything, so the after-state is diffable rather than merely green.
- **`internal/cli` suite is the primary equivalence proof.** It is the only test tree that does not move and does not change; it exercises the facade end to end. If it passes untouched, the facade is faithful. TDD candidate: run it after the facade exists but before deleting the old `quarry/` implementation files, to catch surface gaps early.
- **`quarry/facade_test.go`** — TDD candidate, write it before the facade body. Assertions: alias identity (assign engine value → facade type → engine type, no conversion), sentinel pointer identity, and a reference to every delegating func.
- **`internal/quarryengine/seam_enforcement_test.go`** — TDD candidate. Port the existing walker to recurse, assert it visits both trees, assert non-zero files *and* the expected package count, then confirm it still fails when pointed at a deliberately-planted banned import.
- **`internal/quarryengine/layering_test.go`** — TDD candidate, new. Must be shown to fail on a planted violating import (e.g. `registry` importing `daemon`) before being accepted as green.
- **`lsp/lspclient_guard_test.go`** — verify it still fails on a planted third-party import and on a planted first-party import other than the root package; the widening must not become a hole.
- **Moved suites** (`lsp`, `registry`, `daemon`, `query`) — no new scenarios; the bar is that every existing subtest still exists, still runs, and still passes under its new package. Count subtests before and after.
- **Message-prefix rename** — `grep -rn 'scoutengine' --include='*.go' .` must return only `refs_integration_test.go:55`'s historical comment afterwards. Check whether any `internal/cli` test asserts on a full engine error string; if one does, update it as part of the rename rather than as a separate step.
- **Build-tagged compilation** — `go vet -tags lsp ./...` must be clean even on a machine with no `gopls`, so the `//go:build lsp` files are proven to still compile after their identifier renames.
- **Sequencing risk:** this is a large mechanical diff. Prefer moving one package at a time in a compiling state (`lsp` and `registry` first — they only depend on the root — then `daemon`, then `query`, then the facade) over one atomic move, so a break is localized.

## Q&A log

- **Q:** How many packages should the engine split into, and where do the pieces that both halves need go? **A:** [auto-pick] Five packages — `internal/quarryengine` (root leaf: errors, `Position`, log handler, doc), `lsp`, `registry`, `daemon`, `query`. **Why:** the brief's literal two-way daemon/query split is not implementable — `lspclient.go` depends on `position.go`'s wire types (18 references) while `refs.go`/`symbol.go`/`definition.go` depend on `lspclient.go`, and both halves depend on `errors.go`; a shared leaf is forced, and once forced, the remaining boundaries follow the measured dependency clusters instead of being invented.
- **Q:** Does `internal/cli` import the new facade or `internal/quarryengine` directly? **A:** [auto-pick] The facade, unchanged. **Why:** it keeps the facade exercised by a real consumer instead of leaving it dead code, and it makes "`internal/cli` diff is empty" a one-line check a reviewer can verify.
- **Q:** What exactly does the facade re-export, and as aliases or wrappers? **A:** [auto-pick] Exactly today's 15 exported identifiers; types as `=` aliases, sentinels as re-export vars, functions as delegating wrappers. **Why:** `internal/cli` runs `errors.As`/type assertions against `quarry.ErrAmbiguousSymbol` and `quarry.ErrSymbolNotFound`; a defined type or a fresh `errors.New` would compile in `quarry/` and silently stop matching at the CLI call site.
- **Q:** Where does the caller-facing `Position` type live once the wire types move to `lsp`? **A:** [auto-pick] The root `quarryengine` package, with the wire shapes and conversion in `lsp`. **Why:** `Position` is caller-facing value vocabulary like the error set, depends on nothing, and would otherwise force `registry` to import a transport package.
- **Q:** How is the engine/CLI seam guard adapted to five packages? **A:** [auto-pick] One subtree-walking test at `internal/quarryengine/seam_enforcement_test.go` covering `internal/quarryengine/...` plus `quarry/`, with the zero-files fatal kept and a package-count assertion added. **Why:** the invariant is a property of the whole tree; five copies of the same check would drift apart.
- **Q:** `lspclient.go`'s stdlib-only guard will fail once it imports the root package for `ErrServerTimeout` and the log handler. Keep it strict, widen it, or drop it? **A:** [auto-pick] Widen by exactly one hardcoded path (stdlib + `internal/quarryengine`). **Why:** the guard's intent is keeping third-party weight out of the transport; a dependency-free first-party leaf does not violate that, and keeping it strict would force the error vocabulary into the `lsp` package and wreck the layering.
- **Q:** Should a test pin the new import-direction layering, or is Go's cycle detection enough? **A:** [auto-pick] Add `internal/quarryengine/layering_test.go`. **Why:** the compiler only rejects cycles; `registry` importing `daemon` is a legal DAG edge that would pass the build while dissolving the design this task exists to create.
- **Q:** What happens to the 289-line `quarry/doc.go`? **A:** [auto-pick] Split three ways — overview to `internal/quarryengine/doc.go`, per-section prose to the matching subpackage doc comments, short facade doc in `quarry/doc.go`. **Why:** the text is accurate and expensive to reproduce, and every section already maps onto exactly one new package, which independently corroborates the boundaries.
- **Q:** Do the test files move as-is, and do they stay in-package? **A:** [auto-pick] Move alongside the code they cover, stay in-package white-box, mechanical renames only. **Why:** they drive unexported state deliberately (fake transports, injected `installGoToolchain`/`userCacheDir`, hand-built `daemonState`); converting to external test packages would either lose coverage or force test-only exports.
- **Q:** The `scoutengine: ` prefix on every engine error and log message names a package that no longer exists. In scope or a follow-up? **A:** **In scope — remove it** (operator decision, overriding the auto-pick, which had deferred it as an observable-behaviour change). Replace with `quarry: ` across all 60 occurrences in 9 production files, and fix `docs/port-equivalence.md`'s live claim about the old prefix. **Why:** the operator explicitly pulled it into this task; `quarry: ` rather than no prefix, because the marker still identifies the error's origin once `internal/cli` wraps it into the JSON envelope, and `quarry` is the name of both the module and the binary. This is the single intentional behaviour change in an otherwise pure reorganization and is called out as such under Scope.
