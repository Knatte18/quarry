# Plan: Per-capability quarry-mcp benchmark suite

```yaml
task: "Per-capability quarry-mcp benchmark suite"
slug: "mcp-capability-bench"
approved: true
started: "20260829-124916"
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
    name: suite-skeleton-and-generation
    file: 01-suite-skeleton-and-generation.md
    depends-on: []
    verify: uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests/test_ladder_config.py -q
  - number: 2
    name: usage-extraction
    file: 02-usage-extraction.md
    depends-on: [1]
    verify: uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests/test_extract_usage.py -q
  - number: 3
    name: validation-gates
    file: 03-validation-gates.md
    depends-on: [1, 2]
    verify: uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests/test_gates.py -q
  - number: 4
    name: blinded-scoring
    file: 04-blinded-scoring.md
    depends-on: [1]
    verify: uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests/test_score_run.py -q
  - number: 5
    name: summarize-and-separation
    file: 05-summarize-and-separation.md
    depends-on: [1, 2, 3, 4]
    verify: uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests/test_summarize.py -q
  - number: 6
    name: run-orchestration
    file: 06-run-orchestration.md
    depends-on: [1, 2, 3, 4, 5]
    verify: uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests/test_run_ladder.py -q
  - number: 7
    name: protocol-readme
    file: 07-protocol-readme.md
    depends-on: [1, 6]
    verify: null
  - number: 8
    name: execute-matrix
    file: 08-execute-matrix.md
    depends-on: [6, 7]
    verify: null
  - number: 9
    name: conclusion
    file: 09-conclusion.md
    depends-on: [8]
    verify: null
```

## Shared Decisions

### Decision: no quarry source is modified

- **Decision:** No file under `cmd/`, `internal/`, or `quarry/` is edited, created, or deleted by any card in this plan. Every card writes only under `bench/loomyard-eval/ladder/` plus the single `.gitignore` edit in card 1.
- **Rationale:** the discussion's Scope block puts quarry itself explicitly out of scope — this task measures the existing MCP exposure layer, it does not change it. A quarry defect surfaced by the suite is written up in the conclusion and filed, never fixed here.
- **Applies to:** all batches

### Decision: `#006`'s committed results are read-only

- **Decision:** nothing under `bench/loomyard-eval/results/` is edited, moved, or deleted. The task files under `bench/loomyard-eval/tasks/` and `bench/loomyard-eval/README.md` are likewise read-only — they are consumed verbatim, never amended.
- **Rationale:** the discussion's Scope block puts re-running or amending `#006`'s results out of scope, including its corrections. This suite writes only into its own `ladder/` tree so two methodologies with different tracking rules never mix.
- **Applies to:** all batches

### Decision: six modules under `scripts/`, not four

- **Decision:** the discussion's layout names four scripts. This plan ships those four as the documented entry points (`run_ladder.py`, `extract_usage.py`, `score_run.py`, `summarize.py`) plus two importable helper modules in the same directory: `ladder_config.py` (ladder.yaml loading and validation, deny-list and settings generation, preamble generation) and `gates.py` (the per-run validation-gate predicates).
- **Rationale:** the discussion's Testing section names deny-list generation, preamble generation, and every validation gate as separately-tested deterministic units. Leaving them inside `run_ladder.py` would force each test to import the dispatch layer, and would push one file past 900 lines while it also owns worktrees, warmth, sequencing, resume, and the attempt cap. The layout decision is about tree shape — a sibling directory holding the suite's own scripts — not about forbidding helper modules, and the README's tree is updated to show all six.
- **Applies to:** all batches

### Decision: `permissions.allow` does not restrict; only `permissions.deny` enforces

- **Decision:** the generated settings file's `permissions.allow` entries exist solely to keep a headless run from blocking on a permission prompt. They are not treated as an allowlist anywhere in this suite, and no claim in the README, the gates, or the conclusion may rest on them bounding the toolset. The only structural restriction is `permissions.deny`, whose effectiveness is established once by the preflight probe (card 24) before any matrix run starts.
- **Rationale:** established empirically while planning: a run launched with `permissions.allow` set to `["Read","Grep","Glob"]` under `--permission-mode dontAsk` still executed a `Bash` call successfully, with an empty `permission_denials` array in its result envelope. The discussion already frames `allow` as prompt-avoidance rather than enforcement; this decision makes that binding on every downstream claim so the suite never overstates what was enforced.
- **Applies to:** all batches

### Decision: the `Task` tool is denied uniformly in all 45 runs

- **Decision:** every generated settings file — for both `none` controls and every quarry rung — carries `Task` in `permissions.deny`, alongside that config's quarry deny-list.
- **Rationale:** without `--forward-subagent-text` a subagent's own tool calls never reach the captured `stream-json` transcript, so a run that dispatched `Task` would report a falsely low `tool_uses` and an incomplete `tool_uses_breakdown` — exactly the self-report-style undercount the transcript-extraction decision exists to eliminate, one layer down. Denial is uniform across all 45 runs, so it cannot bias any rung-vs-rung, rung-vs-control, or warm-vs-cold comparison. `Task` is not a code-navigation capability, so this does not weaken the "non-quarry tools stay available" decision, which is about Read/Grep/Glob/Bash. `--forward-subagent-text` was rejected as the alternative because its documented scope is text and thinking blocks, and whether it also forwards `tool_use` blocks is unverified.
- **Applies to:** all batches

