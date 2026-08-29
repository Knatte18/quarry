# quarry-mcp capability ladder

Self-contained orchestration protocol. A fresh agent with no memory of how
this was designed should be able to execute this end to end from this file
alone. Do not skip steps because they seem obvious — the design has specific
rationale behind each constraint (see "Design rationale" at the bottom);
deviating from it quietly invalidates the result.

## The question

Which specific capabilities inside quarry's MCP surface — exposed to a
Claude Code agent as `mcp__quarry__*` tools, not as a CLI — actually earn
their keep, attributed one tool (or small tool group) at a time rather than
as one undifferentiated bundle? This suite does **not** benchmark the
`quarry` CLI, and it builds no CLI binary for any run: every run in the
matrix launches the `quarry-mcp` server binary and talks to it only through
the client-side `mcp__quarry__*` tool names an MCP-integrated agent actually
sees.

## What this fixes

Three methodology defects in the sibling `bench/loomyard-eval` suite this
one exists to correct, without touching that suite's own committed results:

- **Single-bundle exposure that cannot attribute a result to any tool.**
  Agent A there gets all seven quarry verbs at once; a win or a loss is a
  verdict on "quarry" as a whole, never on `toc_file` versus `impact`
  versus `workspace_symbol` specifically.
- **Orchestrator-executed runs that were retracted as contaminated.** A run
  dispatched by an orchestrating agent that had already seen other context
  cannot be trusted as a blind measurement. This suite's runs are each an
  independent `claude -p` subprocess the harness launches directly.
- **One run per arm.** A single run per config is a temperature check, not
  a result — one bad or lucky run silently becomes the whole verdict. This
  suite runs `reps: 3` repetitions of every one of its 15 configs.

## The two ladders

Reproduced exactly as `bench/loomyard-eval/ladder/ladder.yaml` declares
them — **`ladder.yaml` is the single source of truth**; if either table
below and the file ever disagree, the file wins and this README is stale.
Every config id is ladder-qualified (`a...`/`b...`) and globally unique
across both ladders.

**Ladder A — task `01-reed-geometry-exploration` (exploration, `toc`
family):**

| id | allowed tools |
| --- | --- |
| `a0-none` | (none — control) |
| `a1-toc-file` | `toc_file` |
| `a2-toc-dir` | `toc_dir` |
| `a3-toc-pair` | `toc_dir`, `toc_file` |
| `a4-toc-pair-symbol` | `toc_dir`, `toc_file`, `workspace_symbol` |
| `a5-bundle` | all seven: `toc_dir`, `toc_file`, `textDocument_definition`, `textDocument_references`, `workspace_symbol`, `impact`, `assert_no_callers` |

**Ladder B — task `04-shedadapters-shuttle-impact` (impact analysis, LSP
family):**

| id | allowed tools |
| --- | --- |
| `b0-none` | (none — control) |
| `b1-symbol` | `workspace_symbol` |
| `b2-definition` | `textDocument_definition` |
| `b3-references` | `textDocument_references` |
| `b4-lsp-trio` | `workspace_symbol`, `textDocument_definition`, `textDocument_references` |
| `b5-impact` | `impact` |
| `b6-assert-no-callers` | `assert_no_callers` |
| `b7-bundle` | all seven (same set as `a5-bundle`) |

**The cold cell** — `a5-bundle-cold`: the same tool set as `a5-bundle`,
`cold: true`, `warm_counterpart: a5-bundle`. It runs against a fresh,
per-repetition disposable worktree (`cold_worktree_template`) so no prior
run has already started the daemon for that path, and is contrasted against
`a5-bundle` only through its declared `warm_counterpart` field — never by
parsing the `-cold` suffix off its id.

15 configs total: 6 (ladder A) + 8 (ladder B, including `b0`) + 1 (cold) =
15; 3 repetitions each = 45 runs, plus the one preflight probe run described
under Enforcement.

## Enforcement

