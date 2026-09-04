# Plan: Ladder, toc rerun (T7)

```yaml
task: "Ladder, toc rerun (T7)"
slug: "ladder-toc-rerun"
approved: false
started: "20260904-112745"
parent: "main"
root: ""
verify: null
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: pre-matrix-gates
    file: 01-pre-matrix-gates.md
    depends-on: []
    verify: null
  - number: 2
    name: matrix-run
    file: 02-matrix-run.md
    depends-on: [1]
    verify: go test ./bench/loomyard-eval/ladder/...
  - number: 3
    name: write-up
    file: 03-write-up.md
    depends-on: [2]
    verify: null
```

## Shared Decisions

### Decision: results-root-name

- **Decision:** the results root is `bench/loomyard-eval/ladder/results/2026-09-04-toc`. Every path
  this plan spells is literal and assumes that name. If the matrix's first successful invocation
  falls on a later calendar date, the root is renamed for that date before any card writes into it,
  and every plan path is read with the new date substituted — the root is named for the run, not for
  the task branch.
- **Rationale:** the plan and the task both spell `results/<date>-toc`; every `v1-final` root uses a
  `YYYY-MM-DD` prefix, and the `-toc` suffix separates this matrix from the other ladder files that
  will one day write roots on the same day.
- **A restart never reuses this path.** If a harness defect forces a restart, the fresh root takes a
  `-r2` suffix (`results/2026-09-04-toc-r2`, then `-r3`), and the abandoned directory is never passed
  to `--results` again — re-invoking against it makes `ReadProvenance` find the existing record and
  `MergeProvenance` append to it, so every pre-fix repetition still satisfies `RepIsComplete` and is
  skipped, silently merging two measurement regimes into one root. The abandoned root keeps its
  partial artifacts and gains an `ABANDONED.md` naming the fix, the date and the successor root.
  **Card 4 owns that file** — it is the only card that knows both the fix and the successor root —
  and writes it on the restart path only, which is why it appears in that card's `Creates:` as a
  conditional artifact.
- **Applies to:** all batches

### Decision: clean-tree-and-no-edits-mid-matrix

- **Decision:** every `_mill/` artifact and every source change is committed before the matrix's
  first invocation, and the run aborts if `git status --porcelain` is non-empty at that moment. No
  file in the repository is edited between the first repetition and the last — the conclusion is
  written only after the run has finished.
- **Rationale:** this is the harness rule that carried over from V1 verbatim. `provenance.json`
  records `quarry_dirty` and `quarry_dirty_files`; a dirty tree makes the committed record describe
  something that is not in git, which is the exact fault the 2026-09-02 post-mortem fixed.
- **One carve-out, stated rather than hidden: a resumed invocation records `quarry_dirty` true by
  construction.** `CollectInvocation` derives the flag from plain `git status --porcelain`, which
  lists untracked files, and the machine artifacts the first invocation writes into the tracked
  results root are untracked until card 4 commits them after the run terminates. Invocations 2 and 3
  therefore see them and record dirty. That is accepted, because committing between invocations
  would edit the repository mid-matrix, which is the larger of the two faults. What is **not**
  accepted is a dirty file list naming anything else: before each re-invocation the driver confirms
  every entry of `git status --porcelain` is a path under the results root, and card 6 reports
  `quarry_dirty` and `quarry_dirty_files` per invocation entry so a reader sees exactly which files
  were untracked at each one. A dirty entry outside the results root voids the root.
- **Applies to:** all batches

### Decision: matrix-runs-backgrounded-under-env-u

- **Decision:** the matrix runs as a detached background process with stdout and stderr tee'd to a
  log under `.scratch/`, and the driving card polls that log rather than blocking on the command.
  Every invocation of the harness — the matrix and the guarded live test alike — carries the prefix
  `env -u CLAUDECODE -u CLAUDE_CODE_ENTRYPOINT`.
- **Rationale:** ten measured runs at ~60 s each plus ten Opus scorer calls is 20–40 minutes of
  wall-clock, far past any foreground tool call's ceiling, and a killed invocation would strand the
  advisory `.ladder.lock` under the ladder worktree root. The `env -u` is because the harness's
  measured `claude -p` children inherit the driving agent's environment: `CLAUDECODE` and
  `CLAUDE_CODE_ENTRYPOINT` announce "you are running inside a Claude Code session", which is not a
  condition the V1 measurement ran under and not one to introduce silently into a regression gate.
- **Applies to:** matrix-run

### Decision: drift-is-checked-explicitly-not-by-warning

- **Decision:** two checks are performed by hand and their values transcribed into the conclusion:
  (1) `quarry_commit` equality across every entry of `provenance.json`'s `invocations` list, and
  (2) an out-of-band `sha256sum` of the built server binary, taken immediately **after** each
  invocation of the matrix command and appended to the run log.
