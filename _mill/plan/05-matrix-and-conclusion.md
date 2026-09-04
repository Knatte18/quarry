# Batch: matrix-and-conclusion

```yaml
task: "Ladder breadth (M1)"
batch: "matrix-and-conclusion"
number: 5
cards: 3
verify: go test ./bench/loomyard-eval/ladder/internal/ladder/
depends-on: [1, 4]
```

## Batch Scope

This batch spends the task's real budget and writes the answer. It runs the six-cell matrix at
`reps: 5` into one results root, then writes the conclusion that reads it. It is one batch because
the conclusion quotes `summary.json` and `table.txt` from the root the same batch produced — the two
are inseparable, and a conclusion written from a different batch's root would be quoting numbers it
did not watch being produced.

It depends on batch 1 because M2 must be committed before the matrix starts: a harness change cannot
land mid-matrix, and the matrix's own non-control cells completing end to end are what prove it. It
depends on batch 4 because every offline gate must be green before a real call is made.

The external interface batch 6 consumes is the conclusion's per-shape verdict and its aggregate
statement — whether toc separates anywhere in this root, or nowhere — which batch 6 propagates into
`docs/roadmap.md` and `HANDOFF.md` §3.

Batch-local decisions that differ from the overview's `## Shared Decisions`:

- **The results root's date is pinned in this plan as `2026-09-04-breadth`.** If the invocation
  actually begins on a different calendar date, use that date instead — the root name is a label, and
  `provenance.json` carries the real timestamps. The substitution scope is **not** limited to this
  batch: see the overview's `results-root-date-substitution` decision, which extends it to batch 6's
  cards 17 and 18 and to the overview's own `## All Files Touched`, since those two cards write the
  path and the root name into two tracked documents.
- **One root, any number of invocations.** "One results root" is the invariant; "one invocation" is
  not. Re-invoking the same `run` command over the same `--results` root is how the harness resumes
  and is expected. `provenance.json`'s `invocations` list grows one entry per invocation and the
  conclusion reports how many there were. What is forbidden is spreading this task's cells across two
  roots, or editing the harness or the stimulus between invocations.
- **Resumes are capped at two.** Past that, stop and report rather than re-invoking again.

## Cards

### Card 14: pre-matrix preflight

