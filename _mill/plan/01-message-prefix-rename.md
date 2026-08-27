# Batch: message-prefix-rename

```yaml
task: Thin quarry/ facade over internal/quarryengine
batch: message-prefix-rename
number: 1
cards: 2
verify: go test ./... && go test -tags lsp -run "^$" ./...
depends-on: []
```

## Batch Scope

This batch removes the dead `scoutengine` name from every engine error and log message, entirely in place — no file moves, no package changes, no new files. It runs first so that batch 2's 34-file relocation carries a purely structural diff instead of mixing a string rename into a move. It delivers no interface for a later batch to consume; batch 2 simply inherits files whose message strings are already correct.

Batch-local decision that differs from the overview: card 2 renames the stale doc-comment reference `scoutengine.ErrServerNotFoundSentinel` to `quarry.ErrServerNotFoundSentinel` — the package-qualified name that is correct *at this point in the plan*, while `errors.go` still lives in `package quarry`. Batch 2 card 3 retargets that same comment to `quarryengine.ErrServerNotFoundSentinel` when the file moves. Two small edits, each accurate at its own moment, rather than one edit that names a package that does not exist yet.

## Cards

### Card 1: Rename `scoutengine: ` and `scout: ` message prefixes to `quarry: `

- **Context:**
  - `quarry/doc.go`
- **Edits:**
  - `quarry/daemonstate.go`
  - `quarry/detect.go`
  - `quarry/ensureserver.go`
  - `quarry/errors.go`
  - `quarry/load.go`
  - `quarry/lspclient.go`
  - `quarry/refs.go`
  - `quarry/registry.go`
  - `quarry/toolchain.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Replace the literal message prefix `scoutengine: ` with `quarry: ` in every `errors.New`, `fmt.Errorf`, `fmt.Sprintf`, `defaultLogHandler.Warn`, and `defaultLogHandler.Info` string literal across the nine listed files — 59 occurrences total, distributed as `ensureserver.go` 18, `errors.go` 12, `lspclient.go` 8, `daemonstate.go` 6, `toolchain.go` 5, `registry.go` 4, `detect.go` 2, `load.go` 2, `refs.go` 2. Additionally replace the single outlier prefix in `(*ErrServerSpawnTimeout).Error()` at `quarry/errors.go:160` — the string `"scout: gave up waiting for the supervised daemon for %q to become ready"` becomes `"quarry: gave up waiting for the supervised daemon for %q to become ready"` — so all six error types carry one consistent prefix. Change only the prefix token; leave every message body, format verb, and argument list byte-identical. Do not touch `quarry/errors.go:22`, which is a doc comment and is handled by card 2. Do not touch the historical comment at `quarry/refs_integration_test.go:55`, which describes Loomyard's package and stays as it is.
- **Commit:** `refactor(quarry): rename scoutengine/scout message prefixes to quarry`

### Card 2: Fix the stale doc-comment reference and the port-equivalence claim

- **Context:**
  - `quarry/registry.go`
- **Edits:**
  - `quarry/errors.go`
  - `docs/port-equivalence.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** In `quarry/errors.go`, the doc comment above `ErrServerNotFoundSentinel` (line 22) references `scoutengine.ErrServerNotFoundSentinel`; retarget it to `quarry.ErrServerNotFoundSentinel`, which is the package-qualified name that is correct while this file still declares `package quarry`. In `docs/port-equivalence.md`, correct the live claim around lines 105–106 that states quarry's own errors still carry a `"scoutengine: "` prefix, including the quoted sample JSON error string `{"error":"scoutengine: symbol \"QuarryNoSuchSymbolXYZ123\" not found ...`, so both describe the new `quarry: ` prefix. Leave every other `scoutengine` mention in `docs/` untouched — `docs/scout-multilang.md`, `docs/scout-vs-grep.md`, `docs/scout-agent-usage-findings.md`, and `port-equivalence.md`'s own provenance tables describe Loomyard's `internal/scoutengine` package as a past state and remain accurate.
- **Commit:** `docs(quarry): retarget stale scoutengine references to quarry`

## Batch Tests

`verify:` runs `go test ./... && go test -tags lsp -run "^$" ./...`. No test in the repository asserts on the literal `scoutengine: ` or `scout: ` prefix — a repository-wide grep before this task confirmed the only non-production hits are `quarry/refs_integration_test.go:55` (a historical comment) and the `docs/` files listed above — so the existing suite passing unchanged is the correctness bar for this batch. No new test is added: the acceptance check here is a grep, not an assertion, and it is stated in the discussion's Testing section (`grep -rnE 'scoutengine|"scout: ' --include='*.go' .` must return only `refs_integration_test.go:55` afterwards). The `-tags lsp` compile pass covers `refs_integration_test.go`, the one tagged file that mentions `scoutengine` at all.
