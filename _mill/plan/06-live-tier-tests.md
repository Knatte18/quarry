# Batch: live-tier-tests

```yaml
task: "Improve gopls query precision (build tags + scoping)"
batch: "live-tier-tests"
number: 6
cards: 3
verify: PATH="$HOME/.cache/quarry/tools/go/v0.23.0:$PATH" go test -tags lsp ./internal/quarryengine/query/ ./internal/cli/
depends-on: [1, 4, 5]
```

## Batch Scope

This batch delivers the two end-to-end tests that a fake server cannot stand in for, because both symptoms are gopls's own behaviour rather than this wrapper's wiring: a reference behind a `//go:build` constraint being invisible without `--build-tags`, and an interface-method query conflating structurally-identical interfaces across packages. Each runs on its own fixture module and is guarded by `exec.LookPath("gopls")`.

It is one batch because both tests are the same shape — spawn a real gopls against a `testdata/` fixture and assert on the resulting reference set — and because they are the last gate before the documentation batch describes behaviour as shipped. Batch-local decision: the interface-conflation test is written at the `internal/cli` level rather than in `query`, because the thing worth pinning is the whole `assert-no-callers` answer including `--no-verify`, and that composition lives in the CLI.

## Cards

### Card 23: build-tag visibility fixture module

- **Context:**
  - `go.mod`
  - `testdata/clockfixture/go.mod`
- **Edits:** none
- **Creates:**
  - `testdata/buildtagfixture/go.mod`
  - `testdata/buildtagfixture/lib/lib.go`
  - `testdata/buildtagfixture/consumer/plain.go`
  - `testdata/buildtagfixture/consumer/tagged.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Create a self-contained module `testdata/buildtagfixture/go.mod` declaring `module buildtagfixture` and a `go` directive matching the repo's own `go.mod`, with no dependencies beyond the standard library.
  - `testdata/buildtagfixture/lib/lib.go` is package `lib` and declares one exported function, the symbol under test. Give it a name unlikely to collide with anything gopls might also match in the workspace.
  - `testdata/buildtagfixture/consumer/plain.go` is package `consumer` with no build constraint and calls that function exactly once.
  - `testdata/buildtagfixture/consumer/tagged.go` is package `consumer`, carries a `//go:build sometag` constraint as its first line followed by a blank line, and calls the same function exactly once from a differently-named function so the two call sites are distinguishable.
  - The tagged file must compile under `sometag` and be excluded without it, which is the whole point of the fixture. Keep both files free of anything else gopls would have to resolve.
- **Commit:** `test(fixture): add a build-tag-constrained fixture module for the live tier`

### Card 24: build-tag visibility live test

- **Context:**
  - `internal/quarryengine/query/refs_integration_test.go`
  - `internal/quarryengine/query/refs.go`
  - `internal/quarryengine/registry/registry.go`
  - `testdata/buildtagfixture/lib/lib.go`
  - `testdata/buildtagfixture/consumer/plain.go`
  - `testdata/buildtagfixture/consumer/tagged.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/query/buildtags_lsp_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Create a `//go:build lsp`-tagged test file in package `query` holding one test that skips via `exec.LookPath("gopls")`, resolves the fixture module directory from the repo root exactly as `internal/quarryengine/query/refs_integration_test.go` resolves its own paths, and points a fresh state directory at `t.TempDir()` so the run never touches a real daemon state directory or an already-running daemon.
  - Call `References` twice against the exported symbol declared in `testdata/buildtagfixture/lib/lib.go`, resolved as an explicit position located by scanning the fixture source rather than by hard-coded line numbers: once with `BuildTags` empty and once with `BuildTags` set to the fixture's tag.
  - Assert that the untagged call reports the call site in `testdata/buildtagfixture/consumer/plain.go` and does not report the one in `testdata/buildtagfixture/consumer/tagged.go`, and that the tagged call reports both. That difference is the entire defect issue #2 describes, and it is not reproducible without a real gopls.
  - Use two different state directories for the two calls, or let the tag-keying produce them, so the second call cannot be answered from the first call's warm daemon view. State in a comment which of the two you relied on.
  - Assert nothing about latency or daemon counts. A daemon-identity assertion is only worth adding here if it can be made deterministic without racing the daemon lifecycle; the hermetic path-derivation tests in batch 5 are the contract for that, so leave it out rather than write a flaky test.
- **Commit:** `test(query): assert build-tag-constrained references are visible only with --build-tags`

### Card 25: interface-conflation live test at the CLI level

- **Context:**
  - `internal/cli/cli.go`
  - `internal/cli/exec.go`
  - `internal/cli/cli_test.go`
  - `docs/implementation-widening-spike.md`
  - `testdata/clockfixture/builder/poll.go`
  - `testdata/clockfixture/runner/tick.go`
  - `testdata/clockfixture/sched/wait.go`
- **Edits:** none
- **Creates:**
  - `internal/cli/assertnocallers_lsp_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Create a `//go:build lsp`-tagged test file in package `cli` holding one test that skips via `exec.LookPath("gopls")` and drives the command tree the way the existing tests in `internal/cli/cli_test.go` already do, capturing the JSON envelope from the writer they pass in.
  - Run `assert-no-callers` against the `Now` method of the `clock` interface declared in `testdata/clockfixture/builder/poll.go`, passing `--target-dir` at the clock fixture module and `--state-dir` at a `t.TempDir()`. Locate the query position by scanning the fixture source rather than hard-coding line numbers.
  - Assert that the reported `callers` list contains only call sites inside the fixture's `builder` package, and none from `testdata/clockfixture/runner/tick.go` or `testdata/clockfixture/sched/wait.go` — the unrelated structurally-identical interfaces that produce issue #1's false violations.
  - Assert the exact reported count against the post-fix figure recorded in `docs/implementation-widening-spike.md`, not against issue #1's pre-fix "31 → 2" measurement, which was taken on a different repository. If the recorded figure and the observed one disagree, fix the code or re-run the spike; do not adjust the assertion to whatever the run produced.
  - Run the same query again with `--no-verify` and assert the result is strictly larger and does include at least one call site from `runner` or `sched`, pinning both the fix and its escape hatch in one fixture.
  - Assert the exit code in both runs: the envelope carries `violation: true` and the process exit is 1 whenever the `callers` list is non-empty, unchanged from today's contract.
- **Commit:** `test(cli): pin the interface-conflation fix and the --no-verify escape hatch against live gopls`

## Batch Tests

`verify:` runs the `lsp`-tagged tier for the two packages this batch adds tests to, with the pinned `gopls v0.23.0` from the toolchain cache prepended to `PATH`. That prefix is load-bearing rather than cosmetic: `gopls` is not on `$PATH` on this machine, and both tests skip on `exec.LookPath` failure, so without it the batch would report success having exercised nothing. Card 23 adds no runnable Go surface to the main module — the fixture lives under `testdata/`, which the Go tool ignores. The hermetic tier is untouched by this batch and is covered at the batch boundary by the overview's module-wide `go vet` in both tag views.
