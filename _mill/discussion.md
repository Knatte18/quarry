# Discussion: Per-capability quarry-mcp benchmark suite

```yaml
task: Per-capability quarry-mcp benchmark suite
slug: mcp-capability-bench
status: discussing
parent: main
```

## Problem

`quarry-mcp-vs-cli-bench` (#006, done) benchmarked quarry's MCP exposure as a single
bundle: every "A" arm had all seven tools available at once, and the only enforcement
was prompt wording. Task 01 was nominally "the toc task", but the transcript shows the
agent also reached for `workspace_symbol` and `textDocument_references`. The result —
"the bundle helps, and once warm it helps a lot" — therefore cannot be attributed to any
individual tool. quarry-mcp deliberately exposes two different verb shapes (LSP-mirrored
for `textDocument_definition` / `textDocument_references` / `workspace_symbol`,
quarry-native for `toc_dir` / `toc_file` / `impact` / `assert_no_callers`), and if some
of those verbs carry the whole measured benefit while others are dead weight, that is
worth knowing before investing further in this exposure layer.

Why now: #006 just landed and its own scorecards name this gap explicitly. Tasks 03 and
04's scorecards each end with a correction retracting their "warm MCP decisively wins"
claim — the fast reruns were executed by the orchestrating session *after* it had already
read the task's ground truth, so they measured zero-orientation-cost, not tool value. Both
corrections point at this task by slug as the thing that will settle the question with
repeated, uncontaminated runs. #006's own design note ("one run per arm, first pass...
this is a temperature check, not a publishable study") flags the same variance gap. Every
one of those defects is a methodology defect, so the fix is a methodology redesign, not
another pass of the existing harness.

## Scope

**In:**

- A new benchmark suite under `bench/loomyard-eval/ladder/` — a self-contained protocol
  README, a run harness, a metrics extractor, and a scoring step.
- An incremental capability ladder: two task-shaped ladders totalling 14 configurations,
  each isolating one quarry-mcp tool or a named small combination, plus a zero-quarry
  control and a full-bundle rung per ladder.
- Hard enforcement of each rung's capability set via a generated permission deny-list, so
  a rung's restriction is structural rather than instructed.
- Blind execution: every run is a fresh headless agent process with no memory of this
  conversation or of any other run.
- Repetition: N = 3 runs per configuration (42 runs), plus a 3-run cold-daemon comparison
  cell against the pre-warmed task-01 full-bundle cell (45 runs total). All 45 dispatch
  sequentially.
- Freshly written MCP-shaped agent preambles per rung, replacing #006's CLI-shaped
  "Agent A preamble" template.
- Full benchmark accounting per run: wall-clock duration, every token class, total tool
  calls, per-tool call breakdown, and non-quarry fallback counts — all extracted from the
  run's own transcript JSONL, never from an agent's self-report.
- Correctness scoring of every run against the existing tracked `c.json` fasit for its
  task — #006's exploration rule for Ladder A, and a new impact-analysis rule, written in
  this discussion, for Ladder B.
- A git-tracked written conclusion synthesising the whole matrix.
- A `.gitignore` entry making raw per-run artifacts untracked.

**Out:**

- Any change to quarry itself — the CLI, `internal/mcpserver`, or the tool schemas. This
  task measures the existing exposure layer; it does not modify it. If the suite surfaces
  a quarry defect, it is written up in the conclusion and filed, not fixed here.
- Re-running or amending #006's existing results under `bench/loomyard-eval/results/`.
  Those stay exactly as they are, including their corrections; this suite writes to its
  own tree.
- Benchmarking the quarry **CLI** exposure. #006 already covered CLI-vs-MCP at bundle
  level; this task is per-capability within the MCP surface only, and no
  `/tmp/quarry-bench` binary is built for any run.
- Loomyard task 02 (never run, no fasit exists to score against) and task 03 (its own
  postmortem established that a plain undefined-symbol compile break is caught by
  `go build` regardless of tooling, so it cannot discriminate between rungs).
- New benchmark tasks written from scratch. See the "Reuse existing tasks" decision.
- Statistical significance claims. See the "Reporting discipline" decision.
- Changing the pinned Loomyard SHA or the task texts.

## Decisions

### Ladder shape — two task-shaped ladders, not one flat list

- Decision: run two separate ladders, each against the one existing task whose shape the
  capability under test is plausibly relevant to, rather than running every rung against
  every task.

  **Ladder A — exploration, against task 01 (`01-reed-geometry-exploration`), 6 configs:**

  | config id | quarry tools allowed |
  |---|---|
  | `a0-none` | (none) |
  | `a1-toc-file` | `toc_file` |
  | `a2-toc-dir` | `toc_dir` |
  | `a3-toc-pair` | `toc_dir`, `toc_file` |
  | `a4-toc-pair-symbol` | `toc_dir`, `toc_file`, `workspace_symbol` |
  | `a5-bundle` | all seven |

  **Ladder B — navigation / impact analysis, against task 04
  (`04-shedadapters-shuttle-impact`), 8 configs:**

  | config id | quarry tools allowed |
  |---|---|
  | `b0-none` | (none) |
  | `b1-symbol` | `workspace_symbol` |
  | `b2-definition` | `textDocument_definition` |
  | `b3-references` | `textDocument_references` |
  | `b4-lsp-trio` | `workspace_symbol`, `textDocument_definition`, `textDocument_references` |
  | `b5-impact` | `impact` |
  | `b6-assert-no-callers` | `assert_no_callers` |
  | `b7-bundle` | all seven |

  **Cold-daemon cell (see "Daemon warmth"), 1 config:** `a5-bundle-cold` — identical tool
  set and task to `a5-bundle`, differing only in daemon state.

  14 main configs × N = 3 → 42 runs, plus the cold cell's 3 → 45.

  **Config ids are ladder-qualified and globally unique**, and `ladder.yaml`'s `id` field is
  that unique key. Everything downstream is keyed on it: the `raw/<config-id>/<n>/`
  directory, the `run.json` resume lookup, `summarize.py`'s per-config grouping, and the
  "exactly 3 complete runs per config" gate. A bare `none` or `bundle` would collide two
  ladders' runs into one directory and make "its own ladder's `none` control" unresolvable
  from the id, so the ladder letter is part of the id rather than a separate column. Each
  config also carries an explicit `ladder: a|b` field, so the disjoint-range rule resolves
  its control by lookup rather than by parsing the id string.

- Rationale: a full cross product (11+ rungs × 2 tasks × 3 reps) is ~70 runs for
  information the pairing already yields — `assert_no_callers` on an open-ended
  exploration task, or `toc_dir` on a "find every real caller" task, would predictably
  measure nothing. Pairing each rung with the task shape it targets keeps the matrix at a
  size that can be executed and analysed while still covering all seven tools
  individually. Both ladders carry their own `none` control and their own `bundle` rung,
  so every rung is comparable within its ladder against both the floor and the ceiling,
  and the two ladders are never compared to each other directly.

  Each ladder's rungs are genuinely runnable in isolation: `lspEntry` and `nativeEntry`
  (`internal/mcpserver/lspentry.go:34`, `internal/mcpserver/nativeentry.go:20`) each accept
  a bare `symbol` form, so `textDocument_references`, `impact`, and `assert_no_callers`
  can all resolve a symbol project-wide with no `workspace_symbol` call to seed them. No
  rung is structurally crippled by its own isolation.

  The task body flags that task 01 may not be cleanly answerable by toc alone. That is
  accepted and is in fact the measurement: a rung that cannot reach the fasit answer scores
  low on recall, and *that is the finding*. The tasks are held fixed across rungs precisely
  so recall differences are attributable to the capability set and nothing else.
- Rejected: full cross product (too many runs for the information gained). One combined
  ladder run against both tasks (same problem, and it dilutes each rung with a task it
  cannot engage with). Dropping `textDocument_definition` and `assert_no_callers` as
  obviously weak (that judgement is exactly what the benchmark exists to make on evidence;
  a rung showing no benefit is a result, not a wasted run).

### Reuse existing tasks 01 and 04

- Decision: reuse `bench/loomyard-eval/tasks/01-reed-geometry-exploration.md` and
  `bench/loomyard-eval/tasks/04-shedadapters-shuttle-impact.md` unchanged, at their
  existing pins, with their existing `<TASK TEXT>` blocks and output schemas. Exclude
  tasks 02 and 03.
- Rationale: both reused tasks already have a tracked, human-reviewed fasit
  (`bench/loomyard-eval/results/2026-08-28/*/c.json`) produced by a high-effort blind
  reference agent. Reusing them means 45 runs need zero new fasit work and are directly
  comparable to #006's own numbers for the same tasks. They also cover the two distinct
  shapes the ladders need: open-ended subsystem survey (01) and precise caller
  enumeration with a deliberate interface-conflation decoy (04). Task 02 has no fasit and
  was never executed; task 03's postmortem in its own scorecard established it cannot
  discriminate.

  **Ladder B is expected to discriminate on efficiency, not correctness, and that is not
  the same defect that excluded task 03.** Task 04's committed scorecard records both #006
  arms at 100% recall/precision with the decoy correctly excluded by both, and calls their
  cost "statistically indistinguishable" — so Ladder B's correctness axis is likely
  saturated across all 8 rungs, and its 24 runs will mostly be read as an efficiency and
  process comparison (tool calls, turns, tokens, duration, how far each rung has to route
  around its own restriction via grep). The asymmetry with task 03 is deliberate: task 03
  was excluded because `go build` answers its question outright, so *no* arm's process
  differs meaningfully — there is nothing left to measure. Task 04's arms reach the same
  answer by visibly different routes, which is exactly what the per-capability question is
  about. This is also stated so the conclusion cannot later present a saturated correctness
  axis as evidence that every Ladder B rung "works": if all 8 rungs including `none` score
  100%, the honest reading is that this task cannot separate them on correctness, and any
  claim about a rung must then rest on its efficiency ranges alone.
- Rejected: authoring new, narrower tasks per rung (each would need its own fasit run,
  and a task shaped to make one capability shine is a rigged measurement — the honest
  question is how much of a *realistic* task each capability can carry alone). Including
  task 02 (would require producing a fasit first, expanding scope). Including task 03
  (known non-discriminative).

### Hard enforcement via generated per-run settings deny-list

- Decision: each run executes as its own headless `claude -p` subprocess, launched with an
  explicit `--mcp-config` pointing at the repo's quarry server and an explicit
  `--settings <generated>` file whose `permissions.deny` array names every
  `mcp__quarry__<tool>` **not** in that config's allowed set. The deny file is generated per
  config by the harness from a single declarative ladder definition, so the deny-list can
  never drift out of sync with the config table.

  **The `none` configs get no `--mcp-config` at all.** They are launched without the quarry
  server declared, rather than with it declared and all seven tools denied. A declared
  server named `quarry` exposing a `mcp__quarry__*` namespace is exactly the structural
  leak the blinding constraint forbids — a `none` agent must not be able to observe that
  quarry exists, and denying the tools still advertises the name. Each run's process working
  directory is its own task worktree under `/tmp`, never this quarry repository, so nothing
  in a run's ambient context points at quarry's source, docs, or `.mcp.json`.

  That cwd choice removes the *invitation*, not the *capability*: Bash is in the allow-set
  and cwd does not bound what an absolute path can read, so a `none` agent that went looking
  could still reach this repo. The blinding is therefore enforced by detection, not by
  construction — the per-run gate rejecting any `none` run whose transcript contains a
  `mcp__quarry__*` tool or the string "quarry" is what actually holds the constraint, and a
  run that trips it is invalidated. This is a weaker guarantee than #006's structural
  blinding claim and is stated as such rather than overclaimed.

  **Run environment, pinned identically across all 45 runs:**
  - `--model` is pinned to one explicit model id for the entire matrix and recorded in
    `ladder.yaml` and in every `usage.json`. **The operator sets that id in `ladder.yaml`
    once, before the matrix starts; it is deliberately not fixed in this discussion.** The
    right choice depends on 45-run cost and on what the operator wants comparability
    against, and hard-coding a model id into a design document would rot the moment the
    available models change. The harness refuses to start if the field is unset, and a
    validation gate rejects any run whose transcript reports a different model than the
    pinned one. Every headline metric this suite reports —
    `duration_ms`, all four token classes, `cost_usd`, `num_turns` — is model-dependent, so
    an unpinned or drifting model would make cells incomparable and the matrix
    unreproducible. If the matrix ever has to be re-run on a different model, that is a new
    dated results directory, not a mixed one.
  - The generated settings file also carries the `permissions.allow` entries the non-quarry
    toolset needs (Read, Grep, Glob, and Bash), and the harness runs each subprocess in a
    non-interactive permission mode, so a headless run can use its standard tools without
    blocking on a prompt. The allow-set is identical across all 45 runs; only the quarry
    deny-list varies.
  - `QUARRY_STATE_DIR` is cleared and `--state-dir` is never passed, for every run. See the
    warmth decision for why this matters to the daemon key.

  **`denied_tool_attempts` rests on an unverified premise.** Whether `permissions.deny`
  hides a denied tool from the advertised tool list or advertises it and rejects the call
  decides whether that metric can ever be non-zero: if denied tools are filtered out of what
  the model sees, the count is identically zero in every restricted rung and measures
  nothing. The harness must establish which semantics apply with a single throwaway probe
  run before the matrix starts, and record the answer in the results directory. If denied
  tools are hidden, `denied_tool_attempts` is dropped from the reported metrics rather than
  reported as a meaningless column of zeros — and the enforcement is if anything stronger,
  since a hidden tool cannot be attempted at all. Either outcome is fine; silently reporting
  zeros as if they were evidence is not.

  **The same probe must first establish that denial blocks at all.** Before the
  hidden-vs-advertised question is even meaningful, the probe run is given a deny-list and
  instructed to call a denied `mcp__quarry__*` tool; the probe passes only if that call does
  not succeed. This is the load-bearing premise of the entire suite — if `permissions.deny`
  turned out not to block MCP tools, every rung would silently be the full bundle and all 45
  runs would be worthless. It is checked once, up front, for the cost of one run, rather
  than assumed across the matrix and discovered afterwards. A probe failure halts before any
  matrix run starts.
- Rationale: the task body is explicit that instruction alone is not reliable — this
  session's predecessor observed tasks nominally scoped to one verb where the agent used
  three. A deny-list makes the restriction structural: a denied tool call fails rather
  than silently succeeding, and the failure is visible in the transcript. Deriving the
  deny-list from the allow-set (rather than hand-writing 14 deny files) means adding a
  tool to quarry later cannot silently leak it into every restricted rung.

  Running each arm as a separate headless process is the same decision viewed from the
  other side: it is also what makes each run **blind**. #006's retracted warm-run results
  were contaminated precisely because the orchestrating session executed the runs itself
  after having read the ground truth. A fresh process cannot be contaminated by the
  orchestrator's context, and it yields a machine-readable result envelope and a
  per-session transcript file for verification as a side effect.
- Rejected: prompt instruction alone (the defect being fixed). Per-rung subagent
  definitions under `.claude/agents/` with restrictive `tools:` frontmatter (works for
  restriction, but the agent still runs inside the orchestrator's session, so it does not
  solve contamination or give a clean per-run transcript). Adding a `--tools` allow-list
  flag to `cmd/quarry-mcp` (changes quarry, which is out of scope, and would test a
  different server configuration than the one shipped).

### Non-quarry tools stay available in every arm

- Decision: every run keeps the standard toolset (Read, Grep, Glob, Bash) available,
  including the `none` control. Only quarry MCP tools are ever denied. No
  `/tmp/quarry-bench` CLI binary is built for any run, so the CLI cannot become a backdoor
  around a denied MCP tool.
- Rationale: this matches #006's arms and keeps the comparison honest — the real question
  is not "can an agent work with only `toc_file`" but "does having `toc_file` improve an
  agent that also has grep". It also preserves the grep-fallback counts as a live signal:
  when a rung's agent routes around its missing capability with grep, that shows up as a
  measurable number, which is itself evidence about that capability's value. Denying grep
  would produce artificially large deltas that say nothing about real use.
- Rejected: denying Grep/Bash in quarry-enabled arms (would manufacture the result).
  Building the CLI binary for these runs (a denied MCP tool would remain reachable via
  Bash, defeating the enforcement decision entirely).

### Repetition count N = 3

- Decision: N = 3 runs per configuration, run independently, each with its own fresh
  process and its own transcript.
- Rationale: the task body specifies N ≥ 2–3. Three is the smallest N that yields a median
  rather than a mean of two, which matters because agent runs are heavy-tailed — one run
  that wanders costs 3× the others and would drag a two-run mean badly. Three also keeps
  the matrix at 45 runs, which is executable. Going to N = 5 (75 runs) buys tighter
  intervals but the reporting discipline below already refuses to make claims those
  intervals would support.
- Rejected: N = 2 (no median, one outlier dominates). N = 5+ (cost out of proportion to
  the strength of claim this suite is allowed to make). Adaptive N — more reps only for
  ambiguous cells (defensible, but it makes the matrix non-uniform and invites
  stopping-rule bias; a uniform N is easier to defend and to re-run).

### Daemon warmth held constant, with one explicit comparison cell

- Decision: every run in the 42-run main matrix executes against a **pre-warmed** quarry
  daemon and a pre-built worktree, warmed by the harness before the run's process starts.
  Warmth is a controlled constant, not a ladder dimension. Separately, one dedicated cell
  runs config `a5-bundle-cold` — the same tool set and task as `a5-bundle` — N = 3 times
  against a **cold** daemon, giving a
  clean warm-vs-cold contrast at n = 3 per side with blind agents on both sides.

  **How the cold cell is actually made cold.** quarry keys its daemon per absolute
  target-dir path (`workspaceKey`, `internal/cli/paths.go:76`) and the daemon only
  self-expires after `daemonIdleTimeout = 10 * time.Minute`
  (`internal/quarryengine/daemon/ensureserver.go:143`). Task 01's shared read-only worktree
  therefore keeps one warm daemon alive across every task-01 run, so "run the cold cell
  later" cannot produce a cold daemon. Instead each of the 3 cold runs gets **its own
  freshly-built disposable worktree at a distinct path**
  (`/tmp/loomyard-eval-01-cold-<n>`), which yields a distinct `workspaceKey` and therefore
  a genuinely unstarted daemon, and that worktree is removed immediately after its run so
  the path is never reused. No pre-warm step is performed for these three runs.

  **Per-path keying holds only on the supervised strategy, so the cold cell must assert
  it.** `workspaceKey` derives the state directory, and the state directory derives the
  socket, only for `ensureSupervised`. `EnsureServer` falls back to `ensureNative`, whose
  own `nativeArgv` doc comment states plainly that gopls — not quarry — picks the shared
  `-remote=auto` daemon address and that "that address is not a function of the state
  directory at all" (`internal/quarryengine/daemon/ensureserver.go`, `nativeArgv`'s comment).
  On that path a distinct worktree path buys nothing: three "cold" runs could all join one
  already-warm shared gopls daemon, and because the native path writes no `daemon.json`, a
  naive "no state exists for this key" check would pass vacuously and report contamination
  as coldness.

  **Supervised-vs-native is detected from the state directory, because nothing else is
  observable.** `cmd/quarry-mcp/main.go` writes exactly one stderr line (the resolved
  target directory) and nothing under `internal/` writes to stderr at all; `ConnKind`
  (`internal/quarryengine/daemon/ensureserver.go`) is never surfaced to a client. Since
  quarry may not be modified, the only source-grounded signal available is the state
  artifact: the supervised strategy writes `<stateDir>/<lang>/daemon.json`
  (`internal/quarryengine/daemon/daemonstate.go`), the native strategy writes nothing there.
  So per cold run the harness: (1) resolves the run's expected state directory the same way
  `ResolveStateDir` does, and asserts it holds no `daemon.json` **before** the run —
  otherwise the daemon is already warm and the run is invalidated; (2) asserts a
  `daemon.json` **appeared** there after the run, which is the positive confirmation the
  connection was supervised — its absence means the native fallback was taken and the run
  is invalidated rather than reported as cold; and (3) runs with `QUARRY_STATE_DIR` cleared
  and no `--state-dir`, since
  `ResolveStateDir` (`internal/cli/paths.go`) gives both of those precedence over
  `workspaceKey` and either would silently collapse all three cold runs onto one shared key.
  If the native fallback turns out to be unavoidable on the operator's machine, the cold
  cell is reported as not-run rather than reported with numbers that cannot be trusted.

  **The cold cell runs last, and its daemons are drained between runs.** A supervised cold
  run leaves a gopls daemon resident for `daemonIdleTimeout` (10 minutes,
  `internal/quarryengine/daemon/ensureserver.go`) against a worktree the harness then
  deletes. Placing the cold cell after the entire main matrix means no leaked daemon can
  ever be resident during a timed main-matrix run; and between the three cold runs the
  harness waits for the previous run's daemon to exit before starting the next, so no cold
  run is timed alongside a resident daemon from its predecessor either. Ordering it last
  rather than first also avoids the alternative failure: run first, and its three daemons
  would be resident through the opening minutes of the main matrix, loading exactly the
  `duration_ms` figures the separation rule reads.
- Rationale: #006's scorecards leave warm-vs-cold genuinely unresolved (n = 1 blind, and
  the apparently decisive warm numbers were retracted as contaminated), and both task 03
  and task 04 scorecards name this task as the place to settle it. Making warmth a second
  ladder dimension would double the matrix to 90 runs for a question orthogonal to
  per-capability attribution. One dedicated 3-run cell answers the warmth question at the
  same n as everything else while leaving the main matrix's warmth confound eliminated by
  construction. Per-run distinct worktree paths are what make the cold side real: a
  time-ordering rule would silently degrade into "warm vs warm" the moment the schedule
  slipped, reproducing exactly the contamination the two retracted scorecards asked this
  task to resolve — and a contaminated cold cell is worse than none, because it looks like
  data.