### Decision: filesystem setting sources are disabled per run

- **Decision:** every `claude -p` invocation the harness makes — matrix runs, the preflight probe, and every scoring call — passes `--setting-sources ""` and `--strict-mcp-config`.
- **Rationale:** the discussion neutralises the task worktree's own `.claude/` directory but does not address the operator's user-level settings, which would otherwise layer a second, unaudited permissions and hooks source under the generated `--settings` file — the same defect the worktree-neutralisation decision exists to prevent, sourced from the operator's home directory instead. `--strict-mcp-config` is the matching guard for MCP servers: without it a `none` run could inherit an ambient quarry server declaration and observe the `mcp__quarry__*` namespace, defeating the blinding.
- **Applies to:** all batches

### Decision: every client flag the harness depends on was checked against the installed client

- **Decision:** all 46 benchmark dispatches and all 45 scoring calls depend on a small set of `claude` flags being accepted with the exact spelling the plan gives. Each was exercised against the installed client (2.1.236) while planning, and the harness uses no flag that was not: `--print`, `--output-format stream-json` with `--verbose`, `--output-format json`, `--setting-sources ""`, `--strict-mcp-config`, `--settings <file>`, `--permission-mode dontAsk`, `--model <full model id>`, `--effort high`, `--mcp-config <file>`, and `--max-turns`.
- **Rationale:** a flag rejected by the client fails every one of the 46 runs identically, and would be discovered only after the operator has pinned a model and started a multi-hour paid matrix. `--permission-mode dontAsk` was exercised in both planning probes; `--effort high` alongside an explicit `--model claude-opus-5` in a third. Recording the check here means the plan's flag spellings are evidence rather than recollection, and a future client version that drops one of them fails against a written list rather than silently.
- **Applies to:** batch 4, batch 6, batch 8

### Decision: the cold cell can only measure warmth on a daemon-backed run

- **Decision:** `toc_file` and `toc_dir` are excluded from the set of daemon-backed tools. Every cold-cell assertion about daemon state applies only to a run whose transcript contains at least one call to a tool in `DAEMON_BACKED_TOOLS` — `textDocument_definition`, `textDocument_references`, `workspace_symbol`, `impact`, `assert_no_callers`. A cold run that invoked none of them is valid and is not invalidated; it simply carries no warmth signal, and both the summary and the conclusion say so.
- **Rationale:** established from source while planning: `tocFileHandler` and `tocDirHandler` in `internal/mcpserver/tools_toc.go` call `effectiveTargetDir` and `tocPreflight` directly and never `resolveCall`, so a toc call never reaches `EnsureServer` and never writes a `daemon.json`. The cold cell runs `a5-bundle-cold` against the task-01 exploration task, which a bundle agent can plausibly answer with toc calls alone — so "no `daemon.json` after the run" has two completely different causes, and reading it unconditionally as "the native fallback was taken" would invalidate perfectly good runs and, worse, could invalidate all three and report an environment limitation that does not exist. This is also a real limit on what the cold cell can deliver: if none of its three runs touches a daemon-backed tool, the warm-versus-cold question is reported as unanswered rather than answered from runs that never started a daemon.
- **Applies to:** batch 3, batch 5, batch 6, batch 8, batch 9

### Decision: token classes are summed from assistant events, not read off the result envelope

- **Decision:** `input_tokens`, `output_tokens`, `cache_read_input_tokens`, and `cache_creation_input_tokens` are each summed across every `type: "assistant"` event's `message.usage` object in the transcript. The result event's own `usage` object is additionally recorded verbatim as `result_usage`, for cross-checking only; no reported per-class figure is taken from it.
- **Rationale:** the result envelope's `usage` reports a final-iteration view (observed while planning: `input_tokens: 4` against an `iterations` array covering the whole run), so reading per-class totals off it would undercount every run. Summing assistant events is also exactly what `#006`'s own transcript re-extraction did, which keeps the one axis this suite is allowed to compare — correctness — sitting beside cost numbers derived the same way.
- **Applies to:** batch 2, batch 5, batch 8

### Decision: pytest runs through `uv`, not a bare `python -m pytest`

- **Decision:** every batch's `verify:` and the README's documented test command are `uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests/<file> -q`, run from the repo root.
- **Rationale:** the discussion specifies `python -m pytest bench/loomyard-eval/ladder/tests`, but pytest is not installed in this machine's system Python (confirmed while planning: `ModuleNotFoundError: No module named 'pytest'`), so that command cannot run as written. `uv` is present and `uv run --no-project` provisions an ephemeral environment without adding a `pyproject.toml` to a Go repository — which keeps the "minimal test setup scoped to the ladder directory" intent intact. The README documents the plain `python -m pytest` form as the equivalent for a machine that already has pytest and PyYAML.
- **Applies to:** batches 1 through 6

