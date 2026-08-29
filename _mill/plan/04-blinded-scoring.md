# Batch: blinded-scoring

```yaml
task: "Per-capability quarry-mcp benchmark suite"
batch: "blinded-scoring"
number: 4
cards: 2
verify: uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests/test_score_run.py -q
depends-on: [1]
```

## Batch Scope

This batch delivers `score_run.py`: the answer redaction that keeps the scoring agent blind to which rung it is grading, and the pinned, three-input scoring dispatch that turns one run's answer into a `score.json`. It depends only on batch 1 because it reads the ladder for the pinned scorer settings and the per-task fasit path, and never touches a transcript.

The external interface batch 6 consumes is `score_run.py`'s `score_run(ladder, config, run_dir, task_text)`, which writes both `answer.redacted.json` and `score.json` into the run directory and returns the parsed score mapping. `run_ladder.py` calls it immediately after metrics extraction; a run is not complete until it has succeeded.

Batch-local decision: the scorer's prompt template is selected by the task's `schema` field — `exploration` for Ladder A, `impact` for Ladder B — and both templates are module-level constants so a drifting scorer prompt is impossible without a tracked edit.

## Cards

### Card 11: redact the answer before it reaches the scorer

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/tests/conftest.py`
  - `bench/loomyard-eval/results/2026-08-28/04-shedadapters-shuttle-impact/c.json`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/scripts/score_run.py`
  - `bench/loomyard-eval/ladder/tests/test_score_run.py`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Write the tests first. Add module-level `REDACTION_TOKEN = "<tool>"` and:
  - `redact_text(text)` — replace, case-insensitively, every `mcp__quarry__*` client-side name, every bare canonical quarry tool name from `QUARRY_TOOLS` **except `impact`**, the word `quarry`, and CLI invocation forms — the literal string "/tmp/quarry-bench", a "quarry &lt;verb&gt;" shell form, and a "--target-dir"-style flag — with `REDACTION_TOKEN`. Collapse a run of adjacent tokens produced by one phrase into a single token so the redacted prose stays readable.

    `impact` is excluded from the bare-name pass, and only its `mcp__quarry__impact` and CLI-invocation forms are redacted. It is the one canonical name that is also an ordinary English word, and every Ladder-B run is an *impact analysis* whose `evidence` and `reason` prose uses it as a common noun — a case-insensitive bare-name pass would turn "the impact of this change" into "the `<tool>` of this change" across all 24 Ladder-B answers, destroying the very field the impact scoring rule reads. The six remaining names have no such collision.
  - `redact_answer(answer)` — return a deep copy with `redact_text` applied to every free-text field and to every free-text field inside a nested list entry, specifically the exploration schema's `summary`, `open_questions`, and each `key_symbols[].role`, and the impact schema's `open_questions`, each `callers_to_update[].evidence`, and each `excluded_lookalikes[].reason`. Structural fields — `relevant_files`, every `file`, every `line`, every `name`, and `confidence` — are returned untouched.
  - `write_redacted(run_dir)` — read `answer.json`, write `answer.redacted.json` beside it, and leave the original byte-identical.

  Tests use an impact-shaped answer whose `evidence` reads like the committed `#006` A-arm phrasing in the Context fasit — naming a quarry tool and the word quarry — and assert:
  - an `evidence` string using "impact" as a common noun survives redaction unchanged, while the same string's `mcp__quarry__impact` occurrence does not; the tool name and the word are both gone from the redacted copy; file paths and line numbers are unchanged; `confidence` is unchanged; the original file on disk is byte-identical after `write_redacted`; and an exploration-shaped answer's `relevant_files` survives while its `summary` is redacted. A separate test asserts that a redacted answer contains no case-insensitive occurrence of `quarry` and no `mcp__quarry__` prefix at all.
- **Commit:** `feat(bench): redact tool provenance from answers before scoring`

### Card 12: dispatch the pinned, blinded scoring call