- Rejected: warmth as a full second dimension (90 runs, orthogonal question). Ignoring
  warmth entirely (leaves it as an uncontrolled confound, and silently declines a question
  two committed scorecards explicitly delegate here). Cold everywhere (would make every
  cell's numbers dominated by first-call daemon startup, compressing the differences the
  ladder is trying to resolve). Ordering the cold cell first against the shared task-01
  worktree (only the first of its three runs would be cold; the other two would inherit
  its still-live daemon, and any later re-run of that cell would be fully warm). Killing
  the daemon process between runs (relies on internal process/state layout this task is not
  allowed to modify and would silently stop working if quarry's daemon lifecycle changed;
  a distinct key needs no such coupling).

### Runs dispatch sequentially

- Decision: all 45 runs execute one at a time, never concurrently, in this order: the whole
  42-run main matrix first, then the 3 cold-cell runs last. Each
  run owns a directory under `results/<date>/raw/<config-id>/<n>/`, and its terminal state
  is recorded in a `run.json` written there **only after** the run's answer parsed, its
  `usage.json` extracted, and its validation gates passed. Re-invocation skips a run iff
  that `run.json` exists and records `state: "complete"`. Invalidating a cell (a failed
  gate, a native-fallback cold run, a missing answer block) deletes that run's `run.json`
  and moves its directory aside to `<n>.invalid-<k>/`, so the next invocation re-runs it and
  the discarded attempt is still inspectable.

  **Re-runs are capped at 3 attempts per run.** Several gates can fail deterministically —
  a `permissions.deny` that does not actually block, a model mismatch, a cold run that keeps
  taking the native fallback — and an uncapped "invalidate and re-run" would burn paid runs
  forever on a cell that can never pass. On the third failure the harness stops the whole
  matrix (not just that cell), reports which gate failed and why, and leaves every completed
  run intact so the matrix resumes once the cause is fixed. Halting the matrix rather than
  marking one cell not-run is deliberate: a deterministic gate failure is almost always a
  harness or environment fault that invalidates the other cells' premises too, and
  discovering that after 40 more paid runs is worse than discovering it immediately. The
  cold cell keeps its own separate disposition (reported as not-run when the supervised
  strategy is unavailable on the operator's machine), because that is an environment
  limitation rather than a fault.
- Rationale: `duration_ms` is the metric the disjoint-range separation rule leans on
  hardest, and concurrent runs contend for CPU, for the gopls daemon backing a shared
  worktree, and for model-side rate limits — any of which would make wall-clock
  incomparable across configs and could manufacture or erase a separation. Sequential
  dispatch is also what makes the cold cell's per-run fresh worktrees meaningful (a
  concurrent run against a sibling path could otherwise be racing daemon startup) and what
  makes the resumability constraint straightforward: a completed run is a finished record
  on disk, not a partially-drained parallel batch. The cost is wall-clock for the operator,
  not accuracy, and that is the right trade here.