### Decision: the run model is an operator prerequisite between batch 7 and batch 8

- **Decision:** `ladder.yaml` ships with `run_model: null`, and card 24 refuses to start the matrix while it is unset, halting with an explicit instruction naming the field. The operator sets it once, before batch 8 runs.
- **Rationale:** the discussion is explicit that the model id is deliberately not fixed in the design and that the harness must refuse to start without it. The consequence for this plan is that batch 8 is not autonomously reachable: an orchestrator executing batches 1-7 will halt at card 24 until the operator edits `ladder.yaml`. That is the intended behavior — silently picking a model would make the whole matrix unreproducible — and it is called out here so the halt reads as the designed gate rather than as a harness fault.
- **Applies to:** batch 8, batch 9

### Decision: batches 8 and 9 are execution, and cost real money and wall-clock

- **Decision:** batch 8 dispatches 1 probe run plus 45 benchmark runs plus 45 scoring calls, sequentially, and batch 9 writes prose from their results. Cards 25 and 26 carry `Commit: none` because their entire product lands under the gitignored `results/**/raw/` tree.
- **Rationale:** the discussion makes the completed matrix and the written conclusion part of the deliverable, not a follow-up. Sequential dispatch is required by the timing-metric decision, so this is many hours of wall-clock and a real per-run cost. The harness's own resumability (a `run.json` with `state: "complete"` per run) is what makes an interrupted batch 8 safe to re-enter rather than a restart.
- **Applies to:** batch 8, batch 9

### Decision: dated results directory

- **Decision:** the tracked results directory is `bench/loomyard-eval/ladder/results/<YYYY-MM-DD>/`, where the date segment is the UTC date on which the matrix run started. Every literal path in batches 8 and 9 spells it `2026-08-29`; the implementer substitutes the actual start date if the matrix runs on a different day, and uses that same directory for `probe.json`, `summary.json`, `conclusion.md`, and `raw/`.
- **Rationale:** a plan cannot know the execution date, but the validator and the reviewer need concrete paths. Pinning one literal and stating the substitution rule keeps both true. Re-running the matrix on a different model or a later date is a new dated directory, never a mixed one.
- **Applies to:** batch 8, batch 9

### Decision: Python style for the suite

- **Decision:** all six scripts target the system Python 3 already present (3.14), use only the standard library plus `PyYAML`, carry module-level docstrings in the shape `bench/loomyard-eval/scripts/gen_compact_toc.py` already uses (summary line, then a `Usage:` block), guard every executable entry point behind `if __name__ == "__main__":`, and expose their deterministic units as module-level functions so the tests import them without triggering any dispatch.
- **Rationale:** the repo has exactly one existing Python script and no packaging; matching its conventions keeps the suite legible beside it. Standard-library-plus-PyYAML is what the `uv run --with` invocation provisions, so no dependency drifts in unnoticed.
- **Applies to:** batches 1 through 6

## All Files Touched

- `.gitignore`
- `bench/loomyard-eval/ladder/README.md`
- `bench/loomyard-eval/ladder/ladder.yaml`
- `bench/loomyard-eval/ladder/results/2026-08-29/cold_cell.json`
- `bench/loomyard-eval/ladder/results/2026-08-29/conclusion.md`
- `bench/loomyard-eval/ladder/results/2026-08-29/probe.json`
- `bench/loomyard-eval/ladder/results/2026-08-29/summary.json`
- `bench/loomyard-eval/ladder/scripts/extract_usage.py`
- `bench/loomyard-eval/ladder/scripts/gates.py`
- `bench/loomyard-eval/ladder/scripts/ladder_config.py`
- `bench/loomyard-eval/ladder/scripts/run_ladder.py`
- `bench/loomyard-eval/ladder/scripts/score_run.py`
- `bench/loomyard-eval/ladder/scripts/summarize.py`
- `bench/loomyard-eval/ladder/tests/conftest.py`
- `bench/loomyard-eval/ladder/tests/fixtures/bundle-mixed-tools.jsonl`
- `bench/loomyard-eval/ladder/tests/fixtures/cold-native-fallback.jsonl`
- `bench/loomyard-eval/ladder/tests/fixtures/denied-attempt.jsonl`
- `bench/loomyard-eval/ladder/tests/fixtures/errored-tool-result.jsonl`
- `bench/loomyard-eval/ladder/tests/fixtures/none-target-origin-mention.jsonl`
- `bench/loomyard-eval/ladder/tests/fixtures/targetdir-override.jsonl`
- `bench/loomyard-eval/ladder/tests/fixtures/zero-tool-calls.jsonl`
- `bench/loomyard-eval/ladder/tests/test_extract_usage.py`
- `bench/loomyard-eval/ladder/tests/test_gates.py`
- `bench/loomyard-eval/ladder/tests/test_ladder_config.py`
- `bench/loomyard-eval/ladder/tests/test_run_ladder.py`
- `bench/loomyard-eval/ladder/tests/test_score_run.py`
- `bench/loomyard-eval/ladder/tests/test_summarize.py`
