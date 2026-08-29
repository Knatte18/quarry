# Batch: suite-skeleton-and-generation

```yaml
task: "Per-capability quarry-mcp benchmark suite"
batch: "suite-skeleton-and-generation"
number: 1
cards: 5
verify: uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests/test_ladder_config.py -q
depends-on: []
```

## Batch Scope

This batch lays down the ladder suite's tree and its single declarative source of truth, then implements every deterministic unit derived from that source: config loading and validation, per-config `permissions.deny` and settings-file generation, and per-rung prompt-preamble generation. It is one batch because all three generation units read the same `ladder.yaml` schema and are meaningless without it — splitting them would force the schema to be re-established three times.

The external interface the later batches consume is `ladder_config.py`'s module-level functions: `load_ladder`, `LadderConfig`, `deny_list_for`, `settings_document_for`, and `preamble_for`. Batches 2 through 6 import from this module and never re-derive a deny-list or a preamble themselves.

Batch-local decision: there are two places the seven tool names appear, with distinct jobs. `QUARRY_TOOLS` in `ladder_config.py` is the **validation** constant — the canonical set `load_ladder` checks the ladder file's own `quarry_tools:` against, so a ladder that drifts from quarry's real surface is rejected at load. `deny_list_for` derives from the **loaded** `Ladder.quarry_tools`, which is what makes the drift guard meaningful and what a genuine eighth tool would flow through. Either way no file in this suite ever writes a deny-list literal: every deny-list is `<the ladder's tool set> - allowed`.

## Cards

### Card 1: untrack raw run artifacts and bootstrap the pytest path

- **Context:**
  - `bench/loomyard-eval/scripts/gen_compact_toc.py`
- **Edits:**
  - `.gitignore`
- **Creates:**
  - `bench/loomyard-eval/ladder/tests/conftest.py`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Append a new block to the repo `.gitignore`, placed after the existing `.scratch/` line and before the `# === mill-managed` marker block, containing a two-line comment explaining that per-run ladder artifacts are disposable once summarised while `conclusion.md` and `summary.json` beside them stay tracked, followed by the single pattern `bench/loomyard-eval/ladder/results/**/raw/`. Do not touch the mill-managed block. Create the conftest with a module docstring stating it puts the suite's `scripts` directory on `sys.path` so the tests import the harness modules by bare name, and a body that computes `SCRIPTS_DIR = Path(__file__).resolve().parent.parent / "scripts"` and calls `sys.path.insert(0, str(SCRIPTS_DIR))` at import time, guarded so a repeated insert cannot stack duplicates.
- **Commit:** `chore(bench): untrack ladder raw artifacts and add ladder pytest bootstrap`

### Card 2: declare the ladder in `ladder.yaml`