- Rejected: concurrent dispatch (faster, but corrupts the primary timing metric and the
  cold cell's premise). Concurrency only across different tasks (still contends for CPU and
  rate limits, and buys little since the matrix is two tasks). Dropping `duration_ms` in
  favour of tokens so concurrency becomes safe (throws away a headline benchmark metric the
  operator explicitly asked to keep).

### Metrics — full benchmark accounting, extracted from transcripts

- Decision: every run records, into a per-run `usage.json`:
  - `duration_ms` — wall-clock for the run.
  - `tokens` — a complete breakdown, each field reported separately and never summed into
    a single opaque number: `input_tokens`, `output_tokens`, `cache_read_input_tokens`,
    `cache_creation_input_tokens`.
  - `cost_usd` — when the result envelope reports it.
  - `num_turns` — assistant turns, which is what wall-clock latency actually tracks.
  - `tool_uses` — total tool-call count.
  - `tool_uses_breakdown` — a per-tool-name count covering **every** tool the run invoked,
    quarry and non-quarry alike (`toc_file`, `workspace_symbol`, `Read`, `Grep`, `Bash`,
    …), so "which tools were actually used" is answerable directly rather than inferred.
  - `quarry_tool_uses` — the subtotal across `mcp__quarry__*` tools.
  - `bash_grep_count` — Bash invocations whose command matches `grep`/`rg`. This is
    exactly #006's `grep_fallback_count` definition (README "Dispatch protocol" step 4
    greps the transcript's Bash `"command"` fields only), and is the field used for any
    comparison against #006's numbers.
  - `grep_tool_count` — calls to the native Grep tool, which #006's definition does not
    cover. Reported as its own field and never silently folded into the one above; a
    combined `grep_fallback_total` is derived from the two for within-suite comparisons
    only, where it is the more honest signal of routing-around behaviour.
  - `denied_tool_attempts` — attempts to call a tool the rung's deny-list blocked. A
    non-zero value is itself a finding (the agent wanted that capability). Reported only if
    the probe described in the enforcement decision shows denied tools are advertised
    rather than hidden; otherwise dropped.
  - `session_id` and the transcript path, so any number can be re-derived.

  - `worktree_dirtied` — whether the run left its shared worktree modified. Not a failure;
    see the worktree decision in Technical context.

  **How the transcript is obtained.** Each run is launched with
  `--output-format stream-json`, and the harness captures that stream directly to
  `raw/<config-id>/<n>/transcript.jsonl` as the process runs. It does not locate a session
  file after the fact by `session_id` — that would depend on the client's on-disk session
  layout, which this suite does not control and which could change underneath it. Capturing
  the stream also means the transcript exists even for a run that dies mid-way, which is what
  makes a failed run diagnosable rather than just absent. `extract_usage.py`'s fixtures are
  therefore excerpts of this stream format, and that is the only format it needs to parse.

  Every one of these is extracted programmatically from that transcript by a script under
  `bench/loomyard-eval/ladder/scripts/`. No number in this suite is ever taken from an
  agent's self-report.