- **Context:**
  - `bench/loomyard-eval/ladder/ladder-toc.yaml`
  - `bench/loomyard-eval/ladder/internal/ladder/prematrix_test.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
  - `bench/loomyard-eval/ladder/.gitignore`
  - `.scratch/ladder.env`
  - `HANDOFF.md`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Confirm, and report to the operator, every precondition below before card 15 spends a single real
  call. This card changes no file; its whole job is that no invocation starts against a tree that
  cannot produce a clean record.

  - `go test ./... && golangci-lint run` from the repository root exits 0. This is the repository's
    own done gate and it must be green here, not only at handoff.
  - `go test ./bench/loomyard-eval/ladder/internal/ladder/` is green, including batch 4's
    `TestPreMatrix_ControlPromptsAreBlind` and `TestPreMatrix_NewFasitsAreWellFormed`. A red gate
    here is a stop, not a warning.
  - `git status --porcelain` from the repository root is clean apart from the one tolerated
    carve-out: the narrow `_mill/briefs/<currently-executing-batch>*.md` file the orchestrator writes
    before any card can commit it. A dirty path outside that carve-out is committed or reverted
    first — `CollectInvocation` records `quarry_dirty` and `quarry_dirty_files` into
    `provenance.json`, and a dirty tree at invocation time makes that record dishonest. Read
    `provenance.go` to see exactly what is captured.
  - `.scratch/ladder.env` exists and names a `LADDER_LOOMYARD_REPO` that resolves. It is gitignored
    and per machine; never commit it and never hardcode its value into a tracked file.
  - The MCP server actually connects. Run the `--setting-sources ""` probe `HANDOFF.md` §4
    documents, and confirm it before invoking rather than after. This matters more than it looks:
    the harness's `connectFailures == attempts` whole-run abort is a fresh-root protection only —
    `connectFailures` is loop-local while `attempts` is cumulative — so on a resumed root an unfixed
    server fault no longer stops the invocation and quietly burns one real call per repetition. On a
    resume, the cost guard is this probe and nothing else.

  A `--reps 1` smoke run of `c1-toc-dir` and `d1-toc-dir` is **not** taken on this task's own
  authority: it costs real budget and the cost discipline reserves any widening for the operator. It
  is available with the operator's explicit word, and it is the only thing that would surface the one
  residual risk the offline layer cannot catch — a prompt that is blind and loads cleanly but asks
  the wrong thing. If it is authorised, it goes into a throwaway results root **outside**
  `bench/loomyard-eval/ladder/results/` and that root is deleted afterwards; a measured root must
  contain only the matrix it reports.
- **Commit:** none

### Card 15: run the six-cell matrix into one results root

- **Context:**
  - `bench/loomyard-eval/ladder/ladder-toc.yaml`
  - `bench/loomyard-eval/ladder/cmd/ladder/main.go`
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
  - `bench/loomyard-eval/ladder/.gitignore`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/results/2026-09-04-breadth/provenance.json`
  - `bench/loomyard-eval/ladder/results/2026-09-04-breadth/summary.json`
  - `bench/loomyard-eval/ladder/results/2026-09-04-breadth/table.txt`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Run the matrix from the repository root:

  ```
  go run ./bench/loomyard-eval/ladder/cmd/ladder run \
    --config bench/loomyard-eval/ladder/ladder-toc.yaml \
    --results bench/loomyard-eval/ladder/results/2026-09-04-breadth \
    --cells b0-none,b8-toc-dir,c0-none,c1-toc-dir,d0-none,d1-toc-dir
  ```

  All six cells in one `--cells` list, into one root, at the ladder file's own `reps: 5`. Every run
  parameter comes from the ladder file and none is overridden: `run_model` `claude-sonnet-5`,
  `run_effort` `medium`, `max_turns` 60, scorer `claude-opus-5` at `high`, `quarry_tools` `[toc]`,
  server built from `./cmd/quarry-mcp`. Do not pass `--reps`. Ladder a is not re-run; task 01 was
  measured by T7 on this host and harness and re-measuring it here buys nothing and costs ten real
  calls.

  This is a long-running invocation making thirty real `claude -p` calls plus scorer calls. Run it in
  the background and wait for it rather than polling.

  No separate report step is needed on a clean run: `cmd/ladder`'s `runCommand` already calls
  `summarizeAndReport(*resultsRoot)` after `ladder.Run` returns, so `summary.json` and `table.txt`
  exist alongside `provenance.json` when the invocation exits. Run the `report` subcommand only to
  **re-derive** them — after a resume, or after an invocation that was killed before it reached its
  own report call, where a stale or missing `summary.json` would otherwise sit beside the newly
  written repetitions. Read `main.go` for the exact invocation before using it.

  **Shortfalls.** The done-when is every measured cell a real MCP cell completing end to end, `5/5`
  complete non-blinding-failed repetitions per cell, `unscored_count: 0`, and gate 1
  (`granted_tool_used`) reported per rung cell. When any of those falls short, these four remedies
  are the whole set — record which ones fired, for card 16's coverage section:

  1. **A repetition exhausts `MaxAttempts`** and its cell is recorded incomplete. Read the
     `invalid_reason.txt` files batch 1 now writes, fix the named cause out of band, and re-invoke
     over the same root — a repetition that is not `RepIsComplete` is re-attempted. Resume buys **one
     attempt, not a fresh three**: `run.go` compares `InvalidateRep`'s return value, the cumulative
     `.invalid-<n>` suffix, against `MaxAttempts`, so a repetition already carrying three invalid
     directories gets exactly one more attempt per invocation. Verify the fix out of band before
     re-invoking rather than using the matrix as the test. Cap this at **two resume invocations**;
     past that, stop and report.
  2. **The whole invocation aborts** on `connectFailures == attempts`. That is an environment or
     configuration fault by construction, not data: fix it, then re-invoke over the same root.
     Repetitions already complete are not re-run. Note that this abort is a fresh-root protection
     only, which is why card 14's server probe is the real guard on a resume.
  3. **A repetition is complete but unscored** (`ScoreSkipReason: "scorer_failed"`). `RepIsComplete`
     returns true for it, so a re-invocation will never re-attempt it. The only route back to
     `unscored_count: 0` is to delete that one repetition directory under the untracked `raw/` tree
     and re-invoke, which re-runs and re-scores it. That deletion is permitted, is the only
     hand-touch of a measured root this task allows, and must be recorded in card 16's coverage
     section — which repetition, why, and that it was re-run. If the same repetition fails scoring a
     second time, stop deleting: accept it, report `unscored_count` non-zero, and read that cell's
     recall and precision at the reduced `n` while its cost metrics stay at full `n`.
  4. **A control repetition fails the rendered-prompt blinding gate.** Batch 4's offline assertion is
     what catches this, before the matrix, never during it. If one fires anyway mid-matrix, stop the
     invocation immediately: the prompt is measured stimulus and cannot be edited with the matrix in
     flight, so the root is abandoned with an `ABANDONED.md` and restarted after the prompt is fixed.

  **Committing the root.** Commit `provenance.json`, `summary.json` and `table.txt`. Nothing under
  `raw/` is committed — `bench/loomyard-eval/ladder/.gitignore`'s `results/*/raw/` entry stays exactly
  as it is and is not modified. Committing the partial root between resume invocations is deliberately
  not required: a half-written `summary.json` is not a record worth committing, and the root is
  committed once, here, when the matrix is done.

  On a resumed invocation, `git status --porcelain` will list the root's own `provenance.json`,
  `summary.json` and `table.txt` as untracked, because `.gitignore` covers `raw/` but not those.
  Those entries are benign — every one of them is inside the root being written. A dirty path
  **outside** the results root on a resume is not covered by that tolerance and must be committed or
  reverted before re-invoking.