- **Context:**
  - `bench/loomyard-eval/ladder/ladder.yaml`
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/README.md`
  - `bench/loomyard-eval/results/2026-08-28/01-reed-geometry-exploration/c.json`
  - `bench/loomyard-eval/results/2026-08-28/04-shedadapters-shuttle-impact/c.json`
- **Edits:**
  - `bench/loomyard-eval/ladder/scripts/score_run.py`
  - `bench/loomyard-eval/ladder/tests/test_score_run.py`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Write the tests first. Add two module-level prompt templates and the dispatch:
  - `EXPLORATION_RULE` — the exploration scoring rule reproduced from the committed benchmark README's Scoring section unchanged: recall is the fasit's `relevant_files` and `key_symbols` also present in the run's over the fasit's total; precision is the run's entries corroborated by the fasit over the run's total; plus a qualitative judgement of whether the run's `summary` describes the same mechanism the fasit found.
  - `IMPACT_RULE` — the rule the discussion writes for Ladder B, stated in full: recall is the fasit's `callers_to_update` entries matched on file **and** line, where a line must denote the same call site rather than merely the same file, over the fasit's total; precision is the run's `callers_to_update` entries corroborated by the fasit over the run's total; `decoy_admitted` is `true` when the run's `callers_to_update` contains a call site the fasit identifies as a lookalike, reported as its own field and never folded into precision; and `lookalikes_matched` is the count of the run's `excluded_lookalikes` the fasit also names, credited but never required, so a run naming none loses no points.
  - `strip_fasit_meta(fasit)` — return the fasit with its top-level `_meta` block removed. The committed `c.json` files carry a `_meta` whose `role` string contains the word `quarry` and whose `see_also` names `scorecard.md`, and feeding that verbatim beside an answer deliberately scrubbed of that same word is an undocumented asymmetry. It leaks no *config* — `_meta` is identical across all 45 runs of a task — but it is scoring-irrelevant, so it is dropped rather than left to sit unexplained next to the redaction.
  - The fasit's own free-text fields are **left verbatim**, and deliberately so. The committed task-04 fasit says things like "quarry impact on singlellm.go:39:2" in three `callers_to_update[].evidence` entries and in an `excluded_lookalikes[].reason`, which looks like the same leak the answer redaction exists to close, but it is not the same leak: the fasit is one fixed file, identical across all 45 runs of its task, so it cannot tell the scorer *which config* it is grading — and that is the only thing the blinding claims. The answer varies per rung and is therefore redacted; the fasit does not and is not. Redacting it would also damage the thing it is there for, since the evidence fields are what the recall and precision rules match a run's own entries against. Only `_meta` is dropped, because it is scoring-irrelevant rather than because of what it contains.
  - `build_scorer_prompt(ladder, config, redacted_answer, fasit, task_text)` — assemble the prompt from exactly three inputs plus the rule: the redacted answer, the `_meta`-stripped fasit, and the task text. It must never embed the config id, the ladder letter, the allowed-tool list, the transcript, or any other run's answer. The task text is included because the exploration rule cannot judge the summary without it, and it is identical across a ladder's rungs.
  - `score_run(ladder, config, run_dir, task_text, runner=...)` — call `write_redacted`, build the prompt, dispatch it through the injected `runner`, parse a fenced json block out of the reply, and write `score.json` carrying `recall`, `precision`, the impact-only `decoy_admitted` and `lookalikes_matched`, the qualitative `summary_matches` judgement for exploration, and the resolved scorer `model`, `effort`, and `prompt_template` so a drifting scorer is visible in the record.
  - The default `runner` shells out to `claude -p` with `--setting-sources ""`, `--strict-mcp-config`, `--permission-mode dontAsk`, `--model` and `--effort` from the ladder's pinned `scorer` mapping, and `--output-format json`, and raises `ScoringError` on a non-zero exit or an unparseable reply. The permission mode matches every other dispatch in the suite: a scoring call that reached for a tool would otherwise block on a prompt no one is there to answer, stalling the matrix mid-run.

  Tests inject a fake `runner` and never make a live call. They assert: the prompt contains the redacted answer, the fasit, and the task text and contains none of the config id, the ladder letter, or any allowed-tool name; the fasit reaching the prompt carries no `_meta` block while its `callers_to_update[].evidence` text survives verbatim, both asserted against the committed fasit shape; the exploration template is selected for a Ladder-A config and the impact template for a Ladder-B config; `score.json` records the pinned scorer model, effort, and template; a reply with no fenced json block raises `ScoringError`; and `score_run` passes the redacted copy to the runner while the original `answer.json` is never read by it. A separate test asserts an impact score carries `decoy_admitted` and `lookalikes_matched` as their own fields, with `decoy_admitted` absent from the precision computation the template describes.
- **Commit:** `feat(bench): dispatch pinned blinded scoring per run`

## Batch Tests

`verify:` runs `bench/loomyard-eval/ladder/tests/test_score_run.py`, the only test file this batch creates. Both cards' units are pure once the model call is injected, so the whole batch is tested without a network call or a live model call — which is exactly the boundary the discussion draws: the dispatch layer is exercised by actually running the matrix in batch 8, never by mocking a model inside the unit suite. The injected `runner` here mocks the subprocess boundary, not the model's judgement, and no test asserts anything about what a real scorer would return.