- **Context:**
  - `bench/loomyard-eval/tasks/01-reed-geometry-exploration.md`
  - `bench/loomyard-eval/tasks/04-shedadapters-shuttle-impact.md`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/ladder.yaml`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Write the single declarative source of truth for the whole matrix. Top-level keys:
  - `run_model: null` — the pinned model id for all 45 runs, set by the operator before the matrix starts.
  - `reps: 3`.
  - `max_turns: 60` — the per-run turn ceiling, identical across all 45 runs and the probe, so it bounds cost without biasing any comparison. 60 sits comfortably above the turn counts the sibling suite's runs actually reached on these two tasks, so it bounds a run that wanders without truncating one that is working; unlike `run_model` it is a harness parameter rather than a measurement choice, so it is pinned here instead of being left to the operator.
  - `scorer:` — a mapping with `model: claude-opus-5` and `effort: high`, pinned for the whole matrix, both confirmed accepted by the installed client. The prompt template is not pinned here: it is selected per task from the task's own `schema` field, `exploration` or `impact`. A drifting scorer would shift correctness numbers between rungs graded at different times, so these two values are concrete rather than left blank.
  - `quarry_tools:` — the canonical seven client-side names, in this exact order: `toc_dir`, `toc_file`, `textDocument_definition`, `textDocument_references`, `workspace_symbol`, `impact`, `assert_no_callers`. These are bare tool names; the `mcp__quarry__` prefix is applied by `ladder_config.py`, never spelled here.
  - `tasks:` — a mapping keyed by task slug (`01-reed-geometry-exploration`, `04-shedadapters-shuttle-impact`), each carrying `task_file`, `pinned_sha` (`975578cda8d6f3a81580bd4e73725e060211b766` for both), `worktree` (`/tmp/loomyard-eval-01` and `/tmp/loomyard-eval-04`), `schema` (`exploration` / `impact`), and `fasit` (the tracked `c.json` path under `bench/loomyard-eval/results/2026-08-28/`).
  - `source_repo: /home/knatte/Code/loomyard/wts/loomyard`.
  - `configs:` — a list of 15 entries. Each carries `id`, `ladder` (`a` or `b`, and `a` for the cold cell), `task` (a key of `tasks:`), `allowed` (a list drawn from `quarry_tools`), and `cold` (`false`, except `true` for the cold cell). The 15 ids and allowed sets are exactly the discussion's two tables plus the cold cell: `a0-none` (empty), `a1-toc-file` (`toc_file`), `a2-toc-dir` (`toc_dir`), `a3-toc-pair` (`toc_dir`, `toc_file`), `a4-toc-pair-symbol` (`toc_dir`, `toc_file`, `workspace_symbol`), `a5-bundle` (all seven), `b0-none` (empty), `b1-symbol` (`workspace_symbol`), `b2-definition` (`textDocument_definition`), `b3-references` (`textDocument_references`), `b4-lsp-trio` (`workspace_symbol`, `textDocument_definition`, `textDocument_references`), `b5-impact` (`impact`), `b6-assert-no-callers` (`assert_no_callers`), `b7-bundle` (all seven), and `a5-bundle-cold` (all seven, `cold: true`, task `01-reed-geometry-exploration`). Ladder-A configs use task 01, Ladder-B configs use task 04.
  - `cold_worktree_template: /tmp/loomyard-eval-01-cold-{n}` — the per-cold-run distinct path, with `{n}` the 1-based repetition index.

  The cold entry carries one further field the other 14 do not: `warm_counterpart: a5-bundle`, naming the warm cell it is contrasted against. The warm-versus-cold comparison resolves its warm side through this declared field, exactly as a rung resolves its control through the `ladder` field, so no comparison in this suite is ever built by parsing an id string.

  Carry a header comment stating that this file is the only place the config table lives, that every deny-list is derived from `allowed` rather than written out, and that `run_model` must be set before the matrix starts.
- **Commit:** `feat(bench): declare the quarry-mcp capability ladder`

### Card 3: load and validate the ladder

- **Context:**
  - `bench/loomyard-eval/ladder/ladder.yaml`
  - `bench/loomyard-eval/ladder/tests/conftest.py`
  - `bench/loomyard-eval/scripts/gen_compact_toc.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/tests/test_ladder_config.py`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Write the tests first, then the module. Define the module-level constant `QUARRY_TOOLS` as the canonical seven bare names, `MCP_PREFIX = "mcp__quarry__"`, and `DAEMON_BACKED_TOOLS` — the five canonical names excluding `toc_file` and `toc_dir`, which are tree-sitter-backed and never start a daemon. Add a helper `mcp_name(tool)` returning the prefixed client-side name. Define a frozen dataclass `LadderConfig` with fields `id`, `ladder`, `task`, `allowed` (a tuple), `cold`, and `warm_counterpart` (a config id or `None`). Define `load_ladder(path)` returning a `Ladder` dataclass carrying `run_model`, `reps`, `max_turns`, `scorer`, `quarry_tools`, `tasks`, `source_repo`, `cold_worktree_template`, and `configs` (a tuple of `LadderConfig`). Define `config_by_id(ladder, config_id)` and `control_for(ladder, config)` — the latter returning the config whose `ladder` field matches and whose `allowed` is empty, so a rung resolves its own ladder's control by field lookup and never by parsing its id string. Define `warm_counterpart_for(ladder, config)` returning the config named by a cold config's `warm_counterpart` field, resolved through `config_by_id`, so the warm-versus-cold contrast resolves its warm side by declaration rather than by an id-suffix convention. Define `require_pins(ladder)` raising `LadderConfigError` naming the offending field and the ladder file when any pinned value the matrix depends on is unset: `run_model`, `max_turns`, `scorer.model`, or `scorer.effort`. Only `run_model` ships null and is the operator's to set; the other three ship with values, and the check exists so an edit that blanks one fails before the matrix starts rather than reaching `--model`/`--max-turns` as a null.

  `load_ladder` raises `LadderConfigError` on: a duplicate `id`; an `id` that is not unique across both ladders; a `ladder` value outside `a`/`b`; a `task` key absent from `tasks:`; an `allowed` entry not in `quarry_tools`; a `quarry_tools` list that is not exactly the canonical seven; a ladder with no config whose `allowed` is empty; a ladder with **more than one** config whose `allowed` is empty, which would leave `control_for`'s result undefined; more than one `cold: true` config; a `warm_counterpart` on a config that is not cold; a cold config with no `warm_counterpart`; and a `warm_counterpart` naming an unknown id, a cold config, or the cold config itself. Tests cover each raise, plus a successful load asserting 15 configs, `control_for` resolving `a0-none` for `a5-bundle` and `b0-none` for `b5-impact` and never crossing ladders, `warm_counterpart_for` resolving `a5-bundle` for the cold config, and `require_pins` raising while `run_model` is null, raising independently for a blanked `max_turns`, `scorer.model`, and `scorer.effort`, and passing once all four carry values.
- **Commit:** `feat(bench): load and validate the capability ladder definition`

### Card 4: derive per-config deny-lists and settings documents

- **Context:**
  - `bench/loomyard-eval/ladder/ladder.yaml`
- **Edits:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/tests/test_ladder_config.py`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Write the tests first. Add `deny_list_for(ladder, config)` returning the sorted list of client-side `mcp__quarry__*` names for every canonical tool **not** in that config's `allowed` set, always prefixed via `mcp_name` and never assembled from a literal. Add `settings_document_for(ladder, config)` returning the full settings mapping a run is launched with: `permissions.allow` exactly `["Read", "Grep", "Glob", "Bash"]`, and `permissions.deny` the config's quarry deny-list plus the literal `"Task"`, sorted with `"Task"` included in the sort. Add `write_settings(ladder, config, path)` serialising that mapping as JSON.

  Tests assert: `a0-none` and `b0-none` deny all seven quarry names; `a5-bundle` and `b7-bundle` deny no quarry name; `b5-impact` denies exactly six; every generated deny-list contains `"Task"` and the allow-set is identical across all 15 configs; no non-quarry name other than `"Task"` ever appears in a deny-list; and the drift guard — mutating a **loaded** `Ladder`'s `quarry_tools` to append a fabricated eighth name makes that name appear in every restricted config's deny-list with no per-config edit, asserted directly rather than implied. The mutation is post-load on purpose: `load_ladder` rejects a `quarry_tools` list that is not exactly the canonical seven, so the guard cannot be expressed through the real file path, and adding a genuine eighth tool to quarry would require relaxing that canonical-seven check as well as editing the ladder file. What the guard proves is the derivation — that no deny-list is written per config — not that the loader accepts eight tools.
