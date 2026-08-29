# Batch: protocol-readme

```yaml
task: "Per-capability quarry-mcp benchmark suite"
batch: "protocol-readme"
number: 7
cards: 1
verify: null
depends-on: [1, 6]
```

## Batch Scope

This batch delivers the suite's self-contained protocol document. It is its own batch because it must describe the harness as built, not as planned: it depends on batch 6 so every command, flag, and file name it documents is one that exists. It ships no runnable surface of its own.

## Cards

### Card 23: write the ladder protocol README

- **Context:**
  - `bench/loomyard-eval/README.md`
  - `bench/loomyard-eval/ladder/ladder.yaml`
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `bench/loomyard-eval/ladder/scripts/extract_usage.py`
  - `bench/loomyard-eval/ladder/scripts/gates.py`
  - `bench/loomyard-eval/ladder/scripts/score_run.py`
  - `bench/loomyard-eval/ladder/scripts/summarize.py`
  - `bench/loomyard-eval/tasks/01-reed-geometry-exploration.md`
  - `bench/loomyard-eval/tasks/04-shedadapters-shuttle-impact.md`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/README.md`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Write the protocol so a fresh agent with no memory of this design can execute and re-run the matrix from this file alone, mirroring the framing the sibling benchmark README opens with. Required sections:
  - **The question** — per-capability attribution within quarry's MCP surface, and the explicit statement that this suite does not benchmark the quarry CLI and builds no CLI binary for any run.
  - **What this fixes** — the three methodology defects it exists to correct: single-bundle exposure that cannot attribute a result to any tool, orchestrator-executed runs that were retracted as contaminated, and one run per arm.
  - **The two ladders** — both tables reproduced exactly as `ladder.yaml` declares them, plus the cold cell, with the note that config ids are ladder-qualified and globally unique and that `ladder.yaml` is the single source of truth the tables must be kept consistent with.
  - **Enforcement** — the deny-list derived from the allow-set, that a config with an empty allowed set is launched with no MCP config at all, that `permissions.allow` prevents prompting rather than restricting, that `Task` is denied uniformly in all 45 runs and why, and that the blinding is enforced by transcript detection rather than by construction. State plainly that the `none` arms' blinding is weaker than a structural guarantee.
  - **Run environment** — the pinned model, the identical allow-set, `--setting-sources ""`, `--strict-mcp-config`, the non-interactive permission mode, `QUARRY_STATE_DIR` cleared, `--state-dir` never passed, and the pre-warmed daemon for the main matrix.
  - **Metrics** — every field a `usage.json` carries, with `bash_grep_count`'s Bash-only definition and `grep_tool_count`'s separate scope spelled out, and the rule that a metric is extracted from a transcript and never from an agent's self-report.
  - **Scoring** — the three-input blinded contract, the redaction and its acknowledged residual limit, the exploration rule reused unchanged, and the impact rule in full including the separately-reported decoy column and the credited-but-not-required lookalikes.
  - **Reporting discipline** — medians and full ranges, the disjoint-range bar for all three comparison types, the grep-metric exclusion for rung-vs-control, `n = 3` stated explicitly, and no significance claims.
  - **Hard rule on cross-suite comparison** — correctness may be compared to the sibling suite's committed results; duration, tokens, cost, turns, tool counts, and grep counts may not, even informally, and the reasons why.
  - **How to run** — the operator prerequisite of setting `run_model` in `ladder.yaml` first, then the harness invocation, then the summarise invocation, and the statement that re-invoking the harness resumes by skipping completed runs.
  - **How to test** — `uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests -q` from the repo root, with the note that a machine that already has pytest and PyYAML can use the plain `python -m pytest bench/loomyard-eval/ladder/tests` form.
  - **Layout** — the directory tree as built, showing all six modules under `scripts/`, the tracked `tests/` and `tests/fixtures/`, and which results files are tracked versus gitignored.
  - **Design rationale — do not "simplify" these away** — the load-bearing constraints, matching the sibling README's own closing section: sequential dispatch, the cold cell's per-run distinct worktrees and its supervised-connection assertion, the cold cell running last, the hard worktree restore after every run with dirtying recorded rather than gated, the three-attempt cap halting the whole matrix, and the preamble steering confound and what it forbids the conclusion from claiming.
- **Commit:** `docs(bench): add the per-capability ladder protocol README`

## Batch Tests

`verify:` is null: this is a pure documentation batch with no runnable surface. Its correctness condition is that every command, flag, file name, and metric field it documents matches what batches 1 through 6 actually built, which the card's `Context:` list makes checkable by reading the six modules and `ladder.yaml` beside it. The suite's own test command is documented here but not run by this batch — batches 1 through 6 each verify their own module.