- **The reading is taken after, not before, and that ordering is load-bearing.** `run.go` builds
  `<ladder-worktree-root>/bin/<server-name>` through `BuildServer` *inside* the invocation, so before
  the first invocation the file is either absent or a stale leftover from an unrelated run — a
  pre-invocation reading would hash something that is not the binary under test, and the last
  invocation's binary would never be hashed at all. The guarded live test does not build it either:
  it passes an empty server-binary path. A pre-existing `bin/quarry` at gate time is therefore stale
  and is not a baseline.
- **Rationale:** `WarnOnServerHashDrift` cannot fire after a resume. The harness builds the server
  once per invocation and writes that one hash into `ServerHashes` for every repetition of every
  non-control cell — including repetitions an earlier invocation already completed — and
  `CollectInvocation` leaves the per-invocation copy nil, so after any resume the map holds exactly
  one distinct hash. Since resume is the normal path here, the detector is silent in precisely this
  workflow. The `invocations` list is appended to rather than rewritten, so it survives.
- **The conclusion states the values it checked, never that a warning stayed silent.**
- **Applies to:** matrix-run, write-up

### Decision: resume-not-restart-with-an-explicit-ceiling

- **Decision:** an incomplete or blinding-invalidated repetition is re-attempted by re-running the
  same command against the same results root. The driver caps itself at **three invocations of the
  matrix command per results root**, and the third is the last. Before each re-invocation it lists
  `<root>/raw/<cell>/<rep>.invalid-*` for every missing repetition: a repetition with three or more
  such directories is attempt-exhausted and will never complete, so when every missing repetition is
  exhausted the driver does not re-invoke at all and goes straight to the write-up.
- **Rationale:** `MaxAttempts` is not a stop condition across invocations. `InvalidateRep` derives
  its counter from the `.invalid-<k>` directories already on disk, so a resumed invocation's first
  attempt at an already-thrice-invalidated repetition is attempt 4: the ceiling trips immediately,
  the cell is recorded incomplete and the invocation returns non-zero — after spending a fresh
  measured `claude -p` call getting there. Neither the summary nor the exit code distinguishes
  "never attempted" from "attempt-exhausted", so a loop that simply re-runs until the exit code is
  zero would burn API budget forever.
- **A blinding failure is not covered by `MaxAttempts` at all.** `writeCompleteState` with
  `blindingFailed` set writes the repetition complete-with-the-flag rather than invalidating it, so
  no counter increments and `RepIsComplete` still returns false. Such a repetition is re-attempted
  **exactly once, and only after the cause is diagnosed and named**; a second failure of the same
  repetition stops the matrix. Gate 2 check (d), `CheckRenderedControlPrompt`, is deterministic and
  pre-dispatch, so it gets **zero** re-attempts — stop immediately.
- **Two outcomes are accepted, never retried, and always reported:** a scorer failure (written
  complete with `scored: false` and `score_skip_reason` set to `scorer_failed`, counted in
  `UnscoredCount`, dropped from recall and precision only) and a max-turns completion (complete by
  design, counted in `MaxTurnsCount`).
- **Applies to:** matrix-run, write-up

### Decision: raw-tree-stays-untracked

- **Decision:** `results/**/raw/` stays untracked. `bench/loomyard-eval/ladder/.gitignore` already
  carries `results/*/raw/`, shipped by T2; this task confirms that as the settled answer to plan §11
  and records it in the conclusion and in the plan document.
- **Rationale:** `raw/memory-paths.json` holds resolved auto-memory directory paths, and the
  repository's rule is that no tracked file carries a machine path — `provenance.json` stores sha256
  hashes of those paths precisely to honour it. Second, the raw tree is ten transcripts of a 60-turn
  ceiling plus their answers, scores and usage files: large, per-host, and fully summarised by the
  committed artifacts.
- **Applies to:** all batches

### Decision: committed-artifacts

- **Decision:** the results root commits exactly `conclusion.md`, `summary.json`, `provenance.json`,
  `table.txt` and `probe.md`. Nothing else from the root is added to git.
- **Rationale:** the first three are what every `v1-final` root committed; the rendered table is the
  artifact a reader looks at first; the probe report is the §9a evidence this task must produce. All
  five are small and machine-path-free.
- **Applies to:** all batches

### Decision: metric-key-spellings

- **Decision:** in the summary JSON the metric keys are the harness's own `costMetricNames`; the
  rendered table's `tableColumnNames` renames four of them. The full rename table, and the only four
  places the two spellings differ:

  | summary JSON key              | rendered table column |
  | ----------------------------- | --------------------- |
  | `cache_read_input_tokens`     | `cache_read`          |
  | `cache_creation_input_tokens` | `cache_creation`      |
  | `quarry_tool_uses`            | `prefixed_tool_uses`  |
  | `grep_fallback_total`         | `grep_fallback`       |

  Every other name — `turns`, `duration_ms`, `cost_usd`, `output_tokens`, `input_tokens_total`,
  `tool_uses`, `tool_result_bytes`, `read_bytes` — is spelled identically on both sides. Read a
  comparison entry out of the JSON by its JSON key; read the column off the rendered table. The
  gate 1 tool-use report is exactly where the `quarry_tool_uses` / `prefixed_tool_uses` pair bites.
  Prose in the conclusion may use the short names, provided the artifact it cites is named.
