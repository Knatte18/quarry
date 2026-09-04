# Batch: matrix-run

```yaml
task: "Ladder, toc rerun (T7)"
batch: "matrix-run"
number: 3
cards: 2
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [2]
```

## Batch Scope

This batch spends the task's whole API budget and produces its three machine artifacts. Card 6 is
the gate — the offline suite, the clean tree, the environment preconditions and the harness's own
guarded live test, in that order — and card 7 drives the matrix to a terminal state and commits what
it produced. The two are one batch because they share the same context (the harness's failure
taxonomy, its gates, its resume contract) and because nothing may come between them: the clean-tree
check card 6 ends on is only meaningful if the next thing that happens is the matrix.

The batch is also the only place a harness defect may be fixed. If one blocks the run, it is fixed
under `bench/loomyard-eval/ladder/` with a failing table test written first, committed, and the
matrix restarts in a fresh `-r2` root per `## Shared Decisions`. A defect in the code under test
stops the run instead and is recorded by the next batch.

The interface batch 3 consumes: a results root holding `summary.json`, `provenance.json` and
`table.txt`, committed, plus the run log under `.scratch/` carrying the per-invocation server-hash
readings and the invocation count.

## Cards

### Card 6: Pre-matrix gates and the guarded live test

- **Context:**
  - `bench/loomyard-eval/ladder/ladder-toc.yaml`
  - `bench/loomyard-eval/ladder/internal/ladder/live_test.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `bench/loomyard-eval/ladder/internal/ladder/mcp.go`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** A zero-diff gate card. Run each step from the repository root and stop the whole
  batch on the first failure, reporting what failed.
  (1) **The offline suite.** `go test ./... && golangci-lint run`, with `LADDER_LIVE_TEST` unset so
  the guarded live test skips. The tree is green today, so any failure is something this task
  introduced.
  (2) **The clean tree.** `git status --porcelain` must be empty. A non-empty tree means the matrix
  would record `quarry_dirty` true and describe something that is not in git.
  (3) **The environment preconditions**, each read and reported, none of them written into a
  committed file: `LADDER_LOOMYARD_REPO` is either unset in the invoking environment, or set to a
  value matching what the ladder env file under .scratch/ holds — `ResolveLoomyardRepo` reads the
  process environment **first** and only falls back to that file, so an inherited stale value would
  silently win and never be noticed; the file resolves the Loomyard checkout; that checkout's HEAD
  is at `72c23d9` — the pin the golden test hard-fails on, and a different thing from the
  `pinned_sha` the ladder file names, which is the task worktree commit the harness checks out per
  cell — and the ladder file's own
  `pinned_sha` is a commit reachable in it; `LADDER_WORKTREE_ROOT` is either unset — so
  `ResolveWorktreeRoot` falls back to the cache directory — or set to a path that is not the quarry
  repository root, not under it, and does not contain the substring `quarry`, which
  `ResolveWorktreeRoot` refuses; and no stale `.ladder.lock` sits under the resolved ladder worktree
  root. Never copy the resolved Loomyard path or the resolved worktree root into any file this task
  commits — the repository's rule is that no tracked file carries a machine path, which is why the
  provenance record stores hashes.
  (4) **The guarded live test**, once:
  `env -u CLAUDECODE -u CLAUDE_CODE_ENTRYPOINT LADDER_LIVE_TEST=1 go test ./bench/loomyard-eval/ladder/internal/ladder -run TestLive -v -timeout 20m`.
  Budget it as an eleventh measured repetition, not as a unit test:
  `TestLive_FreshWorktreeGrantsExactlyBuiltins` drives `invokeMeasuredProcess` at the ladder file's
  own run model, run effort and 60-turn ceiling, so it costs one full control repetition's API spend
  and wall-clock. The explicit `-timeout 20m` is there because the default 10-minute limit sits
  uncomfortably close to a 60-turn run's worst case and a timeout kill would read as a seam failure
  when it is not. The same `env -u` prefix the matrix uses is mandatory here: the test exists to make
  a claim about the `claude -p` seam under the conditions the matrix runs in, and running it under
  the very markers the matrix strips would test something else. Because this test is slow and
  expensive, run it in the background with its output tee'd to the fixed path
  .scratch/ladder-live-test.log and poll that file, rather than blocking a foreground call on it. A
  failure blocks the matrix — do not proceed to card 7.
  (5) **No baseline server hash is taken here, and any binary already present is stale.** `run.go`
  builds `<ladder-worktree-root>/bin/<server-name>` through `BuildServer` *inside* each invocation,
  and the guarded live test passes an empty server-binary path and never builds it, so at this point
  the file is either absent or a leftover from an unrelated run. Report whether one is present and
  say plainly that it is stale, then leave it alone: the out-of-band readings the write-up
  transcribes are taken by card 7 immediately **after** each invocation, when the binary the
  invocation actually measured is the one on disk. Make no file change in this card.
- **Commit:** none

### Card 7: Run the toc matrix and commit its machine artifacts

- **Context:**
  - `bench/loomyard-eval/ladder/ladder-toc.yaml`
  - `bench/loomyard-eval/ladder/cmd/ladder/main.go`
  - `bench/loomyard-eval/ladder/internal/ladder/mcp.go`
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/probe.md`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/summary.json`
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/provenance.json`
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/table.txt`
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/ABANDONED.md`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Drive the matrix to a terminal state, then commit the three artifacts it wrote.
  **The invocation**, run from the repository root, detached, with stdout and stderr tee'd — appended,
  never truncated, so all three invocations share one file — to the fixed path
  .scratch/ladder-toc-run.log, and polled rather than blocked on:

  ```
  env -u CLAUDECODE -u CLAUDE_CODE_ENTRYPOINT \
    go run ./bench/loomyard-eval/ladder/cmd/ladder run \
      --config bench/loomyard-eval/ladder/ladder-toc.yaml \
      --results bench/loomyard-eval/ladder/results/2026-09-04-toc \
      --cells a0-none,a2-toc-dir
  ```

  No `--reps` override, so the ladder file's own `reps: 5` applies; no other flag is passed, so the
  file's run model, run effort, 60-turn ceiling, scorer model and scorer effort all come from the
  file. Ten measured repetitions plus ten scorer invocations is the whole matrix.
  **After every invocation**, including the last: take `sha256sum` of the built server binary at
  `<ladder-worktree-root>/bin/<server-name>` and append the reading to the run log. The reading comes
  after because the harness builds that binary inside the invocation — before the first one the file
  is absent or stale, and a before-only rule would leave the final invocation's binary unhashed
  entirely. Two readings that differ mean what a differing commit means: the root mixed two versions
  of the code under test and is void.
  **Before every re-invocation**, confirm no `.ladder.lock` remains under the resolved ladder
  worktree root. `AcquireRunLock` refuses outright when the file exists, so a killed background
  invocation makes the next one fail in under a second having measured nothing. Clear a lock whose
  recorded pid is dead, then re-invoke. **A lock-refused invocation does not consume an arm of the
  three-invocation ceiling** — the ceiling exists to bound API spend, and this failure spends none.
  Also confirm `git status --porcelain` lists nothing outside the results
  root. It will not be empty: the first invocation's machine artifacts sit untracked under a tracked
  directory until this card commits them, so `CollectInvocation` records `quarry_dirty` true for
  invocations 2 and 3 by construction. That carve-out is accepted and reported rather than worked
  around — committing between invocations would edit the repository mid-matrix. An entry outside the
  results root is a different matter and voids the root: stop and report it.
  **The termination rule** is the driver's own, not the harness's, and it has three arms. Stop when
  every selected cell reports 5 complete repetitions; or when three invocations of the command have
  been made, the third being the last; or when every still-missing repetition is attempt-exhausted.
  Before each re-invocation, list `<root>/raw/<cell>/<rep>.invalid-*` for every missing repetition: a
  repetition with three or more such directories will never complete, because `InvalidateRep` derives
  its counter from the directories already on disk, so the next attempt trips the ceiling immediately
  after spending a fresh measured call. When every missing repetition is exhausted, do not re-invoke
  at all. A non-zero exit code alone is never a reason to re-run: `runCommand` exits non-zero
  whenever the summary carries an incomplete or invalid cell, and neither the exit code nor the
  summary distinguishes "never attempted" from "attempt-exhausted".
  **A blinding failure takes a different path and must be handled by hand.** `writeCompleteState`
  with the blinding flag set writes the repetition complete-with-the-flag rather than invalidating
  it, so no counter increments and an unattended loop would re-attempt it forever. Re-attempt such a
  repetition **exactly once**, and only after diagnosing and naming the cause; a second failure of
  the same repetition stops the matrix. Per fatal check: check (a), the MCP prefix in a control
  transcript, and check (b), the quarry repository root path in a control transcript, are both
  non-transient — inspect the cell's configuration and the rendered prompt, and stop until the cause
  is explained. Check (d), `CheckRenderedControlPrompt`, runs pre-dispatch and is deterministic, so a
  re-run can never clear it: **zero** re-attempts, stop immediately. `ScanMemoryPaths` aborts the
  whole invocation rather than one repetition, and the memory it walks is the measured session's own,
  which runs with its working directory in the pinned Loomyard worktree — so it is Loomyard's project
  memory that trips it; one re-attempt after the offending file is dealt with, then stop. Gate 2
  check (c), the bare token `quarry` in a control transcript, is an observation and never fatal —
  expect it to fire and carry it to the write-up as information.
  **Two outcomes are accepted and never retried:** a scorer failure, which `dispatchScorer` has
  already retried alone up to the harness ceiling before writing the repetition complete with
  `scored: false` and `score_skip_reason` set to `scorer_failed`, counted in `UnscoredCount` and
  dropped from recall and precision only; and a max-turns completion, which is complete by design and
  counted in `MaxTurnsCount`. Do not re-run either one's measured process.
  **The restart path, and the one conditional artifact this card owns.** If a harness defect blocks
  the run, fix it under the ladder tree with a failing table test written first, commit that, and
  restart the matrix in a fresh `-r2` root — never against this one, per `## Shared Decisions`. This
  card is then also the card that writes `ABANDONED.md` into the abandoned root, naming the fix, the
  date and the successor root, following the precedent of the V1 tree's own abandoned compact root.
  It is the only card that knows all three facts. That file is listed in `Creates:` as a
  **conditional** artifact: on the ordinary path, where no restart happens, it is never written and
  the card creates three files rather than four. The `Creates:` paths above name this root; on the
  restart path the three machine artifacts land in the `-r2` root instead, per the substitution rule
  in `## Shared Decisions`, and `ABANDONED.md` is the one file that stays here. Also copy `probe.md`
  into the fresh root unchanged, so that root satisfies the five-file list on its own — the operator
  report is about the server, not about which root measured it, and re-deriving it is not possible.
  **If the last invocation exited on an error rather than on a summary, re-derive the artifacts
  before committing.** `runCommand` returns as soon as `ladder.Run` reports an error and never
  reaches `summarizeAndReport`, so `summary.json` and `table.txt` are then missing outright or still
  describe an earlier invocation — while `provenance.json`, written as the run proceeds, is current.
  That path is reachable: a resumed root's `ScanMemoryPaths` abort, a `BuildServer` failure, a
  refused run lock, a reps mismatch, or any per-repetition error. Whenever the final invocation ends
  that way, run `go run ./bench/loomyard-eval/ladder/cmd/ladder report --results
  bench/loomyard-eval/ladder/results/2026-09-04-toc` before the commit. The `report` subcommand needs
  only the results root — it re-derives the summary and the table from the raw tree and the existing
  provenance record, running and scoring nothing — so it costs no API budget and leaves the
  measurement untouched. Record in the run log that it was run and why. Committing a `summary.json`
  from an earlier invocation, or committing three files when one was never written, would make every
  number the write-up quotes describe a run that did not happen.
  **When the run terminates**, record in the run log: the number of invocations made, every
  server-hash reading, the final incomplete and invalid lists, the per-invocation `quarry_dirty`
  file lists, and the per-cell repetition counts. Then `git add` exactly the artifacts named in
  `Creates:` that were actually produced, and commit. Add nothing under the results root's raw tree.
  Do not edit any other file in the repository between the first repetition and this commit, the
  harness fix on the restart path excepted — that fix precedes the fresh root's first repetition,
  not the run it interrupts.
- **Commit:** `bench(ladder): T7 toc matrix results, cells a0-none and a2-toc-dir at reps 5`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` — the harness's own offline suite, scoped to the
one tree this batch is permitted to change. It runs against the fake-runner layer under
`testdata/`, needs no network and no API budget, and skips the guarded live test because
`LADDER_LIVE_TEST` is unset in the verify environment. The scope is deliberate: this batch's only
possible code change is a harness fix (see `## Batch Scope`), and if one lands its failing table
test goes in the same package tree the command covers. The repository-wide gate
(`go test ./... && golangci-lint run`) still runs once as card 6's first step and again from
`pipeline.done_gate` in the hub's mill configuration before the task is marked done.

The batch's real tests, though, are the run's own gates, and the write-up batch reports all of them:
gate 1 tells the conclusion whether `a2-toc-dir` ever called the tool it was granted — a cell that
never did measured prompt cost only; gate 2 tells it whether any control repetition was
contaminated; and the summary's incomplete and invalid lists, plus the non-zero exit, are the machine
check that the matrix finished. The pre-matrix live test in card 6 is the seam check that no offline
test can stand in for.
