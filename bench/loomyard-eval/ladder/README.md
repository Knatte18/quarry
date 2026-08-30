# quarry-mcp capability ladder

Self-contained protocol document for the Go harness. A fresh agent with no memory of how this was
designed should be able to execute this end to end from this file and the tracked
`.claude/skills/ladder-run/SKILL.md` alone. Do not skip steps because they seem obvious — the design has
specific rationale behind each constraint (see "Design rationale" at the bottom); deviating from it
quietly invalidates the result.

## The question

Which specific capabilities inside quarry's MCP surface — exposed to a Claude Code agent as
`mcp__quarry__*` tools, not as a CLI — actually earn their keep, attributed one tool (or small tool
group) at a time rather than as one undifferentiated bundle? This suite does **not** benchmark the
`quarry` CLI, and it builds no CLI binary for any run: every run in the matrix launches the `quarry-mcp`
server binary and talks to it only through the client-side `mcp__quarry__*` tool names an MCP-integrated
agent actually sees.

## What this fixes

Three methodology defects in the sibling `bench/loomyard-eval` suite this one exists to correct, without
touching that suite's own committed results:

- **Single-bundle exposure that cannot attribute a result to any tool.** Agent A there gets all seven
  quarry verbs at once; a win or a loss is a verdict on "quarry" as a whole, never on `toc_file` versus
  `impact` versus `workspace_symbol` specifically.
- **Orchestrator-executed runs that were retracted as contaminated.** A run dispatched by an orchestrating
  agent that had already seen other context cannot be trusted as a blind measurement. This suite's runs
  are each dispatched through the Agent Tool from a live, supervised Claude Code session — one run agent
  per dispatch, watched and killable on sight, never a headless subprocess the operator cannot see inside.
- **One run per arm.** A single run per config is a temperature check, not a result — one bad or lucky
  run silently becomes the whole verdict. This suite runs `reps: 3` repetitions of every one of its 15
  configs.

## The two ladders

Reproduced exactly as `bench/loomyard-eval/ladder/ladder.yaml` declares them — **`ladder.yaml` is the
single source of truth**; if either table below and the file ever disagree, the file wins and this README
is stale. Every config id is ladder-qualified (`a...`/`b...`) and globally unique across both ladders.

**Ladder A — task `01-reed-geometry-exploration` (exploration, `toc` family):**

| id | allowed tools |
| --- | --- |
| `a0-none` | (none — control) |
| `a1-toc-file` | `toc_file` |
| `a2-toc-dir` | `toc_dir` |
| `a3-toc-pair` | `toc_dir`, `toc_file` |
| `a4-toc-pair-symbol` | `toc_dir`, `toc_file`, `workspace_symbol` |
| `a5-bundle` | all seven: `toc_dir`, `toc_file`, `textDocument_definition`, `textDocument_references`, `workspace_symbol`, `impact`, `assert_no_callers` |

**Ladder B — task `04-shedadapters-shuttle-impact` (impact analysis, LSP family):**

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

**The cold cell** — `a5-bundle-cold`: the same tool set as `a5-bundle`, `cold: true`,
`warm_counterpart: a5-bundle`. It runs against a fresh, per-repetition disposable worktree
(`cold_worktree_template`) so no prior run has already started the daemon for that path, and is
contrasted against `a5-bundle` only through its declared `warm_counterpart` field — never by parsing the
`-cold` suffix off its id.

15 configs total: 6 (ladder A) + 8 (ladder B, including `b0`) + 1 (cold) = 15; 3 repetitions each = 45
runs, plus the two preflight probes described under Enforcement.

## Session model

**One session per repetition, for every config, running exactly one run agent and no scorer — plus one
shared scoring session for the whole matrix.** 45 run sessions + 1 scoring session = **46 sessions**.
`ladderbench prepare-session --config-id <id> --rep <n>` materialises a run session's scratch directory
and prints the launch command; `ladderbench prepare-session --scoring` materialises the single scoring
session instead.