- Rationale: #006 caught real self-report discrepancies twice — undercounted `toc_file`
  and `Read` calls in the cold run, and an undisclosed grep call in the warm run. Both
  were only caught by transcript verification, which is why the task body makes transcript
  verification a hard requirement. Automating the extraction closes the remaining hole:
  45 runs is too many to verify by hand, and hand-verification is exactly where the
  undercounts crept in. Reporting every token class separately rather than one total also
  fixes a stated defect in #006's own `_note` — its 2026-08-28 and 2026-08-29 token figures
  are not comparable because the classes were not broken out consistently.
- Rejected: self-reported usage (the defect). A single `tokens` scalar (not comparable
  across runs, per #006's own note). Hand-verification per run (does not scale to 45 and is
  the known failure mode).

### Correctness scoring against the existing fasit

- Decision: each run's answer JSON is scored against its task's existing tracked
  `c.json` by a dedicated scoring agent per run. Its input is exactly three things: the
  run's `answer.json`, the task's fasit `c.json`, and the task's `<TASK TEXT>`. It never
  sees the config id, the rung's tool set, the run's transcript, or any other run's answer.
  The task text is included because Ladder A's rule requires judging whether `summary`
  describes the same mechanism C found, which cannot be assessed without knowing what was
  asked — and the task text is identical across every rung of a ladder, so it reveals
  nothing about the config.

  **The scorer is pinned exactly as the runs are.** Its model, effort, and prompt template
  are fixed in `ladder.yaml` alongside the run model, and the resolved values are recorded
  in every `score.json`. A scorer that drifted across the matrix would shift scores between
  rungs graded at different times — the same defect as an unpinned run model, one axis over,
  and harder to spot because it moves the correctness numbers rather than the cost ones.

  **Ladder A (task 01, exploration schema)** uses the exploration rule already written in
  `bench/loomyard-eval/README.md`'s "Scoring" section unchanged: recall = C's
  `relevant_files`/`key_symbols` also present in the run's, over C's total; precision = the
  run's entries corroborated by C, over the run's total; plus a qualitative judgement of
  whether `summary` describes the same mechanism C found.

  **Ladder B (task 04, impact-analysis schema)** has no rule in that section — it defines
  only exploration and code review, and #006 scored task 04 ad hoc. Its rule is therefore
  stated here, and applies to all 24 Ladder B runs:
  - **Recall** = C's `callers_to_update` entries (file+line) also present in the run's, over
    C's total. Line numbers must match the same call site, not the same file.
  - **Precision** = the run's `callers_to_update` entries corroborated by C, over the run's
    total.
  - **Decoy penalty, scored separately and reported as its own column.** Task 04 exists to
    test one specific mistake: listing the `burler.go:373` same-named-but-unrelated call
    site as a real caller. Any run whose `callers_to_update` contains a call site C
    identifies as a lookalike is marked `decoy_admitted: true`. This is not folded into
    precision, because it is the finding the task was built for and averaging it away into a
    ratio would hide it.
  - **`excluded_lookalikes` is scored as credit, never as a requirement.** C's own
    unbounded-effort answer lists lookalikes (e.g. `webster.go:75`) that #006's scorecard
    explicitly declined to penalise the bounded arms for missing, since they sit outside the
    literal ask. A run that names a lookalike C also names is recorded as
    `lookalikes_matched: N` for qualitative comparison; a run that names none loses no
    points.
- Rationale for the split: reusing #006's exploration rule verbatim keeps Ladder A's numbers
  comparable to the committed 2026-08-28 scorecards. Ladder B needs its own rule because
  none exists, and writing it here rather than leaving it to the harness author is what
  stops 24 of 45 runs from being scored by improvised judgement. Separating the decoy from
  the ratios reflects what the task actually measures.
- Rationale: matching "the same real finding" across two independently-worded answers needs
  semantic judgement, which is why #006 made it an agent call rather than string equality;
  reusing that definition keeps this suite's numbers comparable to #006's. Blinding the
  scorer to the config is new and necessary here: with 14 configs whose whole point is that
  some should score worse, a scorer that knows which rung it is grading has an obvious path
  to confirming the expected pattern.
