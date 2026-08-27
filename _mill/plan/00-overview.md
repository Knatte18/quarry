# Plan: Improve gopls query precision (build tags + scoping)

```yaml
task: "Improve gopls query precision (build tags + scoping)"
slug: "gopls-query-precision"
approved: false
started: "20260827-174944"
parent: "main"
root: ""
verify: go vet ./... && go vet -tags lsp ./... && go test ./internal/quarryengine/
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: lsp-implementation-and-spike
    file: 01-lsp-implementation-and-spike.md
    depends-on: []
    verify: go vet -tags lsp ./... && go test ./internal/quarryengine/lsp/
  - number: 2
    name: registry-build-tag-template
    file: 02-registry-build-tag-template.md
    depends-on: []
    verify: go test ./internal/quarryengine/registry/ ./internal/quarryengine/
  - number: 3
    name: initialization-options-plumbing
    file: 03-initialization-options-plumbing.md
    depends-on: [1, 2]
    verify: go vet -tags lsp ./... && go test ./internal/quarryengine/lsp/ ./internal/quarryengine/daemon/ ./internal/quarryengine/query/
  - number: 4
    name: callers-verification-entry-point
    file: 04-callers-verification-entry-point.md
    depends-on: [1, 3]
    verify: go vet -tags lsp ./... && go test ./internal/quarryengine/query/ ./quarry/
  - number: 5
    name: cli-surface
    file: 05-cli-surface.md
    depends-on: [3, 4]
    verify: go test ./internal/cli/
  - number: 6
    name: live-tier-tests
    file: 06-live-tier-tests.md
    depends-on: [1, 4, 5]
    verify: PATH="$HOME/.cache/quarry/tools/go/v0.23.0:$PATH" go test -tags lsp ./internal/quarryengine/query/ ./internal/cli/
  - number: 7
    name: docs-and-followups
    file: 07-docs-and-followups.md
    depends-on: [5, 6]
    verify: null
```

## Shared Decisions

_Cross-cutting decisions every batch inherits: naming conventions, error-handling posture, test frameworks, style/lint constraints.
One subsection per decision.
Batch-local decisions live in each batch file._

### Decision: layering-is-non-negotiable

- **Decision:** The five-package engine DAG stays exactly as `internal/quarryengine/layering_test.go` encodes it, and the engine/CLI seam stays exactly as `internal/quarryengine/seam_enforcement_test.go` encodes it. Build-tag normalization and `initializationOptions` template rendering live in `registry`; the rendered `map[string]any` is what travels down through `query` → `daemon` → `lsp`. `internal/quarryengine/lsp/lspclient.go` keeps its stdlib-plus-`internal/quarryengine`-only import set (`internal/quarryengine/lsp/lspclient_guard_test.go` enforces it), so it must never import `registry`: it receives an already-rendered `map[string]any` and nothing more. `internal/cli` reaches the engine only through `quarry/facade.go`.
- **Rationale:** Three existing guard tests already fail loudly on a violation; keeping the new code inside those boundaries means the guards keep working instead of needing amendment.
- **Enforcement:** `layering_test.go` and `seam_enforcement_test.go` live in the `internal/quarryengine` root package, so they run only under `go test ./internal/quarryengine/`. A per-batch `go vet` cannot fail them — vet type-checks, it does not run a custom guard test. That command is therefore part of the overview's module-wide `verify:`, which runs at every batch boundary, so a batch that adds a file anywhere under `internal/quarryengine/` or `quarry/` re-runs both guards rather than landing green on an import that drifts across the DAG.
- **Applies to:** all batches

### Decision: empty-tag-set-is-a-uniform-no-op

- **Decision:** An empty normalized tag set is a no-op across all three mechanisms: no `initializationOptions` key is sent on `initialize`, no `tags-<hex>` segment is appended to the resolved state directory, and `nativeArgv` produces today's argv byte for byte (including `-remote=auto` and `-remote.listen.timeout`). Every batch that touches one of those three mechanisms writes the back-compat assertion first.
- **Rationale:** The overwhelming majority of queries are untagged. They must be provably unchanged, and "provably" means an assertion per mechanism, not a claim.
- **Applies to:** all batches

### Decision: rendered-options-non-nil-means-tagged

- **Decision:** `registry.RenderInitializationOptions` returns `(nil, nil)` for an empty tag set and a non-nil map for a non-empty one. Every downstream consumer keys off that one signal: `daemon.EnsureServer` treats a non-nil `initOptions` as "this query is tagged" and therefore spawns a private gopls on the native path, and `lsp.Client.Initialize` sends the `initializationOptions` key exactly when its argument is non-nil.
- **Rationale:** One derivation of "is this query tagged?", computed once in `registry`, rather than a `[]string` and a `map` travelling in parallel down three packages and drifting apart.
- **Applies to:** registry-build-tag-template, initialization-options-plumbing, callers-verification-entry-point, cli-surface

### Decision: verification-is-fail-closed-everywhere

- **Decision:** Verification only ever removes a reference when it positively disproves it. A per-reference `textDocument/definition` that errors, times out, or returns an empty result keeps its reference. A declaration side that is unusable — the declaration-side `textDocument/definition` errored or returned an empty location set, or the `textDocument/implementation` half is unavailable (capability unadvertised, or the call errored) — skips verification entirely and keeps every reference. Verification never turns a lookup into an error, and it is never applied against an empty declaration set.
- **Rationale:** `assert-no-callers` is a delete/move safety gate. A degraded server must never be able to make it green.
- **Applies to:** callers-verification-entry-point, cli-surface, live-tier-tests