- **Commit:** `bench(ladder): run the six-cell breadth matrix at reps 5`

### Card 16: write the conclusion

- **Context:**
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/conclusion.md`
  - `bench/loomyard-eval/ladder/results/2026-09-04-breadth/summary.json`
  - `bench/loomyard-eval/ladder/results/2026-09-04-breadth/table.txt`
  - `bench/loomyard-eval/ladder/results/2026-09-04-breadth/provenance.json`
  - `bench/loomyard-eval/ladder/ladder-toc.yaml`
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
  - `bench/loomyard-eval/ladder/internal/ladder/metrics.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/results/2026-09-04-breadth/conclusion.md`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Write `conclusion.md` following `results/2026-09-04-toc/conclusion.md`'s structure, which is the
  structural model precisely because it quotes rather than re-derives and reports correctness with
  its own `n`. Read it before writing. Sections, in order:

  - **Numbers**, one table per ladder (b, c, d), each row a metric with median and [min–max], quoted
    **verbatim** from `summary.json` and `table.txt` and never re-derived by hand. Re-deriving is what
    made a conclusion unauditable; quoting is what made T7's auditable.
  - **A per-shape verdict**, one section per ladder, applying the separation rule stated in the
    overview's `## Shared Decisions`: separating only when a cost metric's comparison entry is
    `separated: true`, **and** that metric's median moves in the predicted direction (rung cheaper
    than control), **and** neither recall nor precision degrades. Anything short of all three is no
    separation, with the medians and ranges quoted so a reader can see how close it came. Then state
    the aggregate plainly: whether toc separates anywhere in this root, or nowhere.
  - **Coverage: gates, invalidations, drift, provenance.** Report per rung cell whether gate 1
    (`granted_tool_used`) fired — a flat rung that never called its tool measured the tool's prompt
    cost, not the tool, and the conclusion must say which it was. Report the invalidation record from
    the `invalid_reason.txt` files batch 1 writes: how many attempts were invalidated, per cell, with
    their `cause` values and counts. Quote `cause` values and counts only; **never** quote a `detail`
    string verbatim. The reason files live in the untracked `raw/` tree where a stray path would be
    harmless, but this file is tracked and no tracked file in this repository carries a machine path.
    State how many invocations `provenance.json`'s `invocations` list records and why there was more
    than one, if there was. Record any permitted repetition deletion from card 15's remedy 3 —
    which repetition, why, and that it was re-run.
  - **What this settles.** Name the T8 unpark implication in the roadmap's own terms: either a
    measured win that re-establishes the surface's value to an agent, or not. Do **not** make the
    unpark decision — that is the operator's, and the roadmap says so.

  The M2 invalidation record is reported here but is **not** a measured metric and enters no
  comparison. It closes the loop on why M2 was folded into this task — the reader of this root gets
  the diagnostic T7's reader did not have — without pretending an infrastructure artifact is a
  benchmark result.

  If any cell could not reach `5/5` after the capped resumes, report it as **incomplete with its
  `cause` values and their counts named**, do not present its comparison as a result, and name that
  task shape as **unmeasured** rather than as flat. A shape reported flat on `3/5` would be the worst
  possible outcome of this task.

  This file is tracked: it carries no absolute filesystem path from this machine, and the Loomyard
  checkout is referred to only through `LADDER_LOOMYARD_REPO`.
- **Commit:** `bench(ladder): write the breadth matrix conclusion`

## Batch Tests

The matrix is not a test and has no assertions. Its acceptance is the task's own done-when, stated in
card 15: every measured cell a real MCP cell completing end to end, `5/5` complete
non-blinding-failed repetitions per cell, `unscored_count: 0`, and gate 1 reported per rung cell so
the conclusion can say whether a flat rung actually called its tool. When any of those falls short,
card 15's four remedies say what happens instead — resume over the same root, and report the
shortfall rather than paper over it.

`verify: go test ./bench/loomyard-eval/ladder/internal/ladder/` therefore does not test this batch's
own output. It is a tripwire: this batch runs the harness against real repositories for hours and
writes into the results tree, and the package suite confirms the harness code is still exactly as
batches 1 and 4 left it and that nothing in the run perturbed the tracked stimulus. Scoping it to the
one package this task touches keeps it cheap; the repository-wide `go test ./... && golangci-lint run`
runs as the done gate at handoff, and card 14 also runs it as a precondition before the matrix starts.

The offline gate that actually protects this batch already ran, in batch 4, and card 14 re-confirms
it green as its first precondition.
