# Plan: M4 matrix run: execute the descoped kick-start batch (cards 29-32)

```yaml
task: 'M4 matrix run: execute the descoped kick-start batch (cards 29-32)'
slug: 'kickstart-matrix-run'
approved: false
discussion_sha: '8f49f98695d6259a805f628a0f64752aaf3df0d2'
started: '20260906-092512'
parent: 'main'
root: ""
verify: null
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: matrix-and-writeup
    file: 01-matrix-and-writeup.md
    depends-on: []
    verify: go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestPreMatrix'
```

## Shared Decisions

### Decision: results-root-date-resolved-once

- **Decision:** the placeholder `<RUN_DATE>` appearing in every `Creates:`/`Edits:` path under
  `bench/loomyard-eval/ladder/results/` is not a literal directory name. Card 29 resolves it once
  with `date -u +%F`, writes the resolved absolute results-root path into
  `.scratch/kickstart-results-root.txt`, and cards 30, 31 and 32 read that file and reuse the
  resolved path verbatim. No card re-derives the date.
- **Rationale:** the frozen batch spec calls the date "a fact about the run, not a plan constant".
  Hard-coding a date in the plan breaks if mill-go executes on a later day, and re-deriving it per
  card breaks if the run crosses midnight UTC.
- **Applies to:** all batches

### Decision: commit-clean-before-each-harness-invocation

- **Decision:** immediately before each of the two harness invocations — `ladder pack` in card 29
  and `ladder run` in card 30 — the worktree must be committed clean, so that
  `git status --porcelain` reports nothing outside ignored paths.
- **Rationale:** the harness records `git status --porcelain` of the quarry repository root into the
  provenance record, and that root resolves to this worktree. `_mill/**` is tracked, so uncommitted
  mill bookkeeping would land in `quarry_dirty_files` and force the conclusion to carry a carve-out
  paragraph. The reference results root records `quarry_dirty: false`; matching it costs one commit.
- **Applies to:** all batches

### Decision: no-optional-stopping

- **Decision:** the measurement design is frozen and this plan never renegotiates it. No cell
  selection, no `--reps` override, no dropped or added repetitions, no re-running an arm because its
  numbers look wrong, and no resume of an invocation that completed normally. The one permitted
  resume is after an abnormal process death, and only before any result has been read. The one
  permitted repetition deletion is the memory-path-taint case in card 30, likewise before any result
  has been read, and it is recorded in the conclusion.
- **Rationale:** the primary comparison, its metrics, its alternative, its n and its critical value
  were predeclared and frozen before the harness was written, precisely so this run could be
  executed later without renegotiation.
- **Applies to:** all batches

### Decision: harness-and-target-repo-are-read-only

- **Decision:** no Go source under `bench/loomyard-eval/ladder/internal/ladder/`,
  `bench/loomyard-eval/ladder/cmd/`, `quarry/`, `glyph/`, `internal/` or `cmd/` is changed by this
  plan, and neither is any test. The loomyard checkout at `/home/knatte/Code/loomyard/wts/loomyard`
  is read-only input; the harness pins its own worktree from `pinned_sha`.
- **Rationale:** the task's job is to run the harness exactly as merged. Any harness change would
  make the measurement a measurement of different code than the one whose behaviour was predeclared.
- **Applies to:** all batches

### Decision: done-gate-runs-with-the-env-var-unset

- **Decision:** the repository-wide done gate `go test ./... && golangci-lint run` runs with
  `LADDER_LOOMYARD_REPO` unset in the shell environment. It is never exported. The bench reaches the
  target repository through `.scratch/ladder.env` instead.
- **Rationale:** the engine's golden test reads the variable directly and skips when it is unset,
  but fails when the named checkout's HEAD is not the pinned commit — and the operator's checkout is
  not at the pin. Leaving the variable unset makes the golden test skip while the harness still
  resolves the repository from the scratch env file.
- **Applies to:** all batches

## All Files Touched

- `.scratch/kickstart-results-root.txt`
- `bench/loomyard-eval/cards/07-e0-names.md`
- `bench/loomyard-eval/cards/07-e1-pack.md`
- `bench/loomyard-eval/cards/07-e2-files.md`
- `bench/loomyard-eval/ladder/ladder-kickstart.yaml`
- `bench/loomyard-eval/ladder/results/<RUN_DATE>-kickstart/conclusion.md`
- `bench/loomyard-eval/ladder/results/<RUN_DATE>-kickstart/pack-resolve.json`
- `bench/loomyard-eval/ladder/results/<RUN_DATE>-kickstart/provenance.json`
- `bench/loomyard-eval/ladder/results/<RUN_DATE>-kickstart/summary.json`
- `bench/loomyard-eval/ladder/results/<RUN_DATE>-kickstart/table.txt`
- `bench/loomyard-eval/tasks/07-fabric-merge-state-tracing.md`
- `docs/roadmap.md`