- **The deny-list is derived, never hand-written.** `ladder_config.py`'s
  `deny_list_for` computes, for a given config, every canonical tool in
  `quarry_tools` **not** in that config's `allowed` set, prefixed to its
  `mcp__quarry__*` client-side name. No config's deny-list is a literal
  anywhere in this suite.
- **An empty `allowed` set gets no MCP config at all.** `write_run_inputs`
  writes `mcp.json` only when `config.allowed` is non-empty. A `none`
  control config is launched with no `--mcp-config` flag whatsoever — the
  quarry server is never declared to it, because a declared server named
  "quarry" exposing an `mcp__quarry__*` namespace would itself be the
  structural leak the blinding forbids.
- **`permissions.allow` prevents prompting; it does not restrict.** The
  generated settings' `permissions.allow` (`Read`, `Grep`, `Glob`, `Bash`)
  exists solely to keep a headless run from blocking on a permission
  prompt. It is never treated as an allowlist anywhere in this suite, and
  no claim here rests on it bounding the toolset — established empirically
  while planning: a run launched with `permissions.allow` set to
  `["Read","Grep","Glob"]` under `--permission-mode dontAsk` still executed
  a `Bash` call successfully, with an empty `permission_denials` array in
  its result envelope. The only structural restriction is
  `permissions.deny`.
- **`Task` is denied uniformly in all 45 runs.** Every generated settings
  file — every `none` control and every quarry rung alike — carries `Task`
  in `permissions.deny` alongside that config's quarry deny-list. Without
  `--forward-subagent-text` a subagent's own tool calls never reach the
  captured `stream-json` transcript, so a run that dispatched `Task` would
  report a falsely low `tool_uses` and an incomplete `tool_uses_breakdown`
  — the same self-report-style undercount the transcript-extraction
  decision below exists to eliminate, one layer down. Denial is uniform
  across all 45 runs, so it cannot bias any comparison.
- **`permissions.deny`'s effectiveness is established once, before any
  paid matrix run**, by the preflight probe (`run_probe` in
  `run_ladder.py`): a throwaway `claude -p` run that declares the quarry
  server with `mcp__quarry__impact` denied, asks the agent to call it and
  report exactly what happened, and records `denial_blocks` (whether the
  denied call failed to succeed) into `probe.json`. `run_probe` raises and
  halts the matrix before a single paid run when `denial_blocks` is False
  — every rung would otherwise silently be the full bundle.
- **Blinding is enforced by transcript detection, not by construction.**
  `gates.gate_blinding`, applied only to a config whose `allowed` is empty,
  is fatal when the transcript contains an `mcp__quarry__` tool name, the
  literal `/tmp/quarry-bench`, or a filesystem path into `repo_root`. A
  bare "quarry" mention confined inside a `tool_result` payload is
  non-fatal (the target codebase mentions quarry in its own tracked files,
  so a bare-string gate there would wrongly halt the matrix); a "quarry"
  mention found anywhere else in the transcript is fatal. **State this
  plainly: the `none` arms' blinding is weaker than a structural
  guarantee** — it is a post-hoc detector over what the agent actually
  said and called, not a proof that no signal could possibly have reached
  it.

## Run environment

Every run in the 45-run main matrix, the 3-run cold cell, and the single
preflight probe shares:

- **The pinned model** — `ladder.yaml`'s `run_model`, identical across
  every run (see "How to run" below for why it starts `null`).
- **The identical allow-set** — `permissions.allow: ["Read", "Grep",
  "Glob", "Bash"]` on every run regardless of config, per Enforcement
  above.
- **`--setting-sources ""`** on every `claude -p` invocation this suite
  makes (matrix runs, the preflight probe, and every scoring call) —
  neutralises the operator's own user-level settings, which would
  otherwise layer a second, unaudited permissions/hooks source under the
  generated `--settings` file.
- **`--strict-mcp-config`** on the same set of invocations — the matching
  guard for MCP servers: without it a `none` run could inherit an ambient
  quarry server declaration and observe the `mcp__quarry__*` namespace,
  defeating the blinding.
