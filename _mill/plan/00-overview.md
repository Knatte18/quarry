# Plan: Kick-start pack bench: pre-resolved glyph spans in the prompt (M4)

```yaml
task: 'Kick-start pack bench: pre-resolved glyph spans in the prompt (M4)'
slug: ladder-kickstart
approved: false
discussion_sha: c3dcea342e86b0be6ced2bc6e5a6e3fbd2bebff5
started: '20260905-113436'
parent: main
root: ""
verify: go build ./...
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: config-surface
    file: 01-config-surface.md
    depends-on: []
    verify: go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestLoadLadder|TestConfigIsControlAndGrantsTools'
  - number: 2
    name: control-sweep-and-card
    file: 02-control-sweep-and-card.md
    depends-on: [1]
    verify: go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestRenderPrompt|TestLoadTaskFile|TestLoadCardFile|TestMCPConfigDocument|TestCheck|TestE2E|TestPreMatrix'
  - number: 3
    name: provenance
    file: 03-provenance.md
    depends-on: []
    verify: go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestReadProvenance|TestWriteProvenance|TestWriteReadProvenance|TestMergeProvenance'
  - number: 4
    name: pack-generation
    file: 04-pack-generation.md
    depends-on: [1, 3]
    verify: go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestRenderKickstartPack|TestPackBlock|TestPack_'
  - number: 5
    name: run-gate-and-subcommand
    file: 05-run-gate-and-subcommand.md
    depends-on: [2, 4]
    verify: go build ./bench/loomyard-eval/ladder/... && go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestE2E|TestPreMatrix'
  - number: 6
    name: benchmark-content
    file: 06-benchmark-content.md
    depends-on: [1, 2, 4]
    verify: go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestPreMatrix|TestLoadLadder_RealKickstartFile'
  - number: 7
    name: matrix-and-writeup
    file: 07-matrix-and-writeup.md
    depends-on: [5, 6]
    verify: go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestPreMatrix'
```

## Shared Decisions

### Decision: go-only-no-python

- **Decision:** every artefact in this task is Go, YAML, JSON or Markdown. No Python is introduced
  anywhere — not in the harness, not in an analysis step, not in a helper script. The Mann–Whitney
  arithmetic in `conclusion.md` is written out by hand, exactly as
  `results/2026-09-05-ladder-d/conclusion.md` does it.
- **Rationale:** the repository's own `CLAUDE.md` states "This is a Go repo. Do not introduce
  Python", and discussion.md's Constraints section repeats it. No other consumer needs a statistics
  implementation, so adding one would be new surface for a single write-up.
- **Applies to:** all batches

### Decision: control-and-grants-tools-are-two-predicates

- **Decision:** after batch 1, `Config.IsControl()` means *this cell is its ladder letter's
  comparison baseline* and `Config.GrantsTools()` means *this cell has an MCP server attached*.
  Every branch in the package is classified by asking which of the two it depends on. A branch about
  building, attaching, connecting to or allow-listing a server reads `GrantsTools()`; a branch about
  the comparison baseline, the blinding obligation or the summary pairing reads `IsControl()`.
- **Rationale:** all three cells of ladder e grant no tools, so today's single predicate would call
  all three controls: `validate` would reject the file and `summarize` would find no rung to pair.
  Blinding must still cover all three, and the server must be built for none. The two meanings have
  coincided only by accident in `ladder-toc.yaml`.
- **Applies to:** control-sweep-and-card

### Decision: backwards-compatibility-is-a-tested-property

- **Decision:** every existing ladder file, results root and test fixture must behave exactly as it
  does today. `control:` is an optional pointer field that defaults to today's
  `len(Allowed) == 0`; `card:` defaults to empty and renders today's prompt byte for byte;
  `pack:`/`pack_targets:` default to absent and disable every new gate. Batch 1 and batch 2 each
  carry a regression test asserting this rather than leaving it as a claim.
