# Plan: Thin quarry/ facade over internal/quarryengine

```yaml
task: Thin quarry/ facade over internal/quarryengine
slug: quarry-thin-facade
approved: false
started: 20260827-112023
parent: main
root: ""
verify: null
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: message-prefix-rename
    file: 01-message-prefix-rename.md
    depends-on: []
    verify: go test ./... && go test -tags lsp -run "^$" ./...
  - number: 2
    name: engine-repackage
    file: 02-engine-repackage.md
    depends-on: [1]
    verify: go test ./... && go test -tags lsp -run "^$" ./...
  - number: 3
    name: architecture-guards
    file: 03-architecture-guards.md
    depends-on: [2]
    verify: go test ./... && go test -tags lsp -run "^$" ./...
  - number: 4
    name: doc-redistribution
    file: 04-doc-redistribution.md
    depends-on: [2]
    verify: go test ./... && go test -tags lsp -run "^$" ./...
```

## Shared Decisions

### Decision: package layout is a five-package DAG under `internal/quarryengine`

- **Decision:** the engine is split into `internal/quarryengine` (root leaf: `errors.go`, `position.go`, `log.go`, `doc.go`), `internal/quarryengine/lsp` (`lspclient.go`, `wire.go`), `internal/quarryengine/registry` (`registry.go`, `load.go`, `detect.go`), `internal/quarryengine/daemon` (`ensureserver.go`, `toolchain.go`, `daemonstate.go`, `probe.go`), and `internal/quarryengine/query` (`definition.go`, `refs.go`, `symbol.go`). A sixth, test-support-only package `internal/quarryengine/daemon/daemontest` sits outside the production DAG. Allowed import directions: the root imports no subpackage; `lsp` and `registry` import the root only; `daemon` imports root + `registry` + `lsp`; `query` imports all four; `daemontest` imports `daemon` and is imported only from `_test.go` files.
- **Rationale:** the brief's literal two-way daemon/query split does not compile — `lspclient.go` uses `position.go`'s wire types while `refs.go`/`symbol.go`/`definition.go` use `lspclient.go`, and both halves use `errors.go`. A shared leaf package is forced; given that, the remaining boundaries follow the measured dependency clusters. Full rationale and rejected alternatives: `_mill/discussion.md` → Decisions → *Package layout*.
- **Applies to:** all batches

### Decision: no behaviour change except the message-prefix rename

- **Decision:** this is a pure reorganization. The one intentional observable change is `scoutengine: ` → `quarry: ` (and the single `scout: ` outlier) in engine error and log messages, delivered in batch 1 before any file moves. No control-flow, timeout, teardown, retry, or protocol change anywhere.
- **Rationale:** separating the string rename from the move keeps batch 2's diff purely structural, so a reviewer can read it as a rename diff rather than hunting for behaviour changes inside a 34-file move.
- **Applies to:** all batches

### Decision: `internal/cli` and `cmd/quarry` are never edited

- **Decision:** no card in this plan lists any file under `internal/cli/`, `internal/output/`, `internal/lock/`, `internal/proc/`, or `cmd/` in `Edits:`, `Creates:`, `Deletes:`, or `Moves:`. `internal/cli/cli.go:44` keeps importing `github.com/Knatte18/quarry/quarry` byte-for-byte unchanged.
- **Rationale:** the untouched `internal/cli` suite is the equivalence proof for the whole reorganization — it exercises the facade end to end without being adapted to it. "The `internal/cli` diff is empty" is a one-line check a reviewer can verify.
- **Applies to:** all batches

### Decision: the facade re-exports today's 29 identifiers, types as aliases

- **Decision:** `quarry/facade.go` re-exports exactly the 29 top-level identifiers `quarry/` exports today: type aliases (`type X = pkg.X`) for `Entry`, `Registry`, `Position`, `Query`, `InFileQuery`, `Options`, `Reference`, `SymbolMatch` and the six error types; re-export vars for the seven `*Sentinel`/`ErrNoLanguage` values; delegating wrapper functions for `BuiltinRegistry`, `LoadRegistry`, `DetectLanguage`, `DaemonStateFile`, `DaemonLock`, `References`, `Definition`, `Symbol`.
- **Rationale:** the error types must be `=` aliases, never new defined types — `internal/cli` runs `errors.As` and type assertions against `quarry.ErrAmbiguousSymbol` and `quarry.ErrSymbolNotFound`, and a defined type would compile in `quarry/` while silently ceasing to match an error constructed inside the engine. The sentinels are re-export vars pointing at the identical `error` value, never fresh `errors.New` calls, for the same reason.
- **Applies to:** engine-repackage, architecture-guards

### Decision: the facade's package doc comment lives at the head of `quarry/facade.go`

- **Decision:** `quarry/` gets no `doc.go` of its own. The short facade package comment heads `quarry/facade.go`. The existing `quarry/doc.go` is `git mv`-ed to `internal/quarryengine/doc.go`.
- **Rationale:** `_mill/discussion.md` describes the facade doc as living in `quarry/doc.go`, but that path is simultaneously the source of a `Moves:` pair, and a path may not appear in both `Moves:` and `Creates:`. Hosting the package comment in `facade.go` is idiomatic Go, preserves the move's git rename history, and changes nothing about the delivered content.
- **Applies to:** engine-repackage, doc-redistribution