**No session ever contains both a run agent and a scoring prompt, for any config.** The scorer prompt
(`ladderbench redact`) embeds the task's unstripped fasit, and a session's own transcript is a shared
surface: a session hosting several repetitions, or a run agent's dispatch alongside a later scoring call,
would put an earlier repetition's answer — and its task's fasit — within `Read` range of the agent
dispatched after it. This applies to every config, not only the blinded `none` controls: a rung's
recall/precision is the primary measured outcome this suite exists to compare, and closing the fasit
channel for the blinded arms alone would have fixed the less important half. The single scoring session
is safe to share across all 45 runs precisely because **the scorer agent definition grants zero tools** —
a scorer cannot read the session transcript accumulating the other runs' fasits, or anything else on
disk.

The orchestration protocol itself — the attempt loop, the scoring loop, retry and single-flight rules —
is not repeated in prose here; it lives in the tracked, versioned skill at
`.claude/skills/ladder-run/SKILL.md`, invoked as `/ladder-run` from inside each launched session. Keeping
one authoritative copy of the protocol, versioned alongside the binary whose subcommands it calls, is
what stops the two from drifting apart.

**Each session runs from a disposable scratch directory**, derived from `ladder.yaml`'s
`session_dir_template` (e.g. `/tmp/ladder-session-a5-bundle-1`). A run session's scratch directory holds
the generated `.mcp.json` (config permitting), `settings.json`, and the run agent definition only; the
scoring session's holds `settings.json` and the scorer definition only. Neither carries the other's, and
neither carries the tracked skill — the skill is installed once per session, at `~/.claude/skills/`
(user scope), never copied into the scratch directory, because the skill body names quarry throughout and
a `none` session's scratch directory is the blinded agent's own working directory.

## Enforcement

**Tool exposure is enforced two ways, with the allowlist as the structural primary and the deny-list as a
backup.**

- **The allowlist: a generated per-config agent definition.** Each config gets a Claude Code agent
  definition whose `tools:` frontmatter is an allowlist — `Read`, `Grep`, `Glob`, `Bash`, plus that
  config's `allowed` quarry tools under their `mcp__quarry__*` client-side names. The Agent Tool has no
  per-call `--settings` equivalent, so per-rung restriction has to be a static subagent-type definition.
  An allowlist is structurally stronger than a deny-list: the `none` arm never sees the `mcp__quarry__*`
  namespace at all, rather than seeing it and being denied. `Task` denial is structural too — an
  allowlist that omits `Task` means a run agent cannot spawn its own subagents.
- **The deny-list: the session's `settings.json`, as a backup layer.** Every run session's `settings.json`
  additionally carries `permissions.deny`: the derived quarry deny-list (every canonical tool not in
  `allowed`) plus `Task`, for every rung. This layer exists so a definition that fails to load does not
  silently promote a rung to the full bundle.
- **A blinded (`none`) session gets no server declaration at all.** `prepare-session` writes `.mcp.json`
  only when the config's `allowed` set is non-empty; a `none` config is launched with no `--mcp-config`
  flag whatsoever, and its `settings.json` denies `Task` and nothing else — the quarry deny-list backup
  guards nothing for a config that declares no server, so it is omitted rather than leaked into the
  blinded agent's own working directory. A declared server named "quarry" exposing an `mcp__quarry__*`
  namespace would itself be the structural leak the blinding forbids: this is what makes the claim "the
  `none` arm never sees the `mcp__quarry__*` namespace" literally true rather than merely
  allowlist-mediated — the namespace does not exist in that session.
- **The deny-list is derived, never hand-written.** `DenyListFor` computes, for a given config, every
  canonical tool in `quarry_tools` **not** in that config's `allowed` set, prefixed to its
  `mcp__quarry__*` client-side name. No config's deny-list is a literal anywhere in this suite.
- **Two preflight probes, not one, established before any paid run.** A single allowlist-only probe
  cannot distinguish "primary works, backup silently broken" from "primary broken, backup catching it" —
  the call fails identically either way, defeating the point of having a backup. So the preflight stage
  runs two throwaway dispatches, each recorded into `probe.json` (`ladderbench probe-record`):
  1. **Allowlist probe** — an agent definition granting `Read/Grep/Glob/Bash` but not
     `mcp__quarry__impact`, in a session where the quarry server *is* declared and `impact` is *not*
     denied. Records `allowlist_blocks`.
  2. **Deny-list probe** — an agent definition that *does* grant `mcp__quarry__impact`, in a session whose
     `settings.json` denies it. Records `denylist_blocks`.
  Either probe's `_blocks` field being false halts the matrix before a single paid run — every rung would
  otherwise silently be the full bundle. `prepare-session --probe allowlist|denylist` materialises each
  probe's inputs; dispatching them is a follow-up matrix task's first step, out of scope here.
