# Batch: execute-matrix

```yaml
task: "Per-capability quarry-mcp benchmark suite"
batch: "execute-matrix"
number: 8
cards: 4
verify: null
depends-on: [6, 7]
```

## Batch Scope

This batch runs the benchmark. It executes the preflight probe, the 42-run main matrix, the 3-run cold cell, and the summarisation, producing the tracked `probe.json` and `summary.json` and the gitignored per-run artifacts under `raw/`. It ships no code: every behavior it exercises was built and tested in batches 1 through 6, and its own correctness condition is the harness's per-run gates, not a test file.

Two things about this batch are unlike every other batch in this plan. First, it is not autonomously reachable: card 24 refuses to start while `ladder.yaml`'s `run_model` is null, and setting it is an operator action taken between batch 7 and this batch. Second, cards 25 and 26 carry `Commit: none` because their entire product lands under the gitignored `results/**/raw/` tree — they are hours of sequential paid execution whose durable record is the two tracked files cards 24 and 27 commit.

Every literal path here spells the results directory `bench/loomyard-eval/ladder/results/2026-08-29/`. If the matrix starts on a different UTC date, the implementer uses that date consistently across all four cards and the batch's own commits, per the overview's dated-results-directory decision.

## Cards

### Card 24: preflight and the denial probe