- **Rationale:** the two spellings are easy to conflate and a conclusion that quotes the wrong key
  is unverifiable against the artifact committed beside it.
- **Applies to:** write-up

### Decision: separation-verdict-comes-from-the-harness

- **Decision:** the headline claim — does `a2-toc-dir` separate from `a0-none` on turns and
  cache-read — is taken from the summary's own comparison entries and their `separated` field,
  quoted alongside `median [min–max]` for each metric. Every number in the conclusion's metric table
  comes from this results root.
- **Rationale:** `Summarize` already recomputes each cost metric per cell over the present, non-void
  repetitions and builds one comparison per rung-cell-and-metric pair with a disjointness verdict.
  Re-deriving the verdict by hand is how a conclusion starts disagreeing with its own artifacts.
  `separated` is a strict no-overlap test on the min–max ranges: at n=5 a real effect can be present
  without it firing, so medians and ranges are reported too and `separated` is treated as evidence,
  not as the sole criterion.
- **Applies to:** write-up

### Decision: prior-numbers-are-cited-not-merged

- **Decision:** the `v1-final` figures appear in one clearly-labelled prior-record section naming
  the branch, root and reps they come from, and never in this root's metric table.
- **Rationale:** cost numbers compare only within one results root — different host, different
  harness, different CLI version, different cache behaviour. The comparison this conclusion is
  entitled to make is qualitative: does the same direction and rough magnitude of effect appear
  again? Correctness metrics are the stated exception and may be compared by id across roots.
- **Applies to:** write-up

### Decision: the-done-gate-runs-offline

- **Decision:** the done gate stays `go test ./... && golangci-lint run` with `LADDER_LIVE_TEST`
  unset, so the guarded live test skips. The live test runs once, explicitly, as the first
  pre-matrix gate.
- **Where the gate is encoded:** in `mill-config.yaml` at the hub root, as `pipeline.done_gate`,
  already carrying exactly that command. It is run from the repository root before the task is
  marked done, so no card and no `verify:` field needs to restate it. This decision is a statement
  of what that key holds and a requirement that it keeps holding it — not a second, parallel gate.
  The plan deliberately does **not** put the command in the overview's module-wide `verify:` field:
  that field runs at every batch boundary, and a full `go test ./...` plus a lint pass after each of
  three batches — two of which change no Go code at all — would pay the gate's cost three times over
  for one signal the done gate already produces once.
- **Rationale:** the guard exists so the repeated gate stays free, deterministic and network-free;
  making it spend API budget on every invocation is the thing that guard prevents.
- **Applies to:** all batches

### Decision: harness-fixes-restart-the-matrix

- **Decision:** if a harness defect blocks the run, it is fixed under `bench/loomyard-eval/ladder/`
  with a failing table test written first, committed, and the matrix restarts in a fresh `-r2` root.
  Repetitions measured by the pre-fix harness are never mixed with post-fix ones in the same root,
  and the conclusion names the fix and the abandoned root. If the defect is in the code under test
  instead, the run stops and the conclusion records it; fixing it is a separate task.
- **Rationale:** the harness is not the code under test, so fixing it breaks no rule — but it changes
  how a repetition is measured, and the per-repetition server hash covers the server binary, not the
  harness. A fresh root is the only honest boundary.
- **Applies to:** matrix-run

### Decision: the-code-under-test-is-not-edited

- **Decision:** no card in this plan edits `internal/`, `quarry/`, `cmd/` or the ladder file's
  measured parameters (cells, reps, models, effort, turn ceiling, pins, tasks, fasit). If the
  measurement exposes a defect there, the conclusion records it and it becomes a separate task.
- **Rationale:** those are the code under test, and the ladder file's parameters are what makes this
  root comparable with the thing it exists to reproduce.
- **Applies to:** all batches

## All Files Touched

- `HANDOFF.md`
- `bench/loomyard-eval/ladder/results/2026-09-04-toc/ABANDONED.md`
- `bench/loomyard-eval/ladder/results/2026-09-04-toc/conclusion.md`
- `bench/loomyard-eval/ladder/results/2026-09-04-toc/probe.md`
- `bench/loomyard-eval/ladder/results/2026-09-04-toc/provenance.json`
- `bench/loomyard-eval/ladder/results/2026-09-04-toc/summary.json`
- `bench/loomyard-eval/ladder/results/2026-09-04-toc/table.txt`
- `docs/rewrite-plan.md`