- **Blinding is enforced by transcript detection, not by construction.** `GateBlinding`, applied only to
  a config whose `allowed` is empty, is fatal when the transcript contains an `mcp__quarry__` tool name or
  a filesystem path into `repo_root`. A bare "quarry" mention confined inside a `tool_result` payload is
  non-fatal (the target codebase mentions quarry in its own tracked files, so a bare-string gate there
  would wrongly halt the matrix); a "quarry" mention found anywhere else in the transcript is fatal. The
  retired `claude -p` port's `/tmp/quarry-bench` literal check is dropped: no such binary exists in this
  suite, so the check could never fire and would read as coverage that is not there. The session's own
  scratch directory is never treated as a leak, since it is legitimately the subagent's own working
  directory. **State this plainly: the `none` arms' blinding is weaker than a structural guarantee** — it
  is a post-hoc detector over what the agent actually said and called, not a proof that no signal could
  possibly have reached it.

## Residual leak channels — hygiene, not a structural guarantee

Three channels the harness itself cannot fully close, stated plainly rather than left implicit:

- **The `skill_listing` channel.** Every subagent transcript carries a `skill_listing` record enumerating
  the session's available skills by name and description, verbatim, with no tool call involved. Two
  mitigations are mechanical: the tracked `ladder-run` skill's own frontmatter name and description are
  constrained to contain no case-insensitive "quarry" (enforced by a unit test over the tracked file), and
  `prepare-session` scans every discoverable skill source it knows about — `~/.claude/skills/*/SKILL.md`
  and `~/.claude/plugins/cache/*/*/*/skills/*/SKILL.md` — hard-failing a `none` config's launch if any
  scanned frontmatter mentions quarry. **This scan's real limit: a session's skill listing also enumerates
  built-in and managed skills that live under neither scanned root, so a clean scan report bounds only the
  skills this harness or the operator installed under the two roots it can see, never the whole channel.**
  `GateBlinding` scans the whole transcript, `skill_listing` records included, and remains the detector
  for what the scan cannot bound.
- **The parent session transcript.** The orchestrating session writes its own transcript, which
  accumulates every `ladderbench` invocation's command line — including the results root path inside
  `repo_root`. A blinded agent that went looking for its own session transcript would find one, and
  `GateBlinding`'s repo-root check would fire fatally and invalidate the run, which is the correct outcome
  rather than a silent leak; a rung is not scanned by that gate, but a rung reading its own command lines
  learns only the results path, never a fasit.
- **The operator's own environment between the precondition check and launch.** `~/.claude/skills/`
  remains reachable by `Bash` from any session's cwd at any time after `prepare-session`'s scan ran, and
  the skill-leak scan and environment precondition are both launch-time checks, not continuous
  enforcement.

## Run environment

Every session materialised for the main matrix, the cold cell, and both permission probes shares:

- **The pinned model** — `ladder.yaml`'s `run_model`, identical across every run session (see "How to
  run" below for why it starts `null`).
- **The identical base allowlist** — `Read`, `Grep`, `Glob`, `Bash` on every run agent definition
  regardless of config, per Enforcement above.