- Rejected: string/set equality on `relevant_files` (misses semantically identical answers
  worded differently, which is why #006 rejected it too). Producing a fresh fasit
  (unnecessary — a tracked, reviewed one exists for both tasks). A scorer that sees the
  config (invites confirmation bias in exactly the dimension being measured).

### Reporting discipline — medians, ranges, and no significance claims

- Decision: per configuration, report the median and full range of every metric across its
  3 runs. A rung is described as carrying benefit **only** when its range on that metric is
  disjoint from its own ladder's `none` control's range. Everything else is reported as
  "not separated at n = 3". The conclusion states n = 3 explicitly and makes no statistical
  significance claim.
- Rationale: n = 3 does not support significance testing, and #006's headline finding was
  retracted precisely because a single dramatic number was over-read. A disjoint-range rule
  is a blunt but honest bar that cannot be satisfied by one lucky run, and it makes every
  claim in the conclusion mechanically checkable against the tracked numbers.
- Rejected: t-tests / p-values at n = 3 (unsupportable). Mean-only reporting (one
  heavy-tailed run dominates). Ranking rungs by median alone with no separation
  requirement (would produce a confident-looking ordering that is mostly noise).

### Layout under `bench/loomyard-eval/ladder/`

- Decision:

  ```
  bench/loomyard-eval/ladder/
    README.md              # self-contained protocol; ladder tables; how to re-run
    ladder.yaml            # the 14 configs + warmth cell, declaratively: id, task,
                           # allowed tools, pinned run model, pinned scorer model/effort/prompt
    scripts/
      run_ladder.py        # generates deny-list settings + per-rung MCP preamble, builds
                           # worktrees, warms (or asserts cold), dispatches sequentially,
                           # collects raw, resumes by skipping completed runs
      extract_usage.py     # transcript JSONL -> usage.json
      score_run.py         # drives one blinded scoring call -> score.json
      summarize.py         # per-config medians/ranges -> the summary table
    tests/                 # TRACKED — pytest suite for the three deterministic units
      fixtures/            # TRACKED — hand-trimmed transcript JSONL fixtures
    results/<YYYY-MM-DD>/
      conclusion.md        # TRACKED — the synthesis
      summary.json         # TRACKED — per-config medians/ranges backing every claim
      raw/<config-id>/<n>/ # UNTRACKED — per-run artifacts:
        answer.json        #   the run's parsed output block
        usage.json         #   extracted metrics
        score.json         #   recall/precision/decoy_admitted/lookalikes_matched
        run.json           #   terminal-state marker
        transcript.jsonl   #   the raw session transcript
  ```

  Scoring is driven by `run_ladder.py` immediately after each run's metrics are extracted,
  by invoking `score_run.py` for that run. `score_run.py` dispatches one blinded scoring
  call under the three-input contract defined in the scoring decision above — the run's
  `answer.json`, the task's fasit `c.json`, and the task's `<TASK TEXT>`, and nothing else —
  and writes `score.json`. **A run is not
  complete until it is scored:** `run.json` is written only after the answer parsed, metrics
  extracted, gates passed, *and* `score.json` exists, so a resumed matrix re-runs anything
  whose scoring failed rather than silently summarising unscored cells.

  `ladder.yaml` is the single source of truth for the config table: `run_ladder.py`
  derives each run's deny-list from it, and the README's tables are kept consistent with it.
- Rationale: a sibling directory under `loomyard-eval` keeps the reused task files, pins,
  role preambles, and tracked fasit adjacent — this suite depends on all of them — while
  keeping its own protocol and results cleanly separate from #006's committed results,
  which must not be disturbed. `scripts/` follows the existing convention
  (`bench/loomyard-eval/scripts/gen_compact_toc.py`). A declarative `ladder.yaml` is what
  makes the "deny-list derived from the allow-set" decision enforceable rather than
  aspirational.
- Rejected: a new top-level `bench/capability-ladder/` (splits it from the tasks and fasit
  it depends on). Writing into `bench/loomyard-eval/results/` alongside #006's dated
  directories (mixes two methodologies with different tracking rules in one tree).

### Raw artifacts untracked, conclusion tracked

- Decision: add `bench/loomyard-eval/ladder/results/**/raw/` to the repo `.gitignore`.
  `conclusion.md` and `summary.json` under each dated results directory are tracked;
  everything under `raw/` is not.
- Rationale: the task body specifies exactly this convention. 45 runs' worth of answer
  JSON, usage JSON, and transcripts is disposable noise once summarised, and committing it
  would bloat the repo for no downstream reader's benefit. `summary.json` is tracked
  alongside the prose specifically so every claim in `conclusion.md` has its supporting
  numbers in the repo even after `raw/` is deleted — the conclusion must never be the only
  surviving evidence of its own basis.
- Rejected: tracking everything (#006's convention; the body explicitly changes it).
  Tracking only `conclusion.md` (leaves prose claims with no in-repo numbers behind them
  once raw is gone).

### This task executes the matrix and writes the conclusion

- Decision: the deliverable is the harness **and** a completed run of all 45 runs **and**
  the written, git-tracked conclusion synthesising them.
- Rationale: the task body's logging convention ("the written conclusion synthesized from
  it DOES need to be git-tracked") only makes sense if a conclusion is produced, and the
  open questions #006 delegated here are answered by data, not by a harness that could
  produce data. A harness with no run behind it would leave every retracted #006 claim
  still retracted and unreplaced.
- Rejected: harness-only, with execution deferred to a follow-up task (leaves the actual
  question unanswered and the two committed scorecard corrections dangling). Executing a
  reduced matrix to save cost (the whole point is per-capability attribution; dropping
  rungs drops findings).

## Technical context

**quarry's MCP surface.** Seven tools, registered in `internal/mcpserver/`:
`toc_dir` and `toc_file` (`tools_toc.go`), `textDocument_definition` and
`textDocument_references` (`tools_lsp.go`), `workspace_symbol` (`tools_symbol.go`),
`impact` (`tools_impact.go`), `assert_no_callers` (`tools_assert.go`). Client-side tool
names are `mcp__quarry__<tool>` — that is the exact string a deny-list entry must use.

Every tool takes a call-wide input with a `targets` **array**, so one call can address
many symbols or files. #006's task-01 scorecard identified this array shape as a real
confound in the bundle result: `toc_file`'s batching let the MCP arm survey 8 files in one
call where the CLI arm made many smaller calls, so "9 vs 26 tool calls" overstated the
per-call efficiency gain. This suite inherits that property. It is not a defect to correct
— batching ergonomics are part of what the MCP exposure *is* — but the conclusion must
report `tool_uses` alongside `num_turns` and token counts rather than treating tool-call
count as a standalone efficiency proxy.

**Entry shapes.** `lspEntry` (`internal/mcpserver/lspentry.go:34`) accepts
`textDocument`+`position` (0-based), bare `symbol`, or `textDocument`+`symbol`.
`nativeEntry` (`internal/mcpserver/nativeentry.go:20`) accepts `file`+`line`+`character`
(1-based, plain paths, no `file://`), bare `symbol`, or `file`+`symbol`. The 0-based vs
1-based split between the two families is deliberate and is worth watching in transcripts —
if restricted rungs show repeated schema errors on one family, that is a real usability
finding about the two-shape design. #006's warm task-01 run already logged one such error
(`within` passed call-wide instead of per-entry).

**Server launch.** `.mcp.json` at the repo root declares `go run ./cmd/quarry-mcp` with no
`--target-dir`, so the server takes its process working directory as the target and prints
`quarry-mcp: resolved target directory <path>` on stderr. Per `docs/mcp-setup.md`, quarry
needs `CGO_ENABLED=1` and a C toolchain, and a cold build cache can make the first
connection exceed the client's connect timeout. The harness must therefore use the
warm-start path — `go build -o quarry-mcp ./cmd/quarry-mcp` once, then point each run's
`--mcp-config` at the built binary with an explicit `--target-dir` for that run's worktree —
rather than `go run`. The built binary is already gitignored (`/quarry-mcp`).

**`--target-dir` is an instructed pin, not a structural one.** Every tool's call-wide input
carries an optional `targetDir` that overrides the launch default for that call (e.g.
`assertInput.TargetDir`, `internal/mcpserver/tools_assert.go`), so a model can retarget the
server per call. The generated preamble must therefore state that `targetDir` is never to be
set — the launch default is already correct — and the validation gates must check no run's
transcript contains a tool call carrying a `targetDir`. A run that retargets is invalidated,
since it breaks both the pinned-worktree constraint and, for the cold cell, the daemon key
the coldness argument depends on. The same applies to `buildTags`, which partitions the
state directory (`ResolveStateDir`) and would likewise split the key.

**Target codebase and pins.** Loomyard at `/home/knatte/Code/loomyard/wts/loomyard`. Task
01 pins to `975578cda8d6f3a81580bd4e73725e060211b766`; task 04 pins to the same SHA. Each
task's `Setup` section builds a disposable `git worktree` at its pin
(`/tmp/loomyard-eval-01`, `/tmp/loomyard-eval-04`) and removes it afterwards — runs never
point at the live main checkout. Note the repo-wide instruction against writing to system
temp directories applies to *this* repo's own scratch files, not to the disposable Loomyard
worktrees, whose paths are fixed by the already-committed task files and must not be
changed. For the 42 main-matrix runs the harness builds each task's worktree **once** and
shares it across that task's runs rather than creating and removing it per run; the 3
cold-cell runs are the deliberate exception and each get their own disposable worktree at a
distinct path, for the daemon-key reason given in the warmth decision.

**The shared worktree is writable, so it is restored between runs — but mutating it is a
legitimate route, not a disqualification.** Every run has Bash in its allow-set and the
worktree is an ordinary checkout, so a run can drop scratch files, populate a build cache,
or edit source. Leaving that state visible to the next 17 runs of the same task would be
exactly the cross-run contamination the blind-process decision exists to eliminate, so the
harness hard-restores the worktree (`git reset --hard` plus `git clean -fdx`) **after every
main-matrix run**, unconditionally, and records `worktree_dirtied: true|false` for that run
as an observation.

It does **not** invalidate a run for dirtying the tree. Task 04's own fasit reaches its
answer by a compiler experiment — `c.json`'s evidence fields record editing the interface
and running `go build -gcflags=-e` to read off the broken call sites — and the task text
frames the run as "you are about to change `Shuttle.Run`". Invalidating runs that do this
would systematically discard exactly the runs a low-capability rung is most likely to
produce (a `none` or `definition` rung has no other precise route), biasing the Ladder B
comparison the ladder exists to make. The compiler experiment is a real strategy an agent
may reasonably choose, and its cost is part of what this benchmark measures; `go build` is
neither cheap nor free, and a rung that has to reach for it should show that in its
duration and tool counts rather than vanishing from the sample.

`worktree_dirtied` is therefore reported per run and summarised per config: a rung where
most runs had to mutate the target to answer is a finding about that capability, and the
conclusion must read a cluster of dirtied runs that way rather than as noise.

**Reusable assets from #006.** The output schemas in `bench/loomyard-eval/README.md`
("Exploration tasks", "Impact-analysis tasks") and the **Agent B preamble** are used
verbatim. The `none` configs use the B preamble unchanged; it must never mention quarry.

The **Agent A preamble must be rewritten, not reused.** The committed template at
`bench/loomyard-eval/README.md` (the "Agent A preamble" block) is #006's *CLI* preamble: it
states `Binary: /tmp/quarry-bench` and documents shell verb syntax (`quarry toc dir <path>`,
`quarry refs <symbol> --target-dir <TARGET_DIR>`, …). The README contains no MCP tool names
anywhere. Trimming its verb list per rung would still hand every quarry-enabled agent a
binary path this task's scope forbids building — the MCP-flavoured preamble #006 actually
used for its MCP arm lived only in an uncommitted, gitignored scratch runbook
(`.scratch/bench-mcp-runbook.md`, cited in `results/2026-08-29/*/usage.json`'s `_note`) that
no longer exists. So: for each quarry-enabled rung the harness generates a fresh
**MCP-shaped** A preamble that names the rung's allowed tools by their client-side MCP names
(`mcp__quarry__toc_file`, …), describes them as tool calls with `targets`-array inputs,
mentions no binary path and no shell verb syntax, and carries over only the
exposure-independent guidance worth keeping from the CLI template: prefer quarry over grep,
do not re-verify a quarry answer with grep, and pass a known `file:line:character` position
instead of a bare symbol name when one is already in hand.

**The parallel-tool-calls block is carried into every generated preamble, verbatim.** Both
committed preambles open with `USE PARALLEL TOOL CALLS` and close with the
`<use_parallel_tool_calls>` block, and the `none` configs keep it by using the B preamble
unchanged. Dropping it from the rewritten A preamble would leave every quarry rung
batching-unsteered against a batching-steered control — a second confound landing squarely
on `num_turns` and `duration_ms`, the metrics the disjoint-range benefit rule leans on
hardest, and one with no justification behind it (unlike the anti-grep steering, which is
deliberately quarry-specific). It is exposure-independent text, so it is carried over
unchanged and asserted in the preamble-generation tests.

**The control's preamble differs in steering, not only in tools — and that confound is
recorded, not papered over.** The B preamble gives standard tools with no steering, while
every quarry rung's A preamble says "prefer quarry over grep, do not re-verify with grep".
So a rung-vs-`none` delta mixes capability with prompt steering, and the grep counts are the
worst affected: `bash_grep_count` and `grep_tool_count` are directly suppressed in quarry
rungs by an instruction the control never receives. Two dispositions were available:
neutralise the preambles, or record the confound. **This suite records it**, because the A
preamble's anti-grep steering is not incidental scaffolding — #006 added it deliberately
after observing task 01's first run spend a third of its Bash calls on grep despite having
quarry, and removing it would benchmark a configuration nobody would actually ship. The
consequences are binding on the conclusion:
- Every rung-vs-`none` delta is reported as "capability + steering", never attributed to the
  capability alone.
- The grep counts are **never** compared between a quarry rung and `none`. They are compared
  only *between quarry rungs*, which all share identical steering and differ only in tools —
  where they do cleanly measure how far a restricted rung routes around its own missing
  capability.
- Rung-vs-rung comparisons within a ladder (e.g. `toc_file` vs `toc_pair`) are unaffected by
  this confound entirely, since those preambles are identical except for the tool list. The
  cleanest attribution claims this suite can make are therefore between quarry rungs, and
  the `none` control is best read as a floor rather than as a matched pair.

Tracked fasit for scoring:
`bench/loomyard-eval/results/2026-08-28/01-reed-geometry-exploration/c.json` and
`.../04-shedadapters-shuttle-impact/c.json`.

**Prior numbers for orientation** (not for direct comparison — different methodology):
task 01 A-mcp cold 9 tool calls / 113.9s, A-cli 26 / 193.1s, B (no quarry) 22 / 133.7s;
task 01 A-mcp warm-blind 8 / 85.8s. Task 04: all arms reached the same correctness, with
warm-blind at 92.9s vs cold 93.0s.

**Repo conventions.** Go module `github.com/Knatte18/quarry`. Existing bench scripts are
Python under `bench/loomyard-eval/scripts/`. `.scratch/` is gitignored and is the correct
place for this repo's own ephemeral files. There is no `CONSTRAINTS.md` and no root
`CLAUDE.md`.

## Constraints

- `CGO_ENABLED=1` and a working C toolchain are required to build anything that links the
  `toc` verbs' tree-sitter backend, including `cmd/quarry-mcp`. A missing toolchain fails
  at compile time naming `quarry_requires_CGO_ENABLED_1_with_a_C_toolchain`
  (`internal/quarryengine/cgoguard_nocgo.go`).
- The Loomyard checkout must exist at `/home/knatte/Code/loomyard/wts/loomyard`. If it does
  not, stop and ask rather than substituting another repo — the task files reference
  specific real files and commits at specific pins.
- No run may point at the live Loomyard main checkout; every run uses a pinned worktree.
- No quarry source is modified by this task.
- `bench/loomyard-eval/results/` (#006's committed results) is read-only for this task.
- The `none` arms must never encounter the word "quarry" in their prompt or their reachable
  filesystem — #006's structural-blinding rule carries over unchanged.
- 45 headless agent runs is a real cost. The harness must be resumable: if a run fails or
  the batch is interrupted, re-invoking it must skip already-completed runs rather than
  re-running the whole matrix.
- One model, pinned identically across all 45 runs and recorded in the results. A matrix
  spanning two models is not a matrix.
- `QUARRY_STATE_DIR` cleared and `--state-dir` never passed, for every run — both take
  precedence over `workspaceKey` in `ResolveStateDir` and would collapse the daemon keying
  the cold cell depends on.

## Testing

This task's product is a benchmark harness plus a data-backed conclusion, so "tests" means
unit tests on the deterministic harness pieces plus explicit validation gates on the run
protocol. The runs themselves are not tests and must not be asserted on.

**Infrastructure.** The repo has no Python test setup today (`bench/loomyard-eval/scripts/`
holds one standalone script, and there is no `pyproject.toml`, `pytest.ini`, or `conftest.py`
anywhere). This task adds a minimal one, scoped to the ladder directory: `pytest` tests under
`bench/loomyard-eval/ladder/tests/`, run with `python -m pytest bench/loomyard-eval/ladder/tests`
from the repo root, with a `conftest.py` there putting `scripts/` on the import path. Fixture
transcripts live in `tests/fixtures/` and **are** git-tracked — they are small, hand-trimmed
JSONL excerpts (a handful of representative lines each, not whole sessions), and the
`results/**/raw/` untracking rule does not reach them. They are test inputs the suite cannot
be re-verified without, which is the opposite of the disposable per-run artifacts that rule
exists for.

**`extract_usage.py` — the strongest TDD candidate.** Its whole job is turning transcript
JSONL into numbers that #006 proved cannot be trusted when produced by hand. Write it
against committed fixture transcripts before writing the extractor. Scenarios: per-tool
call counting across mixed quarry and non-quarry tools; `bash_grep_count` counting Bash
commands matching `grep`/`rg` and **nothing else** (this is the #006-comparable field, so a
Grep tool call must not increment it); `grep_tool_count` counting native Grep tool calls and
nothing else; `grep_fallback_total` deriving as the sum of the two and never being conflated
with either; counting a denied tool attempt as an attempt, not as a successful call; every
token class extracted separately with none silently summed; a run with zero tool calls; a
transcript containing an errored tool result.

**Deny-list generation — the second TDD candidate.** Given a config's allowed set from
`ladder.yaml`, the generated `permissions.deny` must be exactly the seven
`mcp__quarry__*` names minus the allowed ones. Scenarios: the `none` config denies all
seven; the `bundle` config denies none; a single-tool config denies exactly six; a tool
name added to the canonical seven appears in every restricted config's deny-list without
any per-config edit (this is the drift guard the decision above depends on, so it must be
asserted directly); no non-quarry tool ever appears in a deny-list.

**`summarize.py`.** Median and range per metric per config, and the disjoint-range
separation rule. Scenarios: an even/odd run count; a config where ranges overlap the
control's is reported as "not separated"; a config with a disjoint range is reported as
separated; a config with a failed (missing) run is reported as incomplete rather than
silently summarised from two runs.

**Preamble generation.** Assert that a rung's generated A-preamble names exactly the
`mcp__quarry__*` tools that rung allows and no others; that it contains no binary path and
no CLI verb syntax (`/tmp/quarry-bench`, `quarry toc dir`, `--target-dir`, and the like), so
the CLI template can never leak back in; that it forbids setting `targetDir`/`buildTags`;
that it carries the `USE PARALLEL TOOL CALLS` opening and the `<use_parallel_tool_calls>`
block verbatim, exactly as the `none` control's B preamble does (this is the batching-steer
parity the `num_turns`/`duration_ms` comparison depends on); and — as a distinct assertion —
that every `none` config's prompt contains no occurrence of "quarry" anywhere.

**Cold-cell coldness.** Assert that the three cold runs each resolve to a distinct
target-dir path and therefore a distinct daemon key; that no pre-warm step runs for them;
that a pre-existing `daemon.json` under the resolved state directory rejects the run rather
than proceeding; and that a run finishing with **no** `daemon.json` present is invalidated
as a native-fallback rather than reported as cold.

**Resumability.** Assert that a run directory with a `state: "complete"` `run.json` is
skipped; that one without it is re-run; that invalidation removes `run.json` and moves the
attempt to `<n>.invalid-<k>/` without destroying it; and that a run whose `score.json` is
missing is treated as incomplete even when its answer and usage files are present.

**Protocol validation gates, checked per run before its `run.json` is written.** These are
verification steps over the produced data, not unit tests: the run produced a parseable
answer block and a `usage.json`; its transcript shows no successful call to a tool its
config denied; no tool call in it carries `targetDir` or `buildTags`; it ran under the
pinned model; a `none` run's transcript contains no `mcp__quarry__*` tool and no occurrence
of "quarry"; a cold-cell run's state directory held no `daemon.json` before and did hold one
after; and a `score.json` exists, produced by the pinned scorer. Worktree dirtiness is
**recorded, not gated** — see the worktree decision. A run failing a gate is retried at most
3 times, after which the matrix halts. Across the matrix: each config has exactly 3 complete runs, and no
reported number came from a self-report field. A failed gate invalidates that run per the
resumability decision — it is re-run, never reported.

**Not tested:** quarry's own behaviour (out of scope), the wording of the conclusion, and
anything requiring network or live model calls inside the unit-test suite — the harness's
dispatch layer is exercised by actually running the matrix, not by mocking a model.

## Q&A log

- **Q:** What are the ladder's exact rungs and ordering? **A:** [auto-pick] Two
  task-shaped ladders — 6 exploration configs on task 01, 8 navigation/impact configs on
  task 04, each with its own `none` control and `bundle` ceiling. **Why:** a full cross
  product is ~70 runs for information the rung-to-task-shape pairing already yields, and
  pairing avoids spending runs on combinations (`assert_no_callers` on open-ended
  exploration) that predictably measure nothing.
- **Q:** How many repetitions per configuration? **A:** [auto-pick] N = 3. **Why:** the
  body specifies N ≥ 2–3; three is the smallest N giving a median rather than a
  two-run mean, which matters because agent runs are heavy-tailed, and it keeps the matrix
  at 45 executable runs.
- **Q:** Reuse loomyard-eval's existing tasks or design new narrower ones? **A:**
  [auto-pick] Reuse 01 and 04 unchanged; exclude 02 and 03. **Why:** 01 and 04 have
  tracked, reviewed fasit so no new reference runs are needed and results stay comparable
  to #006; 02 has no fasit and 03 was shown non-discriminative by its own postmortem. A
  task shaped to make one capability shine would be a rigged measurement.
- **Q:** How is a rung's capability restriction actually enforced? **A:** [auto-pick]
  Headless `claude -p` per run with a generated `--settings` deny-list derived from
  `ladder.yaml`'s allow-set. **Why:** the body requires structural rather than instructed
  restriction, and a separate process additionally makes every run blind — closing the
  orchestrator-contamination hole that forced the retractions in #006's task 03 and 04
  scorecards.
- **Q:** Do quarry-restricted arms keep Grep/Bash/Read? **A:** [auto-pick] Yes, in every
  arm including the controls; only quarry MCP tools are ever denied, and no quarry CLI
  binary is built for the runs. **Why:** the real question is whether a capability improves
  an agent that also has grep, and keeping grep preserves the grep-fallback counts as a live
  signal about how much agents route around a missing capability.
- **Q:** Warm or cold daemon? **A:** [auto-pick] Warm everywhere in the main matrix, plus
  one dedicated 3-run cold cell on task 01's `bundle` config. **Why:** warmth as a second
  dimension would double the matrix for a question orthogonal to per-capability
  attribution, while one cell still answers the warm-vs-cold question two committed
  scorecards explicitly delegated to this task.
- **Q:** What exactly does each run report? **A:** [auto-pick] Full benchmark accounting —
  duration, every token class separately, cost, turns, total tool calls, a per-tool
  breakdown covering quarry and non-quarry tools alike, quarry subtotal,
  `bash_grep_count` / `grep_tool_count`, and `denied_tool_attempts` — all extracted
  programmatically from
  the run's transcript JSONL. **Why:** #006 caught self-report discrepancies twice, and
  its own note records that a single collapsed token figure made its two run dates
  incomparable; 45 runs is far past what hand-verification can cover.
- **Q:** How is correctness scored? **A:** [auto-pick] Against the existing tracked
  `c.json` fasit using #006's recall/precision definitions, by a scoring agent blinded to
  the config id. **Why:** semantic matching needs judgement rather than string equality,
  and with 14 configs whose whole premise is that some should score worse, a scorer that
  knows the rung has an obvious path to confirming the expected pattern.
- **Q:** How strong a claim may the conclusion make? **A:** [auto-pick] Medians and full
  ranges per config; a rung "carries benefit" only when its range is disjoint from its
  ladder's `none` control; no significance claims, n = 3 stated explicitly. **Why:** n = 3
  cannot support significance testing, and #006's headline finding was retracted precisely
  because one dramatic number was over-read.
- **Q:** Where does the suite live and what gets committed? **A:** [auto-pick]
  `bench/loomyard-eval/ladder/`, with `conclusion.md` and `summary.json` tracked per dated
  results directory and `results/**/raw/` gitignored. **Why:** a sibling of the tasks and
  fasit it depends on, without disturbing #006's committed results;
  `summary.json` is tracked alongside the prose so every claim keeps its supporting numbers
  in-repo after the disposable raw artifacts are deleted.
- **Q:** Does this task execute the matrix, or only build the harness? **A:** [auto-pick]
  Both — build it, run all 45, and write the tracked conclusion. **Why:** the body's
  tracked-conclusion convention presupposes a conclusion exists, and a harness alone would
  leave every retracted #006 claim unreplaced.
- **Q:** How does the cold-daemon cell actually achieve a cold daemon, given the shared
  task-01 worktree and the 10-minute idle timeout? **A:** [auto-pick] Each of the 3 cold
  runs gets its own freshly-built worktree at a distinct path, so it has a distinct
  `workspaceKey` and an unstarted daemon; the harness asserts coldness before each run
  rather than assuming it. **Why:** the daemon is keyed per absolute target-dir path
  (`internal/cli/paths.go:76`) and only self-expires after 10 minutes
  (`internal/quarryengine/daemon/ensureserver.go:143`), so any time-ordering rule against
  the shared worktree would silently degrade into warm-vs-warm — and a contaminated cold
  cell is worse than none, because it looks like data.
- **Q:** Sequential or concurrent run dispatch? **A:** [auto-pick] Sequential, all 45.
  **Why:** `duration_ms` is what the disjoint-range separation rule leans on hardest, and
  concurrent runs contend for CPU, the gopls daemon, and rate limits, which could
  manufacture or erase a separation; sequential dispatch also underpins the cold cell and
  the resumability constraint.
- **Q:** Can #006's committed "Agent A preamble" be reused with its verb list trimmed per
  rung? **A:** [auto-pick] No — it must be rewritten as an MCP-shaped preamble per rung.
  **Why:** the committed template is the CLI one (`Binary: /tmp/quarry-bench`, shell verb
  syntax, no MCP tool names anywhere in the README), so trimming its verb list would still
  point agents at a binary this task's scope forbids building; #006's actual MCP preamble
  only ever existed in an uncommitted scratch runbook that is gone.
- **Q:** Which model and permission configuration do the 45 headless runs use? **A:**
  [auto-pick] One model id pinned identically across all 45 and recorded in `ladder.yaml`
  and every `usage.json`; an identical `permissions.allow` set for Read/Grep/Glob/Bash in a
  non-interactive permission mode; `QUARRY_STATE_DIR` cleared and `--state-dir` never
  passed. **Why:** every headline metric — duration, all four token classes, cost, turns —
  is model-dependent, so an unpinned model makes cells incomparable and the matrix
  unreproducible; the state-dir vars take precedence over `workspaceKey` and would collapse
  the cold cell's daemon keying.
- **Q:** How are Ladder B's 24 runs scored, given the README defines recall/precision only
  for exploration and code review? **A:** [auto-pick] An explicit impact-analysis rule
  written into this discussion: recall/precision over `callers_to_update` matched on
  file+line, with the `burler.go:373`-style decoy scored as its own `decoy_admitted` column
  rather than folded into precision, and `excluded_lookalikes` credited but never required.
  **Why:** no rule exists (#006 scored task 04 ad hoc), and leaving it to the harness author
  would mean 24 of 45 runs scored by improvised judgement; the decoy is the finding the task
  was built for, so averaging it into a ratio would hide it.
- **Q:** Do the `none` arms still get `--mcp-config` with all seven tools denied? **A:**
  [auto-pick] No — they are launched with no quarry server declared at all, and every run's
  cwd is its own task worktree, never this repo. **Why:** a declared server named `quarry`
  exposing a `mcp__quarry__*` namespace is precisely the structural leak the blinding
  constraint forbids; denying the tools still advertises the name.
- **Q:** Can `denied_tool_attempts` actually be non-zero? **A:** [auto-pick] Unknown until
  probed — a throwaway probe run before the matrix establishes whether `permissions.deny`
  hides denied tools or advertises and rejects them, and the metric is dropped rather than
  reported if they are hidden. **Why:** if denied tools never reach the model, the column is
  identically zero everywhere and measures nothing; reporting those zeros as evidence would
  be worse than not reporting the metric.
- **Q:** Is per-worktree-path keying enough to guarantee the cold cell is cold? **A:**
  [auto-pick] No — the harness must also assert a supervised connection per cold run and
  treat a native fallback as invalidated. **Why:** `workspaceKey` derives the socket only on
  the supervised strategy; `nativeArgv`'s own comment states gopls picks the shared
  `-remote=auto` address and that it "is not a function of the state directory at all", and
  the native path writes no `daemon.json`, so a state-absence check would pass vacuously
  while three "cold" runs all joined one warm daemon.
- **Q:** How does a resumed run distinguish a completed run from an invalidated one? **A:**
  [auto-pick] A `run.json` written only after the answer parsed, usage extracted, and gates
  passed; invalidation deletes it and moves the attempt to `<n>.invalid-<k>/`. **Why:**
  "skip already-completed runs" and "invalidate and re-run a failed cell" are
  indistinguishable on disk without an explicit terminal-state marker.
- **Q:** Is `--target-dir` a structural pin? **A:** [auto-pick] No — every tool's call-wide
  input carries an optional `targetDir` overriding the launch default, so the preamble must
  forbid setting it (and `buildTags`) and a validation gate must invalidate any run whose
  transcript sets either. **Why:** a retargeting call breaks both the pinned-worktree
  constraint and the daemon key the cold cell's coldness argument rests on.
- **Q:** Task 04's committed scorecard shows both #006 arms at 100% — doesn't that make it
  non-discriminative, the same reason task 03 was excluded? **A:** [auto-pick] No, but state
  it explicitly: Ladder B is expected to discriminate on efficiency, not correctness. **Why:**
  task 03 was excluded because `go build` answers its question outright, leaving no arm's
  process meaningfully different; task 04's arms reach the same answer by visibly different
  routes, which is the per-capability question. Saying so up front also stops the conclusion
  from later reading a saturated correctness axis as evidence that every rung "works".
- **Q:** How does the harness detect supervised-vs-native for a cold run, given quarry
  can't be modified? **A:** [auto-pick] From the state directory: assert no
  `<stateDir>/<lang>/daemon.json` before the run, and assert one appeared after. **Why:**
  `cmd/quarry-mcp/main.go` writes exactly one stderr line (the resolved target dir) and
  `ConnKind` is never surfaced to a client, so there is no stderr signal to read; the
  supervised strategy writes `daemon.json` (`daemonstate.go`) and the native strategy writes
  nothing, which makes its post-run presence the only source-grounded positive confirmation
  available.
- **Q:** The shared task worktree is described as read-only — is it? **A:** [auto-pick] No,
  it's an ordinary writable checkout and every run has Bash, so the harness hard-restores it
  after every main-matrix run unconditionally. **Why:** a run that drops a scratch file or
  populates a build cache would otherwise leave state visible to the next 17 runs of that
  task — the exact cross-run contamination the blind-process decision exists to eliminate.
  (Whether dirtying the tree also *invalidates* the run is settled separately below: it does
  not.)
- **Q:** Where does scoring actually happen, and is a run complete before it's scored?
  **A:** [auto-pick] `run_ladder.py` invokes `score_run.py` right after metrics extraction;
  it writes `score.json` per run; and a run is **not** complete until `score.json` exists.
  **Why:** without a named producer, storage location, and completion semantics, a resumed
  matrix would skip unscored runs and summarise cells that were never scored.
- **Q:** Which model id, concretely? **A:** [auto-pick] The operator sets it in
  `ladder.yaml` once before the matrix starts; the harness refuses to run with it unset, and
  a gate rejects any run reporting a different model. **Why:** the right id depends on
  45-run cost and on what the operator wants comparability against, and a model id
  hard-coded into a design document rots as soon as the available models change — the
  pinning mechanism is what matters, not the particular value.
- **Q:** The `none` control gets the B preamble while every quarry rung gets a steering A
  preamble — doesn't that confound every rung-vs-control delta? **A:** [auto-pick] Yes, and
  it is recorded rather than neutralised: rung-vs-`none` deltas are reported as "capability
  + steering", grep counts are never compared between a quarry rung and `none` (only between
  quarry rungs, which share identical steering), and `none` is read as a floor rather than a
  matched pair. **Why:** the anti-grep steering is not incidental — #006 added it after
  watching task 01's first run spend a third of its Bash calls on grep despite having quarry
  — so removing it would benchmark a configuration nobody would ship; rung-vs-rung
  comparisons within a ladder are unaffected and are where the cleanest attribution lives.
- **Q:** A run that edits the target and runs `go build` dirties the shared worktree — is it
  invalidated? **A:** [auto-pick] No. The worktree is hard-restored after every run
  unconditionally, and `worktree_dirtied` is recorded as an observation, not a gate. **Why:**
  task 04's own fasit reaches its answer by exactly that compiler experiment
  (`c.json`'s evidence fields cite `go build -gcflags=-e`), and the task text frames the run
  as "you are about to change `Shuttle.Run`" — invalidating those runs would systematically
  discard the ones low-capability rungs are most likely to produce, biasing the very
  comparison Ladder B exists to make.
- **Q:** What bounds the invalidate-and-re-run loop? **A:** [auto-pick] 3 attempts per run;
  on the third failure the whole matrix halts with the failing gate reported, completed runs
  intact. **Why:** several gates fail deterministically (denial not blocking, model
  mismatch, persistent native fallback), so an uncapped loop burns paid runs forever; and
  such a failure usually invalidates the other cells' premises too, making a full halt
  better than discovering it 40 runs later.
- **Q:** Does the probe verify that `permissions.deny` blocks MCP tools at all, or only
  whether denied tools are hidden? **A:** [auto-pick] Both — it first asserts a denied
  `mcp__quarry__*` call does not succeed, and halts before any matrix run if it does.
  **Why:** this is the load-bearing premise of the whole suite; if denial did not block,
  every rung would silently be the full bundle and all 45 runs would be worthless.
- **Q:** Is the scoring agent pinned the way the runs are? **A:** [auto-pick] Yes — model,
  effort, and prompt template fixed in `ladder.yaml` and recorded in every `score.json`; its
  input is the run's answer, the fasit, and the task text, nothing else. **Why:** an
  unpinned scorer drifting across the matrix shifts scores between rungs graded at different
  times, which is the same defect as an unpinned run model but harder to spot, since it
  moves the correctness numbers rather than the cost ones. The task text is needed to judge
  whether `summary` describes the same mechanism, and is identical across a ladder's rungs,
  so it reveals nothing about the config.
- **Q:** Where do the tests and fixtures live, and how are they run? **A:** [auto-pick]
  `pytest` under `bench/loomyard-eval/ladder/tests/` with a local `conftest.py`, fixtures
  git-tracked in `tests/fixtures/`, run via `python -m pytest bench/loomyard-eval/ladder/tests`.
  **Why:** the repo has no Python test infrastructure at all today, so it has to be named
  rather than assumed; the fixtures are small hand-trimmed excerpts the suite cannot be
  re-verified without, which is the opposite of the disposable raw run artifacts the
  untracking rule targets.
- **Q:** How is the transcript captured? **A:** [auto-pick] `--output-format stream-json`
  captured directly to `transcript.jsonl` as each run executes, not located afterwards by
  `session_id`. **Why:** locating a session file after the fact depends on client on-disk
  layout this suite does not control, and streaming means even a run that dies mid-way
  leaves a diagnosable transcript.
- **Q:** Are config ids unique across the two ladders? **A:** [auto-pick] Yes —
  ladder-qualified (`a0-none`, `b0-none`, `a5-bundle`, `b7-bundle`, cold cell
  `a5-bundle-cold`), and `ladder.yaml`'s `id` is the unique key, with a separate
  `ladder: a|b` field for control lookup. **Why:** bare `none`/`bundle` ids collide two
  ladders' runs into one `raw/<config-id>/<n>/` directory and break the `run.json` resume
  key, `summarize.py`'s grouping, and the "3 complete runs per config" gate; the cold cell
  needs its own id for the same reason.
- **Q:** Does the rewritten A preamble keep the parallel-tool-calls directive? **A:**
  [auto-pick] Yes, verbatim, and it is asserted in the preamble tests. **Why:** both
  committed preambles carry it and the `none` control keeps it via the unchanged B preamble,
  so dropping it would leave every quarry rung batching-unsteered against a batching-steered
  control — a second confound landing directly on `num_turns` and `duration_ms`, and unlike
  the anti-grep steering it has no quarry-specific justification for differing.
- **Q:** Where does the cold cell sit in the run order? **A:** [auto-pick] Last, after the
  entire 42-run main matrix, with each cold run's daemon drained before the next starts.
  **Why:** a supervised cold run leaves a gopls daemon resident for the 10-minute idle
  timeout against a worktree the harness then deletes; running the cell first would leave up
  to three of them resident through the opening of the main matrix, loading the very
  `duration_ms` figures the separation rule reads.
- **Q:** Is the `none` arms' blinding structural? **A:** [auto-pick] No — cwd removes the
  invitation but not the capability, since Bash can read absolute paths; the enforcement is
  the per-run transcript gate rejecting any `none` run mentioning quarry. **Why:** claiming
  structural blinding when the guard is detection-based would overstate the guarantee
  relative to #006's, and the conclusion has to be able to describe what was actually
  enforced.
- **Q:** If the review rounds hit the configured cap without converging, block or hand
  off? **A:** Hand off anyway (operator instruction, given mid-session). **Why:** the
  operator explicitly overrode auto-mode's block-on-non-progress behaviour for this task.