- **Rationale:** `ladder-toc.yaml` and the three committed results roots are the comparison basis for
  every previous conclusion. A silent behaviour change in the shared machinery would invalidate them
  without anyone noticing.
- **Applies to:** all batches

### Decision: the-pack-gate-never-keys-on-vcs-state

- **Decision:** `run`'s pre-rep-1 verification compares exactly one thing — the sha256 of the pack
  cell's sentinel-delimited card block against `kickstart_pack.pack_sha256`. It never compares
  `quarry_commit` or `quarry_dirty`. Those two fields are recorded in the `kickstart_pack` block as a
  statement of pack-time state and are never a comparand.
- **Rationale:** `MergeProvenance` derives the top-level `QuarryCommit`/`QuarryDirty` from the latest
  invocation, so the two sides differ after any commit between `ladder pack` and `run` — and
  committing the generated card is exactly such a commit, and the intended workflow. `quarry_dirty`
  is additionally vacuously true on both sides the moment `ladder pack` writes a tracked card, and a
  gate that is always satisfied is not a gate.
- **Applies to:** pack-generation, run-gate-and-subcommand

### Decision: card-files-live-beside-tasks-not-inside-the-harness

- **Decision:** the three per-cell card files live in a new `bench/loomyard-eval/cards/` directory,
  as siblings of `bench/loomyard-eval/tasks/`, named `07-<cell-id>.md`. The ladder file's `card:`
  values are repository-relative paths into that directory, resolved the same way `task_file:` and
  `fasit:` already are.
- **Rationale:** a card is benchmark content, not harness code, and belongs with the task file and
  fasit it accompanies. Putting them under `bench/loomyard-eval/ladder/` would mix authored prompt
  text into the Go harness's own tree.
- **Applies to:** benchmark-content, matrix-and-writeup

### Decision: no-machine-path-in-any-tracked-file

- **Decision:** no file this plan creates or edits may carry an absolute filesystem path from this
  host. The target repository is reached only through `LADDER_LOOMYARD_REPO`, resolved by the harness
  from the process environment or from `.scratch/ladder.env`, which is untracked. `results/*/raw/`
  stays untracked via the existing `bench/loomyard-eval/ladder/.gitignore` entry.
- **Rationale:** the standing rule the provenance record's hashed fields already exist to satisfy;
  `TestMergeProvenance_NoAbsolutePathAnywhereInOutput` enforces the provenance half of it today.
- **Applies to:** all batches

### Decision: the-matrix-run-is-real-spend-and-is-its-own-card

- **Decision:** batch 7 separates the three irreversible operator-facing steps into their own cards:
  generating the pack, running the 3 × 10 matrix, and writing the conclusion. The matrix card runs
  30 measured `claude -p` invocations plus up to 30 scorer invocations against the live API, at
  `run_model: claude-sonnet-5` / `max_turns: 60` with an opus-high scorer.
- **Rationale:** this is real money and roughly an hour of wall time, and once rep 1 has run the
  glyph list and n are frozen for the whole root (no optional stopping, no post-hoc rep exclusion).
  Splitting the cards means the pack can be inspected, and the glyph substitution rule applied, before
  anything is spent. Flagging it here rather than burying it in a card is deliberate: if the operator
  wants to stop after batch 6 and run the matrix by hand, batch 7 is the clean seam to descope.
- **Applies to:** matrix-and-writeup

### Decision: predeclared-decision-rule-is-copied-not-reinterpreted

- **Decision:** the analysis rule is fixed before rep 1 and is transcribed verbatim from
  discussion.md's D9 into `conclusion.md`: primary comparison `e1-pack` vs `e0-names`, primary
  metrics `turns` and `cost_usd`, one-sided Mann–Whitney U with direction "e1 lower", n = 10 per arm,
  α = 0.05, critical U ≤ 27. `e2-files` is secondary and descriptive — medians and ranges only, no
  test. A negative answer is a publishable answer and the conclusion is written the same way in
  either direction.