- **The non-interactive permission mode** — `--permission-mode dontAsk` on
  every dispatch and every scoring call, so a denied or ambiguous tool call
  never blocks on a prompt no one is there to answer.
- **`QUARRY_STATE_DIR` and `QUARRY_BUILD_TAGS` both cleared** from every
  subprocess environment (`run_env()` in `run_ladder.py`). Each would move
  the resolved state directory off the per-path key the cold cell's whole
  argument rests on: the first two take precedence over `workspaceKey`
  outright, and a non-empty tag set appends a `tags-<hex>` segment at every
  tier. A cold cell whose worktree-keyed state directory can be silently
  overridden by either variable cannot support a "this daemon is fresh
  because this path never ran before" claim.
- **`QUARRY_CONFIG` deliberately left in place.** It selects the
  `servers.yaml` overlay naming the language-server command; clearing it on
  a machine that needs an overlay would stop the server from starting at
  all. It is recorded verbatim into `probe.json`'s `quarry_config` field so
  its value is part of the record, not scrubbed from the runs themselves.
- **`--state-dir` never passed.** The suite always resolves the state
  directory through the scrubbed environment, never a launch flag —
  `build_argv`'s docstring states this explicitly, and there is no code
  path in this suite that adds it.
- **The per-run turn ceiling** — `--max-turns` from `ladder.yaml`'s
  `max_turns`, identical across all 45 runs and the probe. A run that hits
  it is `"truncated"`, never retried, and halts the whole matrix on its
  first occurrence (see `run_matrix`'s docstring) — the ceiling is a
  matrix-wide constant a second attempt would hit identically.
- **The per-run daemon warm-up**, for the main matrix only. `warm_daemon`
  runs one `workspace_symbol` call against the target worktree immediately
  before each main-matrix run's dispatch, then asserts a `daemon.json` now
  exists at the resolved state directory. This runs per run, not once per
  worktree, since the daemon self-expires after its own idle timeout;
  it is never called for the cold config.

## Metrics

Every field a run's `usage.json` carries (`extract_usage.py`,
`extract_usage`):

- `duration_ms` — the result event's own reported duration.
- `wall_clock_ms` — harness-measured, start-to-finish of the subprocess.
- `tokens` — `input_tokens`, `output_tokens`, `cache_read_input_tokens`,
  `cache_creation_input_tokens`, each summed independently across every
  `type: "assistant"` event's `message.usage` object — never derived from
  another class, and never read off the result envelope (see the plan's
  "token classes are summed from assistant events" decision: the result
  envelope's own `usage` is a final-iteration-only view, observed to report
  `input_tokens: 4` against an `iterations` array covering the whole run).
- `result_usage` — the result event's own `usage` object, recorded verbatim
  for cross-checking only; no reported per-class figure is ever taken from
  it.
- `cost_usd` — the result event's `total_cost_usd`.
- `num_turns` — the result event's `num_turns`.
- `tool_uses` / `tool_uses_breakdown` — the total and per-tool-name count of
  every `tool_use` block across every assistant event.
- `quarry_tool_uses` — the subset of `tool_uses_breakdown` whose names start
  with `mcp__quarry__`.
- `bash_grep_count` — **Bash-only**: the count of `Bash` tool calls whose
  `command` string invokes `grep` or `rg` as a leading command word (not
  merely containing the substring "grep" somewhere unrelated, e.g. inside a
  path), matching `#006`'s own Bash-transcript-grep definition exactly.
- `grep_tool_count` — a **separate** count: every call to the dedicated
  `Grep` tool. `bash_grep_count` and `grep_tool_count` are never merged
  into one counter; `grep_fallback_total` is their sum and is reported
  alongside both, never substituted for either.
- `denied_tool_attempts` — the length of the result event's
  `permission_denials` array.