- **`--setting-sources user,project`** on every session launch — isolates the session's own settings
  (project scope: the scratch directory's `settings.json` and agent definitions) while still loading the
  installed skill from user scope. **This flag combination ships unverified** — see "Two unverified
  implementation risks" below.
- **`QUARRY_STATE_DIR` and `QUARRY_BUILD_TAGS` both cleared, at three separate application points**, since
  no harness process wraps the dispatch and owns the spawned server's environment the way the retired
  `claude -p` subprocess used to: the generated `.mcp.json`'s server entry carries an explicit `"env"`
  block setting both to the empty string; `prepare-session` hard-fails when either is set non-empty in the
  operator's own shell; and every Go gate that resolves the state directory reconstructs the environment
  with both keys forced empty. Each variable would otherwise move the resolved state directory off the
  per-path key the cold cell's whole argument rests on: both take precedence over `workspaceKey` outright,
  and a non-empty tag set appends a `tags-<hex>` segment at every tier. A cold cell whose worktree-keyed
  state directory can be silently overridden by either variable cannot support a "this daemon is fresh
  because this path never ran before" claim.
- **`QUARRY_CONFIG` deliberately left in place** at all three points. It selects the `servers.yaml`
  overlay naming the language-server command; clearing it on a machine that needs an overlay would stop
  the server starting at all. Whether an MCP server declaration's `env` block replaces or augments the
  inherited environment is the second unverified risk below — the reason `QUARRY_CONFIG` is never touched
  is unaffected by that uncertainty either way.
- **The per-run turn ceiling, now a post-hoc gate.** The Agent Tool has no `--max-turns` equivalent, so
  nothing bounds a run mid-flight; live operator supervision is a real-time backstop, not a replacement
  for a recorded gate. `ladder.yaml`'s `max_turns` is instead evaluated at `ingest`, as a fatal gate on the
  count of assistant records in the run's copied subagent transcript. Exceeding it produces a
  `"truncated"` outcome, identical semantics to today: never retried, halts the matrix on first
  occurrence. **The committed threshold was blanked to `null`.** The old `max_turns: 60` was calibrated
  against the retired `claude -p` client's own `--max-turns` / `result.num_turns` accounting; the gate's
  basis is now "assistant records in a subagent transcript," a different count nothing establishes 60
  still means the same thing on. `require_pins` still rejects an unset value, so the operator must set it
  before the matrix starts, informed by the follow-up matrix task's first real runs — a threshold carried
  over from a different accounting basis would be either effectively dead or spuriously fatal, with no way
  to tell which without a run.
- **The per-run daemon warm-up**, for the main matrix only (`ladderbench warm`). It performs the stdio
  `initialize` + `tools/call` against a freshly spawned `quarry-mcp` using the Go MCP SDK, then asserts a
  `daemon.json` now exists at the resolved state directory. This runs once per attempt, not once per
  repetition or per worktree, since the daemon self-expires after its own idle timeout — a retried attempt
  that skipped the warm-up could dispatch against a cold daemon and silently contaminate the warm arm's
  timings. It is never called for the cold config.

## Two unverified implementation risks

No smoke launch was performed by this plan (a supervised interactive session is not something an
autonomous implementation run can carry out). Two risks are therefore recorded as unverified rather than
claimed settled, each with its documented fallback:

1. **The `--setting-sources user,project` flag combination** — isolating settings while still loading
   project-local agent definitions and user-scope skills — is unverified. **Fallback:** if project-scope
   agent discovery turns out to be suppressed by this flag combination, write the run/scorer/probe
   definitions into `~/.claude/agents/` under a `ladder-<config-id>` name instead, with `prepare-session`
   responsible for removing them again. If user-scope skill discovery is also suppressed, the operator
   invokes the protocol by reading the installed `SKILL.md` path directly — the skill must never be
   relocated into a scratch directory to work around this, since that would leak it into a blinded
   session's own working directory.
2. **Whether an MCP server declaration's `env` block replaces or augments the inherited environment** is
   unverified. If it augments, setting `QUARRY_STATE_DIR`/`QUARRY_BUILD_TAGS` to the empty string in the
   block is sufficient. If it replaces, `QUARRY_CONFIG` must be forwarded explicitly in the same block or
   the server will not start on a machine that needs a `servers.yaml` overlay. `prepare-session`'s
   environment precondition on the operator's own shell exists partly as cover for this uncertainty.

Both risks move to the follow-up matrix task, whose first real runs are what settles them empirically.

## Metrics

Every field a run's `usage.json` carries (`ExtractUsage`), partitioned against what the retired
`claude -p` port's `extract_usage.py` wrote — the subagent transcript this port reads has no terminal
`result` envelope and no `system/init` event, which forces every change below:

**Survives unchanged in definition:** the four token classes (`input_tokens`, `output_tokens`,
`cache_read_input_tokens`, `cache_creation_input_tokens`), each summed independently across every
assistant record's `message.usage` — never derived from another class, and never read off a result
envelope, because there isn't one; `tool_uses` / `tool_uses_breakdown`; `quarry_tool_uses`;
`bash_grep_count` (Bash-only, leading-command-word match, never a bare substring); `grep_tool_count`
(the dedicated `Grep` tool, kept strictly separate from `bash_grep_count`); `grep_fallback_total` (their
sum, reported alongside both, never substituted for either); `transcript` (still the path to the run's
transcript, now the copy inside the run directory); `model` (source moves from the `system/init` event to
the assistant records' `message.model`).

**Changed:** `duration_ms` becomes last-record-timestamp minus first-record-timestamp; `num_turns`
becomes the count of assistant records; `denied_tool_attempts` becomes the count of `tool_result` blocks
marked `is_error` whose text matches a permission-denial shape — **shipped provisional**, see below;
`advertised_tools` is **renamed to `granted_tools`** and sourced from the run's generated agent
definition rather than from the transcript, preserving the "extracted, never self-reported" rule: the
value still comes from a harness-generated file, just not the transcript.

**Dropped entirely, with no synthesised replacement:** `cost_usd`, `wall_clock_ms`, `result_usage`,
`result_subtype`, `result_is_error`, `session_id`. The subagent transcript has no `result` envelope and no
`system/init` event to read `total_cost_usd`, `num_turns`, `subtype`, or the advertised-tool list from. **A
cost figure synthesised from a drifting price table is worse than no number** — the field's whole point
was that it came from the client, not from a harness guess. `wall_clock_ms` would now be the same
measurement as `duration_ms` under two names, since no harness process wraps the dispatch to measure
wall-clock separately.

**Added:** `effort` (from the assistant records' top-level `effort` field, recorded but not gated —
unlike the model, it passes straight from the generated agent definition's own frontmatter with no
harness-owned mapping in between, so a mismatch would be a Claude Code bug rather than an operator or
harness error); `agent_id`; `transcript_source` (the original `~/.claude/projects/.../agent-<id>.jsonl`
path the run-directory `transcript` copy came from); `denied_tool_attempts_provisional`.

**`denied_tool_attempts` ships provisional until a real denial record is observed.** The field replaces
the retired port's structured `permission_denials` array with a pattern match over errored `tool_result`
text (`DenialShapePattern`, a single named constant). Because this task dispatches nothing that provokes
a denial — the deny-list probe's dispatch is out of scope here, and neither probe is run — no real denial
record has been observed, and the pattern is unvalidated against reality. `usage.json` therefore carries
`denied_tool_attempts_provisional: true`, and `summarize` propagates that marker onto the metric's own
stats record in `summary.json`, so no reader mistakes an unvalidated count for a measured one. **What
clears it:** the follow-up matrix task's deny-list probe is by construction the one dispatch that produces
a genuine denial record; `probe-record` captures the offending `tool_result`'s verbatim text into
`probe.json` as `denial_shape_observed`, and the follow-up task's first job is to check the pattern
against it and clear the provisional marker.

**A metric is extracted from a transcript or a harness-generated file, never from an agent's
self-report.** Nothing in `usage.json` is copied from the answer JSON block the agent itself produced.

## Scoring

`ladderbench redact` and `ladderbench record-score` dispatch a **blinded, three-input contract**: the
scoring call sees exactly the redacted answer, the task's `_meta`-stripped fasit, and the task text —
never the config id, the ladder, the allowed-tool list, the transcript, or any other run's answer. The
fasit itself is left verbatim (one fixed file, identical across all rungs of its task, so it cannot leak
which config is being graded); only its `_meta` block is stripped, since that block's `role`/`see_also`
text names quarry and `scorecard.md` by name. The single shared scoring session dispatches this prompt
once per run, always through the zero-tool scorer agent definition — see "Session model" above for why
scoring never shares a session with a run agent.

**Redaction** strips every trace of tool provenance from an answer's free-text fields before it reaches
the scorer: every `mcp__quarry__*` client-side name, every bare canonical tool name except `impact`, the
word "quarry" itself, and CLI-invocation forms (the literal `/tmp/quarry-bench` path, a `quarry <verb>`
shell form, and a `--target-dir`-style flag) — all case-insensitive, all collapsed to a single `<tool>`
token when adjacent. `impact` is deliberately excluded from the bare-name pass because it is also an
ordinary English word every Ladder-B answer's prose legitimately uses; only its `mcp__quarry__impact` and
CLI-invocation forms are redacted. The `/tmp/quarry-bench` redaction branch is kept even though the
gate-side literal check above was dropped: a gate that cannot fire reads as coverage that is not there,
but a redaction branch that never matches removes nothing and asserts nothing — a redactor's job is to
over-remove. **Acknowledged residual limit:** this is a pattern-based redaction over free text, not a
structural guarantee — a sufficiently indirect phrasing of tool provenance could in principle survive it,
the same class of limit the blinding gate above carries for the `none` arms.

**The exploration rule** (task 01) is reused unchanged from the sibling suite: recall = (the fasit's
`relevant_files`/`key_symbols` also present in the answer's) / (the fasit's total); precision = (the
answer's entries corroborated by the fasit) / (the answer's total); plus a qualitative judgment of
whether the answer's `summary` describes the same actual mechanism the fasit found, not just whether file
names overlap.

**The impact rule** (task 04), in full: recall = (the fasit's `callers_to_update` entries matched on file
AND line — a line must denote the same call site, not merely the same file — also present in the
answer's) / (the fasit's total); precision = (the answer's `callers_to_update` entries corroborated by
the fasit) / (the answer's total). `decoy_admitted` is true when the answer's `callers_to_update` contains
a call site the fasit lists under `excluded_lookalikes` — this is **reported separately, never folded
into precision** — it is a materially worse mistake than a missed real caller, since it means the agent
would have shipped a broken edit to an unrelated interface (the `burler.go:373` decoy). `lookalikes_matched`
is the count of the answer's `excluded_lookalikes` the fasit also names — **credited, never required**: an
answer naming none loses no points for it.

## Reporting discipline

- **Medians and full ranges**, never a mean alone — every cell reports `median`/`min`/`max`/`n` per
  metric (`SummariseCell`).
- **A disjoint-range bar applies to all three comparison types** (`RangesDisjoint`): a comparison is
  "separated" only when the two sides' `(min, max)` ranges do not overlap at all — ranges that merely
  touch at a single shared value are **not** disjoint.
- **The grep metrics (`bash_grep_count`, `grep_tool_count`, `grep_fallback_total`) are excluded from
  every rung-vs-control comparison** (`CompareRungToControl`, via `GrepMetrics`) — the control's preamble
  differs in steering as well as tools, so a grep-usage gap between it and a rung cannot be attributed to
  the tool exposure alone. They remain eligible for rung-vs-rung comparisons, where the preambles are
  identical except for the tool list.
- **`n = 3` stated explicitly** on every cell (`reps: 3` in `ladder.yaml`, carried into every stats
  record's `n` field) — never silently implied.
- **No significance claims.** This suite reports disjoint or overlapping ranges at `n = 3`; it does not
  run or report a statistical significance test of any kind, and no comparison output should be read as
  one.

## Hard rule on cross-suite comparison

**Correctness** (recall/precision/summary-match/decoy-admitted/lookalikes) may be compared to the sibling
`bench/loomyard-eval` suite's committed results, since both suites score against the same fasit-style
ground truth and the same rubric shape.

**Duration, tokens, cost, turns, tool counts, and grep counts may not be compared across the two suites —
not even informally.** The sibling suite dispatches its three agents as parallel sub-agent calls inside an
orchestrating session; this suite dispatches each run agent through the Agent Tool from a session running
exactly one repetition, with a different prompt shape per rung (an MCP-tool preamble here versus the
sibling's CLI-tool preamble). Any of those differences alone is enough to move every cost-shaped number by
an amount this suite cannot separate from a genuine capability effect — so cost-shaped numbers stay
suite-local.

## How to run

**Operator prerequisites — set `run_model` and `max_turns` first.** `ladder.yaml` ships with
`run_model: null` and `max_turns: null` — two operator-supplied pins now, not one. `RequirePins` refuses
to run every gate-dependent subcommand while either, or `run_effort`, `scorer.model`, or `scorer.effort`,
is unset. Edit `ladder.yaml` and set both before dispatching a single real run. This is deliberate: the
model id is not fixed in the design, and the turn ceiling's accounting basis changed under this port (see
"Run environment" above), so silently carrying either forward would make the matrix unreproducible or the
ceiling meaningless.

Then, from the repo root, launch a session per the protocol in `.claude/skills/ladder-run/SKILL.md`:

```
ladderbench prepare-session --config-id <config-id> --rep <n> --results-root bench/loomyard-eval/ladder/results/<YYYY-MM-DD>
```

prints the launch command for one run session; run `/ladder-run` inside the launched session to drive its
attempt loop. Once every run session for the matrix has ingested, prepare and launch the single scoring
session:

```
ladderbench prepare-session --scoring --results-root bench/loomyard-eval/ladder/results/<YYYY-MM-DD>
```

and run `/ladder-run` inside it to drive the scoring loop. Finally, from the repo root:

```
ladderbench summarize --results-root bench/loomyard-eval/ladder/results/<YYYY-MM-DD>
```

writes `summary.json` into the same results directory and exits non-zero (after naming the incomplete
cells on stderr) if the matrix is not yet fully complete.

**Re-invoking the harness resumes by skipping completed work.** A run session's own resume point is
`ingest.json` (`ladderbench next-run`); the scoring session's is "`ingest.json` present, `run.json`
absent"; `summarize` still reads only `run.json`'s completeness (`IsComplete`). An interrupted matrix is
always safe to resume into the same results directory — see `.claude/skills/ladder-run/SKILL.md` for the
exact resume-aware loop each session shape runs.

**Cross-session serialisation is an operator obligation.** `ladderbench ingest`'s single-flight check
cannot see another session's state beyond what is on disk, and concurrent sessions contend for CPU, the
shared daemon, and model-side rate limits — any of which would make wall-clock incomparable across
configs. `prepare-session` writes `<results-root>/.session-active` naming the session it prepared and
refuses to prepare a second session while one exists; `prepare-session --release` clears it, and the
tracked skill's final step in every session calls it. **This is a guard, not a proof** — an operator who
deletes the lockfile or launches from a second checkout defeats it — so treat it as catching the ordinary
mistake (forgetting a session is still open), not as a substitute for actually running one session at a
time.

**The matrix targets a pinned Linux checkout and is not expected to run on Windows.** The Go code uses
`path/filepath` and `os.UserCacheDir` throughout so nothing is structurally Linux-only, but no attempt is
made to verify Windows execution, and the pinned worktrees this task's `ladder.yaml` declares are Linux
paths.

## How to test

From the repo root:

```
go test ./bench/loomyard-eval/ladder/...
```

`go vet ./...` is the module-wide check run at every development boundary; the repository's own full test
run is a separate, repo-wide gate this suite's own tests do not invoke.

## Layout

```
bench/loomyard-eval/ladder/
├── README.md                    (this file)
├── ladder.yaml                  the single declarative source of truth
├── cmd/ladderbench/              the eleven-subcommand session-boundary CLI (cobra)
│   ├── root.go                  command tree; the package doc lists all eleven subcommands in order
│   ├── preparesession.go        prepare-session
│   ├── nextrun.go                next-run
│   ├── warm.go                   warm
│   ├── restoreworktree.go        restore-worktree
│   ├── ingest.go                 ingest
│   ├── invalidate.go             invalidate
│   ├── redact.go                 redact
│   ├── recordscore.go            record-score
│   ├── proberecord.go            probe-record
│   ├── coldcell.go               cold-cell
│   └── summarize.go              summarize
└── internal/ladder/              the ported logic package: ladder-config loading and validation,
                                  agent-definition and session-input generation, transcript parsing and
                                  usage extraction, every validation gate, redaction and scorer-prompt
                                  assembly, run-state and resume bookkeeping, and summary/median/
                                  disjoint-range building, plus *_test.go beside every module and
                                  testdata/ fixtures
```

There is no `scripts/` or `tests/` tree any more — the Go binary and its own `internal/ladder/*_test.go`
suite are the sole implementation and test surface. A results tree still looks like:

```
results/<YYYY-MM-DD>/
├── probe.json                   tracked
├── cold_cell.json               tracked
├── summary.json                 tracked
├── conclusion.md                tracked (written by the follow-up matrix task)
└── raw/                         gitignored (results/**/raw/) — per-run transcript.jsonl,
                                  transcript.meta.json, answer.json, answer.redacted.json, usage.json,
                                  ingest.json, score.json, run.json, settings.json, the run agent
                                  definition, and mcp.json when the config declared one
```

`.session-active`, the cross-session lock, lives at `<results-root>/.session-active` and is separately
gitignored (`results/**/.session-active`) — it is live session state, never part of a committed result.

## Design rationale — do not "simplify" these away

- **Sequential dispatch, never concurrent, mechanically enforced within a session.** `ladderbench ingest`
  fails loud when asked to ingest repetition `n` while repetition `n-1` of the same config has neither an
  `ingest.json`, a `run.json`, nor an exhausted attempt record — a predicate the per-repetition session
  loop in `.claude/skills/ladder-run/SKILL.md` satisfies by construction. Concurrent runs would contend
  for CPU, the shared daemon, and model-side rate limits, any of which would make wall-clock incomparable
  across configs.
- **The cold cell's per-run distinct worktrees and its supervised-connection assertion.** Each cold
  repetition's session builds a fresh worktree at `cold_worktree_template` with `{n}` substituted, clears
  the resolved state directory, and asserts the cold-before gate before every attempt (including retries)
  so a failed attempt's leftover `daemon.json` cannot fail the precondition deterministically. The
  cold-after gate then requires, when the transcript used at least one daemon-backed tool, that a
  `daemon.json` actually exists afterward — a daemon-backed call with no resulting state file means the
  native fallback was taken, on which path the shared daemon address is not a function of the state
  directory at all, and the run is invalidated rather than reported as cold. `toc_dir` and `toc_file` are
  excluded from the daemon-backed set entirely: their handlers reach the tree-sitter path directly and
  never touch the server, so a toc-only run carries no warmth signal and is reported as such
  (`cold_no_daemon_backed_call`), never silently read as "native fallback taken." **A failed cold attempt
  relaunches the whole session; there is no in-session cold retry** — see "The cold config" in the tracked
  skill for why.
- **The cold cell runs last**, after the entire main matrix, and its session-preparation step drains
  every daemon the main matrix left resident on both task worktrees before its first repetition — neither
  task daemon may compete with a cold run for CPU or, worse, be mistaken for the cold run's own daemon.
- **The hard worktree restore after every attempt, with dirtying recorded rather than gated.**
  `restore-worktree` (`git reset --hard` + `git clean -fdx` + re-neutralisation) runs unconditionally
  after every main-matrix attempt, successful or not, and is never used by the cold config (whose
  worktree is disposable and torn down outright, never restored in place). The worktree-dirtied
  observation is taken by `ingest`, before `restore-worktree` ever runs, specifically because an
  observation taken after the restore would read "false" for every run — the restore is precisely what
  erases the evidence. Dirtying is recorded as a non-fatal observation, never gated: an agent legitimately
  editing files while exploring is not itself invalid.
- **The three-attempt cap halts the whole matrix, not just the offending config.** `MaxAttempts = 3`: on a
  "failed" outcome the run directory is invalidated (`ladderbench invalidate`) and retried up to three
  attempts; the third failure's own invalidation errors instead of returning a next attempt index, which
  is the matrix halt, leaving every already-completed run intact so a resumed session picks up past them.
  A "truncated" outcome (the post-hoc `max_turns` gate) is never retried and halts on its first
  occurrence, since the ceiling is a matrix-wide constant a second attempt would hit identically.
- **The preamble steering confound, and what it forbids the conclusion from claiming.** A `none` control's
  agent definition and a quarry rung's agent definition differ in more than which tools their `tools:`
  frontmatter lists — the run-agent prompt body is identical text for every config, but a rung's own tool
  descriptions (surfaced by the tools it is granted) still amount to steering a control's identical prompt
  never receives. That is why grep-usage metrics are excluded from every rung-vs-control comparison (see
  Reporting discipline) and why any control-vs-rung reading of this suite's results must be understood as
  "tool exposure plus steering," never "tool exposure alone." The conclusion this suite's results feed must
  not claim to have isolated tool exposure from steering in a rung-vs-control comparison — only
  rung-vs-rung comparisons, where every definition is identical except for the tool list, can support that
  stronger claim.