- **Rationale:** the whole point of a predeclared rule is that it cannot be renegotiated after the
  numbers are visible. Copying it rather than restating it removes the opportunity.
- **Applies to:** matrix-and-writeup

### Decision: recall-and-precision-are-descriptive-only-in-this-matrix

- **Decision:** the correctness gate is `summary_matches: true`. `recall` and `precision` are
  recorded and reported per rep but are never compared across arms, and `conclusion.md` states this
  as a known property of the design rather than discovering it afterwards.
- **Rationale:** `e1-pack`'s card names the seven files verbatim in the prompt, so its
  `relevant_files` recall is inflated by construction. A cross-arm recall comparison would measure
  the prompt's own contents.
- **Applies to:** benchmark-content, matrix-and-writeup

### Decision: verify-commands-are-go-scoped-to-the-harness-package

- **Decision:** every batch's `verify:` is a `go test` against
  `./bench/loomyard-eval/ladder/internal/ladder/` with a `-run` pattern naming the tests that batch
  touches. One batch prefixes that with a scoped `go build ./bench/loomyard-eval/ladder/...` — the
  one that touches the command-line layer, which has no test file of its own, so the build is its
  only gate. The module-wide `verify:` in this file's frontmatter is `go build ./...`. None of them
  carries a `PYTHONPATH=` prefix.
- **Rationale:** this repository is not a Python project, so the `PYTHONPATH=` isolation reset has
  nothing to isolate; the native runner is `go test`. The `-run` scoping keeps a batch's gate to the
  tests it can actually break, and `go build ./...` at each batch boundary catches a cross-package
  compile break in non-test code at the batch that introduces it — it does not compile `_test.go`
  files, so a broken test-file call site is caught by the batch's own `go test` instead. `pipeline.done_gate` is already
  `go test ./... && golangci-lint run` in `mill-config.yaml`, and both halves were confirmed exit-0
  against this worktree's tip before planning, so the repo-wide regression net is already in place.
- **Applies to:** all batches

## All Files Touched

- `bench/loomyard-eval/cards/07-e0-names.md`
- `bench/loomyard-eval/cards/07-e1-pack.md`
- `bench/loomyard-eval/cards/07-e2-files.md`
- `bench/loomyard-eval/ladder/cmd/ladder/main.go`
- `bench/loomyard-eval/ladder/internal/ladder/config.go`
- `bench/loomyard-eval/ladder/internal/ladder/config_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/e2e_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/gates.go`
- `bench/loomyard-eval/ladder/internal/ladder/gates_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/live_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/mcp.go`
- `bench/loomyard-eval/ladder/internal/ladder/mcp_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/pack.go`
- `bench/loomyard-eval/ladder/internal/ladder/pack_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/prematrix_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/prompt.go`
- `bench/loomyard-eval/ladder/internal/ladder/prompt_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
- `bench/loomyard-eval/ladder/internal/ladder/provenance_test.go`
- `bench/loomyard-eval/ladder/internal/ladder/run.go`
- `bench/loomyard-eval/ladder/ladder-kickstart.yaml`
- `bench/loomyard-eval/ladder/results/2026-09-05-kickstart/conclusion.md`
- `bench/loomyard-eval/ladder/results/2026-09-05-kickstart/pack-resolve.json`
- `bench/loomyard-eval/ladder/results/2026-09-05-kickstart/provenance.json`
- `bench/loomyard-eval/ladder/results/2026-09-05-kickstart/summary.json`
- `bench/loomyard-eval/ladder/results/2026-09-05-kickstart/table.txt`
- `bench/loomyard-eval/tasks/07-fabric-merge-state-tracing.fasit.json`
- `bench/loomyard-eval/tasks/07-fabric-merge-state-tracing.md`
- `docs/roadmap.md`