- `result_subtype`, `result_is_error`, `advertised_tools`, `model`,
  `session_id`, `transcript` — recorded for diagnosis and drift-checking.

**A metric is extracted from a transcript and never from an agent's
self-report.** Nothing in `usage.json` is copied from the answer JSON block
the agent itself produced; every field above is either computed from parsed
transcript events or taken from the client's own result envelope.

## Scoring

`score_run.py` dispatches a **blinded, three-input contract**: the scoring
call sees exactly the redacted answer, the task's `_meta`-stripped fasit,
and the task text — never the config id, the ladder, the allowed-tool list,
the transcript, or any other run's answer. The fasit itself is left
verbatim (one fixed file, identical across all rungs of its task, so it
cannot leak which config is being graded); only its `_meta` block is
stripped, since that block's `role`/`see_also` text names quarry and
`scorecard.md` by name.

**Redaction** (`redact_text`/`redact_answer`) strips every trace of tool
provenance from an answer's free-text fields before it reaches the scorer:
every `mcp__quarry__*` client-side name, every bare canonical tool name
except `impact`, the word "quarry" itself, and CLI-invocation forms (the
literal `/tmp/quarry-bench` path, a `quarry <verb>` shell form, and a
`--target-dir`-style flag) — all case-insensitive, all collapsed to a
single `<tool>` token when adjacent. `impact` is deliberately excluded from
the bare-name pass because it is also an ordinary English word every
Ladder-B answer's prose legitimately uses; only its `mcp__quarry__impact`
and CLI-invocation forms are redacted. **Acknowledged residual limit:**
this is a pattern-based redaction over free text, not a structural
guarantee — a sufficiently indirect phrasing of tool provenance could in
principle survive it, the same class of limit the blinding gate above
carries for the `none` arms.