### Decision: every rename is a `git mv`, never a create-plus-delete

- **Decision:** all 34 relocated files (14 production, 20 test) are moved with `git mv` first, then surgically edited — package clause, import block, and the specific identifier retargets named per card. No relocated file is rewritten from scratch.
- **Rationale:** a create-plus-delete destroys git rename detection across a 34-file move, turning a reviewable rename diff into ~7,000 lines of add plus ~7,000 lines of delete.
- **Applies to:** engine-repackage

### Decision: tests stay in-package (white-box) and move with the code they cover

- **Decision:** every relocated `_test.go` file keeps its in-package `package <pkg>` clause. No test is converted to an external `_test` package. The one seam that cannot survive the split — `refs_test.go`'s use of `toolchain_test.go`'s helpers — is resolved by exporting the two `toolchain.go` seams and routing all callers through `daemontest`, not by converting test packages.
- **Rationale:** these tests drive unexported state deliberately (fake `io.ReadWriteCloser` transports, injected installer/cache-dir seams, hand-built `daemonState` fixtures). Converting them would either lose coverage or force test-only exports far beyond the two the `daemontest` decision already accepts.
- **Applies to:** engine-repackage

### Decision: verify runs the full module plus a tagged compile

- **Decision:** every batch verifies with `go test ./... && go test -tags lsp -run "^$" ./...`.
- **Rationale:** this is a whole-module repackaging — there is no narrower scope that would catch a broken import graph, and the full suite runs in about two seconds on this repo. The second invocation compiles the five `//go:build lsp` files (`supervised_lsp_test.go`, `supervised_integration_test.go`, `refs_integration_test.go`, `ensureserver_integration_test.go`, `toolchain_integration_test.go`) without running them, since they need a real `gopls` on `$PATH` that is not present in this environment. Both commands were confirmed exit-0 against the pre-change worktree tip.
- **Applies to:** all batches

### Decision: `pipeline.done_gate` is left `null`

- **Decision:** `mill-config.yaml` is not modified by this plan.
- **Rationale:** every batch's own `verify:` is already the repo-wide `go test ./...`, so a done-gate would run the identical command a second time. `golangci-lint` is not installed in this environment, so the lint-default fallback does not apply either.
- **Applies to:** all batches

## All Files Touched

- `docs/port-equivalence.md`
- `internal/quarryengine/daemon/daemonstate.go`
- `internal/quarryengine/daemon/daemonstate_test.go`
- `internal/quarryengine/daemon/daemontest/daemontest.go`
- `internal/quarryengine/daemon/ensureserver.go`
- `internal/quarryengine/daemon/ensureserver_integration_test.go`
- `internal/quarryengine/daemon/ensureserver_test.go`
- `internal/quarryengine/daemon/probe.go`
- `internal/quarryengine/daemon/quarrydaemon_test.go`
- `internal/quarryengine/daemon/supervised_integration_test.go`
- `internal/quarryengine/daemon/supervised_lsp_test.go`
- `internal/quarryengine/daemon/supervised_test.go`
- `internal/quarryengine/daemon/toolchain.go`
- `internal/quarryengine/daemon/toolchain_integration_test.go`
- `internal/quarryengine/daemon/toolchain_test.go`
- `internal/quarryengine/doc.go`
- `internal/quarryengine/errors.go`
- `internal/quarryengine/layering_test.go`
- `internal/quarryengine/log.go`
- `internal/quarryengine/lsp/lspclient.go`
- `internal/quarryengine/lsp/lspclient_guard_test.go`
- `internal/quarryengine/lsp/lspclient_test.go`
- `internal/quarryengine/lsp/position_test.go`
- `internal/quarryengine/lsp/wire.go`
- `internal/quarryengine/position.go`
- `internal/quarryengine/query/definition.go`
- `internal/quarryengine/query/definition_test.go`
- `internal/quarryengine/query/refs.go`
- `internal/quarryengine/query/refs_integration_test.go`
- `internal/quarryengine/query/refs_test.go`
- `internal/quarryengine/query/symbol.go`
- `internal/quarryengine/query/symbol_test.go`
- `internal/quarryengine/registry/detect.go`
- `internal/quarryengine/registry/detect_test.go`
- `internal/quarryengine/registry/load.go`
- `internal/quarryengine/registry/load_test.go`
- `internal/quarryengine/registry/registry.go`
- `internal/quarryengine/registry/registry_test.go`
- `internal/quarryengine/seam_enforcement_test.go`
- `quarry/daemonstate.go`
- `quarry/detect.go`
- `quarry/ensureserver.go`
- `quarry/errors.go`
- `quarry/facade.go`
- `quarry/facade_test.go`
- `quarry/load.go`
- `quarry/lspclient.go`
- `quarry/refs.go`
- `quarry/registry.go`
- `quarry/toolchain.go`