### Decision: gopls-lives-in-the-toolchain-cache-not-on-path

- **Decision:** `gopls` is not on `$PATH` on this machine; the pinned `v0.23.0` binary the toolchain manager installed lives at `$HOME/.cache/quarry/tools/go/v0.23.0/gopls`. Every `//go:build lsp` test in this repo skips via `exec.LookPath("gopls")`, so a live-tier run that is meant to actually exercise gopls must prepend that directory to `PATH`. The batch verify commands that need it do so explicitly.
- **Rationale:** Without this the live tier silently skips and reports success, which is exactly the "silence looks like a pass" failure this task exists to remove.
- **Applies to:** lsp-implementation-and-spike, live-tier-tests

### Decision: done-gate-stays-null

- **Decision:** `pipeline.done_gate` in `mill-config.yaml` is left at `null` and no batch edits that file.
- **Rationale:** The overview's module-wide `verify:` already runs `go vet` over the whole module in both the default and the `lsp` tag views at every batch boundary, and the per-batch verify commands between them cover every package this plan touches (`lsp`, `registry`, `daemon`, `query`, `quarry`, `cli`) — that is the whole tree. No lint command is defined for this repo either: `golangci-lint` is not installed on this machine, so defaulting `done_gate` to it would make every future task in the hub depend on a binary that is absent.
- **Applies to:** all batches

### Decision: no-envelope-or-exit-code-changes

- **Decision:** The four verbs' JSON envelope shape and exit-code contract are unchanged. `assert-no-callers` keeps exit 0 / 1 / 2 and the `violation` + `callers` fields exactly as `internal/cli/cli.go`'s package comment documents them.
- **Rationale:** Agent callers key on these. This task changes which references are reported, never how they are reported.
- **Applies to:** cli-surface, live-tier-tests, docs-and-followups

### Decision: card-scoped-commits

- **Decision:** Every card that changes a file produces its own commit with the message its `Commit:` field names, and no card's diff is folded into another card's commit. The one exception is a card that changes no file at all: it carries `Commit: none` and makes no commit, because there is nothing to commit. Batch 7's card 29, which only files GitHub issues, is the sole such card in this plan.
- **Rationale:** The repo's existing one-commit-per-card convention, and the harness's no-amend rule makes any cross-card squash instruction unimplementable. The file-less exception is stated here rather than left implicit so the decision and card 29 do not read as contradicting each other.
- **Applies to:** all batches

## All Files Touched

_Full union of every `Creates:` / `Edits:` / `Moves:` **target** path across every batch, sorted alphabetically (Move **source** paths are excluded — they disappear, like `Deletes:` tokens).
Cards are the source of truth;
this section is the input `_plan_validate.py`'s `all-files-touched-mismatch` check cross-references against the derived union of every card's `Edits:`/`Creates:`/Move-target paths, to catch drift between the hand/agent-maintained list here and that derived union._

- `README.md`
- `docs/implementation-widening-spike.md`
- `docs/scout-multilang.md`
- `docs/servers.yaml.example`
- `internal/cli/assertnocallers_lsp_test.go`
- `internal/cli/cli.go`
- `internal/cli/cli_test.go`
- `internal/cli/paths.go`
- `internal/cli/paths_test.go`
- `internal/cli/resolve_test.go`
- `internal/quarryengine/daemon/daemontest/daemontest.go`
- `internal/quarryengine/daemon/doc.go`
- `internal/quarryengine/daemon/ensureserver.go`
- `internal/quarryengine/daemon/ensureserver_integration_test.go`
- `internal/quarryengine/daemon/ensureserver_test.go`
- `internal/quarryengine/daemon/supervised_integration_test.go`
- `internal/quarryengine/daemon/supervised_lsp_test.go`
- `internal/quarryengine/daemon/supervised_test.go`
- `internal/quarryengine/errors.go`
- `internal/quarryengine/lsp/lspclient.go`
- `internal/quarryengine/lsp/lspclient_test.go`
- `internal/quarryengine/query/buildtags_lsp_test.go`
- `internal/quarryengine/query/buildtags_test.go`
- `internal/quarryengine/query/callers.go`
- `internal/quarryengine/query/callers_test.go`
- `internal/quarryengine/query/implementation_spike_lsp_test.go`
- `internal/quarryengine/query/refs.go`
- `internal/quarryengine/query/refs_test.go`
- `internal/quarryengine/query/symbol.go`
- `internal/quarryengine/query/symbol_test.go`
- `internal/quarryengine/query/verify.go`
- `internal/quarryengine/query/verify_test.go`
- `internal/quarryengine/registry/buildtags.go`
- `internal/quarryengine/registry/buildtags_test.go`
- `internal/quarryengine/registry/initoptions.go`
- `internal/quarryengine/registry/initoptions_test.go`
- `internal/quarryengine/registry/load_test.go`
- `internal/quarryengine/registry/registry.go`
- `internal/quarryengine/registry/registry_test.go`
- `quarry/facade.go`
- `quarry/facade_test.go`
- `testdata/buildtagfixture/consumer/plain.go`
- `testdata/buildtagfixture/consumer/tagged.go`
- `testdata/buildtagfixture/go.mod`
- `testdata/buildtagfixture/lib/lib.go`
- `testdata/clockfixture/builder/poll.go`
- `testdata/clockfixture/go.mod`
- `testdata/clockfixture/runner/tick.go`
- `testdata/clockfixture/sched/wait.go`