**The exploration rule** (task 01) is reused unchanged from the sibling
suite: recall = (the fasit's `relevant_files`/`key_symbols` also present in
the answer's) / (the fasit's total); precision = (the answer's entries
corroborated by the fasit) / (the answer's total); plus a qualitative
judgment of whether the answer's `summary` describes the same actual
mechanism the fasit found, not just whether file names overlap.

**The impact rule** (task 04), in full: recall = (the fasit's
`callers_to_update` entries matched on file AND line — a line must denote
the same call site, not merely the same file — also present in the
answer's) / (the fasit's total); precision = (the answer's
`callers_to_update` entries corroborated by the fasit) / (the answer's
total). `decoy_admitted` is true when the answer's `callers_to_update`
contains a call site the fasit lists under `excluded_lookalikes` — this is
**reported separately, never folded into precision** — it is a materially
worse mistake than a missed real caller, since it means the agent would
have shipped a broken edit to an unrelated interface (the `burler.go:373`
decoy). `lookalikes_matched` is the count of the answer's
`excluded_lookalikes` the fasit also names — **credited, never required**:
an answer naming none loses no points for it.

## Reporting discipline

- **Medians and full ranges**, never a mean alone — every `summarize.py`
  cell reports `median`/`min`/`max`/`n` per metric (`summarise_cell`).
- **A disjoint-range bar applies to all three comparison types**
  (`ranges_disjoint`): a comparison is `separated` only when the two sides'
  `(min, max)` ranges do not overlap at all — ranges that merely touch at a
  single shared value are **not** disjoint.
- **The grep metrics (`bash_grep_count`, `grep_tool_count`,
  `grep_fallback_total`) are excluded from every rung-vs-control
  comparison** (`compare_rung_to_control`, via `GREP_METRICS`) — the
  control's preamble differs in steering as well as tools, so a grep-usage
  gap between it and a rung cannot be attributed to the tool exposure
  alone. They remain eligible for rung-vs-rung comparisons, where the
  preambles are identical except for the tool list.
- **`n = 3` stated explicitly** on every cell (`reps: 3` in `ladder.yaml`,
  carried into every stats record's `n` field) — never silently implied.
- **No significance claims.** This suite reports disjoint or overlapping
  ranges at `n = 3`; it does not run or report a statistical significance
  test of any kind, and no comparison output should be read as one.

## Hard rule on cross-suite comparison

**Correctness** (recall/precision/summary-match/decoy-admitted/lookalikes)
may be compared to the sibling `bench/loomyard-eval` suite's committed
results, since both suites score against the same fasit-style ground truth
and the same rubric shape.

**Duration, tokens, cost, turns, tool counts, and grep counts may not be
compared across the two suites — not even informally.** The sibling suite
dispatches its three agents as parallel sub-agent calls inside an
orchestrating session; this suite dispatches each run as an independent
`claude -p` subprocess with a scrubbed environment, no shared session
context, and a different prompt shape per rung (an MCP-tool preamble here
versus the sibling's CLI-tool preamble). Any of those differences alone is
enough to move every cost-shaped number by an amount this suite cannot
separate from a genuine capability effect — so cost-shaped numbers stay
suite-local.

## How to run

**Operator prerequisite — set `run_model` first.** `ladder.yaml` ships with
`run_model: null`. `require_pins` refuses to start the matrix while it, or
`max_turns`, `scorer.model`, or `scorer.effort`, is unset — edit
`ladder.yaml` and set `run_model` to the pinned model id before invoking the
harness. This is deliberate: the model id is not fixed in the design, and
silently picking one would make the whole matrix unreproducible.

Then, from the repo root:

```
python bench/loomyard-eval/ladder/scripts/run_ladder.py \
    bench/loomyard-eval/ladder/ladder.yaml \
    bench/loomyard-eval/ladder/results/<YYYY-MM-DD> \
    --stage all
```

This builds and adopts the two disposable task worktrees, builds the
`quarry-mcp` server binary, runs the preflight probe (unless already
recorded passing), then the sequential 42-run main matrix with a
three-attempt-per-run cap, then the 3-run cold-daemon comparison cell last.
`--stage probe|main|cold` runs one stage in isolation; the default `all`
runs all three in order.

Then summarise:

```
python bench/loomyard-eval/ladder/scripts/summarize.py \
    bench/loomyard-eval/ladder/ladder.yaml \
    bench/loomyard-eval/ladder/results/<YYYY-MM-DD>
```

This writes `summary.json` into the same results directory and exits
non-zero (after naming the incomplete cells on stderr) if the matrix is not
yet fully complete.

**Re-invoking the harness resumes by skipping completed runs.**
`pending_runs` filters every (config, repetition) pair through
`gates.is_complete`, which is true only when that run directory's
`run.json` exists and parses with `state: "complete"` — an interrupted
invocation is always safe to re-run into the same results directory.

## How to test

From the repo root:

```
uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests -q
```

A machine that already has `pytest` and `PyYAML` installed can instead use
the plain form:

```
python -m pytest bench/loomyard-eval/ladder/tests
```

## Layout

```
bench/loomyard-eval/ladder/
├── README.md                    (this file)
├── ladder.yaml                  the single declarative source of truth
├── scripts/
│   ├── ladder_config.py         loads/validates ladder.yaml; deny-list, settings, preamble derivation
│   ├── gates.py                 per-run validation-gate predicates
│   ├── run_ladder.py            harness entry point: worktrees, probe, main matrix, cold cell
│   ├── extract_usage.py         per-run usage.json extraction from a transcript
│   ├── score_run.py             blinded three-input scoring dispatch
│   └── summarize.py             summary.json: medians/ranges/disjoint-range comparisons
├── tests/                       tracked unit tests, one file per scripts/ module
│   ├── conftest.py
│   ├── fixtures/                tracked transcript/answer fixtures the tests read
│   ├── test_ladder_config.py
│   ├── test_gates.py
│   ├── test_run_ladder.py
│   ├── test_extract_usage.py
│   ├── test_score_run.py
│   └── test_summarize.py
└── results/<YYYY-MM-DD>/        one dated directory per matrix run
    ├── probe.json               tracked
    ├── summary.json             tracked
    ├── conclusion.md            tracked
    └── raw/                     gitignored (bench/loomyard-eval/ladder/results/**/raw/) —
                                  per-run transcript.jsonl, answer.json,
                                  answer.redacted.json, usage.json,
                                  score.json, run.json, settings.json, mcp.json
```

`ladder_config.py` and `gates.py` are importable helper modules, not
dispatch entry points: they exist alongside the four documented
entry-point scripts specifically so deny-list generation, preamble
generation, and every validation gate can be unit-tested as deterministic
units without importing the dispatch layer.

## Design rationale — do not "simplify" these away

- **Sequential dispatch, never concurrent.** `run_matrix` drives every
  pending main-matrix pair one at a time. Concurrent runs would contend
  for CPU, the shared daemon, and model-side rate limits, any of which
  would make wall-clock incomparable across configs.
- **The cold cell's per-run distinct worktrees and its supervised-connection
  assertion.** Each cold repetition builds a fresh worktree at
  `cold_worktree_template` with `{n}` substituted, clears the resolved
  state directory, and asserts `gate_cold_before` before every attempt
  (including retries) so a failed attempt's leftover `daemon.json` cannot
  fail the precondition deterministically. `gate_cold_after` then requires,
  when the transcript used at least one daemon-backed tool, that a
  `daemon.json` actually exists afterward — a daemon-backed call with no
  resulting state file means the native fallback was taken, on which path
  the shared daemon address is not a function of the state directory at
  all, and the run is invalidated rather than reported as cold. `toc_dir`
  and `toc_file` are excluded from the daemon-backed set entirely: their
  handlers reach the tree-sitter path directly and never `EnsureServer`, so
  a toc-only run carries no warmth signal and is reported as such
  (`cold_no_daemon_backed_call`), never silently read as "native fallback
  taken."
- **The cold cell runs last**, after the entire main matrix, and drains
  every daemon the main matrix left resident on both task worktrees before
  its first repetition — neither task daemon may compete with a cold run
  for CPU or, worse, be mistaken for the cold run's own daemon.
- **The hard worktree restore after every run, with dirtying recorded
  rather than gated.** `restore_worktree` (`git reset --hard` +
  `git clean -fdx` + re-neutralisation) runs unconditionally after every
  main-matrix attempt, successful or not. `observe_worktree_dirtied` is
  called *before* the restore, inside `run_gates`, specifically because an
  observation taken after the restore would read `False` for every run —
  the restore is precisely what erases the evidence. Dirtying is recorded
  as a non-fatal observation, never gated: an agent legitimately editing
  files while exploring is not itself invalid.
- **The three-attempt cap halts the whole matrix, not just the offending
  config.** `MAX_ATTEMPTS = 3`: on a `"failed"` outcome the run directory
  is invalidated and retried up to three attempts; on the third failure
  `run_matrix`/`run_cold_cell` raise and halt everything, leaving every
  already-completed run intact so a re-invocation resumes past them. A
  `"truncated"` outcome (the `--max-turns` ceiling reached) is never
  retried and halts on its first occurrence, since the ceiling is a
  matrix-wide constant a second attempt would hit identically.
- **The preamble steering confound, and what it forbids the conclusion from
  claiming.** A `none` control's preamble and a quarry rung's preamble
  differ in more than which tools are listed — the rung preamble also
  instructs the agent to prefer its listed tools over grep and not to
  double-check them. That steering difference is why grep-usage metrics are
  excluded from every rung-vs-control comparison (see Reporting discipline)
  and why any control-vs-rung reading of this suite's results must be
  understood as "tool exposure plus steering," never "tool exposure alone."
  The conclusion this suite's results feed must not claim to have isolated
  tool exposure from steering in a rung-vs-control comparison — only
  rung-vs-rung comparisons, where every preamble is identical except for
  the tool list, can support that stronger claim.