- **Commit:** `feat(bench): derive per-config deny-lists and run settings from the ladder`

### Card 5: generate per-rung preambles

- **Context:**
  - `bench/loomyard-eval/README.md`
  - `bench/loomyard-eval/tasks/01-reed-geometry-exploration.md`
  - `bench/loomyard-eval/tasks/04-shedadapters-shuttle-impact.md`
  - `bench/loomyard-eval/ladder/ladder.yaml`
- **Edits:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/tests/test_ladder_config.py`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Write the tests first. Add module-level constants holding the two shared blocks copied byte-for-byte out of the committed benchmark README: `PARALLEL_OPENING` (the `USE PARALLEL TOOL CALLS.` paragraph that opens both committed preambles) and `PARALLEL_BLOCK` (the whole `<use_parallel_tool_calls>` element that closes both). Add `B_PREAMBLE_BODY`, the committed Agent B preamble's own body between those two blocks, copied verbatim and containing no occurrence of the word `quarry`. It **ends at the `<TARGET_DIR>` paragraph** and excludes the committed template's own `<TASK TEXT>` placeholder line: `preamble_for` appends the task text itself, so carrying the placeholder through would emit both the placeholder and the text into every control run's prompt.

  Add `preamble_for(ladder, config, target_dir, task_text, schema_json)` returning the full prompt string for one run:
  - When `config.allowed` is empty, the prompt is the committed B preamble reproduced exactly — `PARALLEL_OPENING`, `B_PREAMBLE_BODY` with `<TARGET_DIR>` substituted, the task text, `PARALLEL_BLOCK`, the closing "end your final message with ONLY a fenced json code block" sentence, and the schema.
  - Otherwise the prompt is a freshly generated MCP-shaped A preamble: `PARALLEL_OPENING`, then a body that names the rung's allowed tools by their `mcp__quarry__*` client-side names and nothing else, describes each as a tool call taking a call-wide input with a `targets` array, states that `targetDir` and `buildTags` must never be set because the server is already rooted at the correct target, carries over the three exposure-independent instructions from the committed A template (prefer these tools over grep, do not re-verify one of their answers with grep, and pass a known `file:line:character` position rather than a bare symbol name when one is already in hand), then the task text, `PARALLEL_BLOCK`, the same closing sentence, and the schema. It must contain no binary path, no shell verb syntax, and no `--`-prefixed flag.

  Tests assert, per rung: the generated prompt names exactly that config's allowed tools under the `mcp__quarry__` prefix and no other quarry tool name; it contains none of the literal strings "/tmp/quarry-bench", "quarry toc dir", "quarry refs", or "--target-dir"; it forbids setting `targetDir` and `buildTags`; and it contains `PARALLEL_OPENING` and `PARALLEL_BLOCK` verbatim. A separate assertion covers every config whose `allowed` is empty: its generated prompt contains no occurrence of `quarry` in any casing, anywhere.
- **Commit:** `feat(bench): generate per-rung MCP preambles and the unchanged control preamble`

## Batch Tests

`verify:` runs `bench/loomyard-eval/ladder/tests/test_ladder_config.py`, which is the only test file this batch creates and covers every unit it ships: ladder loading and its validation raises, control resolution, deny-list derivation including the eighth-tool drift guard, settings-document shape, and preamble generation for both the quarry rungs and the `none` controls. Cards 3, 4, and 5 each write their tests before their implementation, per the discussion's TDD direction for the deny-list and preamble units. The scope is one file because nothing else exists yet; no other test file in the suite is affected by this batch.