- **Context:**
  - `bench/loomyard-eval/ladder/README.md`
  - `bench/loomyard-eval/ladder/ladder.yaml`
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/results/2026-08-29/probe.json`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Confirm the preconditions the protocol names, in this order, and stop rather than working around any of them:
  1. `ladder.yaml`'s `run_model` is set. If it is still null, halt with the operator instruction naming the field — do not choose a model. Every headline metric is model-dependent, so an unpinned model makes the whole matrix unreproducible.
  2. The Loomyard checkout exists at the `source_repo` path the ladder declares. If it does not, stop and ask for the correct path rather than substituting another repository; the task files reference specific real files and commits at a specific pin.
  3. The quarry-mcp server builds, which requires `CGO_ENABLED=1` and a working C toolchain.

  Then run the probe — and only the probe — through the harness's `--stage probe` selector, and commit its record. The stage selector is what makes this card a boundary the operator can stop at: without it the harness would run straight on into the 42-run matrix. The probe establishes two things, in order: first that a denied `mcp__quarry__*` call does not succeed at all — if it does, halt before any matrix run, because every rung would silently be the full bundle and all 45 runs would be worthless; and second whether denied tools are advertised to the model or hidden from it, which decides whether `denied_tool_attempts` is a reportable metric or a column of meaningless zeros. Record both answers, the advertised tool list, and the probe's session id in the tracked probe record. A probe whose first assertion fails halts the batch; a probe that shows denied tools are hidden proceeds, and the metric is still summarised but flagged as carrying no information, so the conclusion cannot read its zeros as evidence.
- **Commit:** `bench(ladder): record permission-deny probe result`

### Card 25: execute the 42-run main matrix

- **Context:**
  - `bench/loomyard-eval/ladder/README.md`
  - `bench/loomyard-eval/ladder/ladder.yaml`
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `bench/loomyard-eval/ladder/results/2026-08-29/probe.json`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Run the harness with `--stage main` over the 14 main configs at three repetitions each, sequentially and never concurrently, into `results/2026-08-29/raw/`. Every artifact this card produces is gitignored, which is why it carries no commit.

  The harness owns the protocol; this card's job is to run it to completion and to react correctly to how it stops:
  - A run that fails a gate is invalidated and retried, up to three attempts. On a third failure the harness halts the whole matrix, naming the failing gate. That halt is a real stop: report the gate and the reason and leave the completed runs intact rather than deleting state or lowering a gate to get past it. A deterministic gate failure almost always invalidates the other cells' premises too.
  - A run stopped by the turn ceiling halts the matrix on its first occurrence, and that halt is not a gate fault to retry past. It means `max_turns` is wrong for this matrix, and every run already completed was measured under a ceiling one of its peers could not meet. The remedy is the operator's: raise `max_turns` and re-run the matrix into a new dated results directory. Do not raise it and resume into the existing one — a matrix spanning two ceilings is not a matrix.
  - An interrupted batch is resumed by re-invoking the harness, which skips every run whose directory records `state: "complete"`. Never restart the matrix from scratch to recover from an interruption.
  - A run that dirtied its worktree is not a failure. The harness hard-restores the worktree after every run and records the observation; a rung whose runs mostly had to mutate the target to answer is a finding about that capability, not noise to be suppressed.

  When the matrix completes, confirm that every one of the 14 main configs has exactly three complete runs before moving on.
- **Commit:** none

### Card 26: execute the 3-run cold cell

- **Context:**
  - `bench/loomyard-eval/ladder/README.md`
  - `bench/loomyard-eval/ladder/ladder.yaml`
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Run the harness with `--stage cold` after the entire main matrix has finished, never before and never interleaved. Each of its three runs gets its own freshly-built worktree at a distinct path, no warm-up, and its coldness asserted on both sides: no daemon state before, and daemon state present after. Between runs the harness waits for the previous run's daemon to exit.

  Four dispositions, and they are not the same thing. A cold run that **used a daemon-backed tool** and finishes with no daemon state present took the native fallback; it is invalidated and retried, and a persistent native fallback means the supervised strategy is unavailable on this machine — the cold cell is then reported as **not run**, an environment limitation, and the matrix is not halted. A cold run that used **no** daemon-backed tool is valid and is kept: `toc_file` and `toc_dir` never start a daemon, so its lack of daemon state is not a fallback; if all three repetitions are of this shape the cell holds three good runs that carry no warmth signal, which is a third outcome distinct from not-run. A cell where some but not all repetitions confirmed cold is recorded as partial with its confirmed count: a legitimate mixed outcome on a machine that takes the native path intermittently, not an interrupted run, and one that yields no warm-versus-cold contrast because that contrast needs three cold runs. A different gate failing three times is a fault and does halt the matrix, exactly as in the main matrix.

  The harness records which of the four outcomes occurred in `results/2026-08-29/cold_cell.json`, and that record is what the summariser and the conclusion read — a disposition reported only in this card's output would leave a not-run cold cell indistinguishable on disk from an interrupted one. That file is untracked only if it lands under `raw/`; it does not, so it is committed with card 27's summary.
- **Commit:** none

### Card 27: summarise the matrix

- **Context:**
  - `bench/loomyard-eval/ladder/ladder.yaml`
  - `bench/loomyard-eval/ladder/scripts/summarize.py`
  - `bench/loomyard-eval/ladder/results/2026-08-29/probe.json`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/results/2026-08-29/cold_cell.json`
  - `bench/loomyard-eval/ladder/results/2026-08-29/summary.json`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Run the summariser over the completed results root and commit the tracked summary. It must exit zero: a non-zero exit names cells that are not complete, and an incomplete cell is resolved by re-running the harness, never by summarising a partial matrix as if it were finished. The one exception is a cold cell whose `cold_cell.json` records `not-run` or `partial` — that cell is legitimately short, the summariser excludes it from its incomplete list, and its exit code stays zero. Commit that record alongside the summary: it is what lets the conclusion say why a warm-versus-cold comparison is missing.

  Confirm before committing that the summary carries: the pinned run model and scorer settings in its metadata, the cold cell's disposition, a stats record for every config, the `denied_tool_attempts_reported` boolean the probe established, the per-config dirtied-run, decoy-admitted, and target-origin-mention counts, and comparison records of all three kinds. This file is tracked specifically so every claim the conclusion makes keeps its supporting numbers in the repository after the disposable raw artifacts are deleted.
- **Commit:** `bench(ladder): record per-config medians, ranges, and separations`

## Batch Tests

`verify:` is null: this batch runs the benchmark rather than adding a runnable surface, and the discussion is explicit that the runs themselves are not tests and must not be asserted on. Its verification is the harness's own per-run gate set, applied before each run's terminal marker is written — a parseable answer and extracted metrics, no successful call to a denied tool, no `targetDir` or `buildTags` override, the pinned model, no leak-shaped evidence in a `none` run, a neutralised worktree, the cold cell's before-and-after daemon-state assertions, and a score produced by the pinned scorer from a redacted answer — plus the two matrix-wide conditions card 27 confirms: exactly three complete runs per config, and no reported number sourced from a self-report field.
