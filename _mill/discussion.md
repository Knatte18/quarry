# Discussion: Ladder, toc rerun (T7)

```yaml
task: Ladder, toc rerun (T7)
slug: ladder-toc-rerun
status: discussing
parent: main
```

## Problem

The entire quarry rewrite rests on one measured number: a directory-level table of contents halved
exploration on unfamiliar code — turns 8 → 4, `cache_read` 127k → 83k, recall unchanged. That
number was measured on branch `v1-final`, by the V1 harness, against the V1 MCP server
(`v1-final:bench/loomyard-eval/ladder/results/2026-09-02-toc/conclusion.md`). Every part of that
stack has since been deleted and rewritten: T2 replaced the harness, T3–T5a replaced the engine and
the facade, T6 replaced the MCP server. **Nothing on `main` reproduces the number.**

T7 is the regression gate for the whole rewrite (plan §12, wave 5). It runs the toc ladder with the
new harness against the new server on the same Loomyard task at the same pin, and writes a
conclusion that either reproduces the separation or says honestly why not. Why now: T2 and T6 — its
two hard dependencies — are both merged on `main`, so T7 is the last task on the critical path
(`T0 → T1 → T3 → T5a → T6 → T7`), and until it has run the rewrite has no evidence behind it.

A second, smaller thing is settled here by construction: plan §11's open decision on whether the
harness's `results/**/raw/` tree is committed. T7 writes the first results root the new harness has
ever produced, so the decision is made and recorded with it.

## Scope

**In:**

- Recording the operator's plan §9a live probe of the merged `cmd/quarry-mcp` (connect, `toc` call,
  allowlist denial) as `probe.md` in the results root, and running the harness's own guarded
  `TestLive` smoke test as a cheap pre-matrix check of the worker's tooling.
- One matrix run: `bench/loomyard-eval/ladder/ladder-toc.yaml`, cells `a0-none` and `a2-toc-dir`,
  reps 5 (the file's own value), via `bench/loomyard-eval/ladder/cmd/ladder run`, against the
  server built from `./cmd/quarry-mcp` at this branch's tip.
- Resuming that same results root until each cell has 5 complete repetitions or the harness's
  `MaxAttempts` ceiling is reached.
- `bench/loomyard-eval/ladder/results/2026-09-04-toc/conclusion.md`, written in the shape of the
  `v1-final` conclusions, plus the machine artifacts the run produced (`summary.json`,
  `provenance.json`, `table.txt`).
- Recording plan §11's `results/**/raw/` decision in the conclusion, and updating
  `docs/rewrite-plan.md` §11 and `HANDOFF.md` §3/§4/§5 to match what T7 found.
- Any harness bug fix T7 is blocked by, with tests — restarting the matrix in a fresh results root
  when that happens.

**Out:**

- Ladder b (`b0-none`, `b8-toc-dir`). The task names two cells; ladder b is the negative control and
  is not part of the regression gate. The yaml keeps its rows; this run does not select them.
- Any change to `internal/engine`, `quarry/`, `internal/cli`, `internal/mcpserver` or
  `cmd/quarry-mcp`. These are the code under test. Editing them is forbidden while the matrix runs
  and unnecessary outside it — if the measurement exposes a defect there, the conclusion records it
  and it becomes a separate task.
- Any change to `ladder-toc.yaml`'s measured parameters (cells, reps, models, effort, `max_turns`,
  pins, tasks, fasit). Changing them would make this root incomparable with the thing it exists to
  reproduce.
- New ladder cells, new tasks, new fasit files, the compact form, annex delivery, a grep-toc
  control. All are named as deliberately-absent in `ladder-toc.yaml`'s own header and each needs
  harness or engine work first.
- T8 (the type checker) and everything else in plan §12 wave 6.
- Committing `results/**/raw/`. See the decision below.

## Decisions

### results-root-name

- Decision: `bench/loomyard-eval/ladder/results/2026-09-04-toc`.
- Rationale: the plan and the task both spell the path `results/<date>-toc`; every `v1-final` root
  uses a `YYYY-MM-DD` prefix, and the `-toc` suffix distinguishes this matrix from the other ladder
  files that will one day write roots on the same day. If the matrix's first successful invocation
  happens on a later date, use that date instead — the root is named for the run, not for the task
  branch.
- Rejected: an undated name (`toc-rerun`), which breaks the existing convention and stops the roots
  from sorting chronologically.

### cells-and-reps

- Decision: `--cells a0-none,a2-toc-dir`, with no `--reps` override, so the file's `reps: 5`
  applies. `max_turns: 60`, `run_model: claude-sonnet-5`, `run_effort: medium`, scorer
  `claude-opus-5` at effort `high` — all taken from the file, none overridden on the command line.
- Rationale: this is exactly the invocation plan §12's T7 row and `ladder-toc.yaml`'s own closing
  comment specify. Ten measured repetitions plus ten scorer invocations is the whole matrix.
- Rejected: running all four cells. Ladder b answers a different question (does toc help when the
  task already names the file); it is not the regression gate, it doubles the cost, and its absence
  costs the conclusion nothing.

### raw-tree-is-ignored

- Decision: `results/**/raw/` stays untracked. `bench/loomyard-eval/ladder/.gitignore` already
  carries `results/*/raw/`, which T2 shipped; T7 confirms that as the settled answer to plan §11 and
  records it in the conclusion. `docs/rewrite-plan.md` §11's bullet is updated from "T2 decides" to
  the decision and its reason.
- Rationale: two independent reasons. First, `raw/memory-paths.json` holds resolved auto-memory
  directory paths, and the repository's own rule is that no tracked file carries a machine path —
  `provenance.json` goes to the length of storing sha256 hashes of those paths precisely to honour
  that rule, which committing `raw/` would defeat. Second, the raw tree is ten transcripts of a
  60-turn ceiling plus their answers, scores and usage files: large, per-host, and fully summarised
  by the three committed artifacts. The V1 record confirms the norm ("the raw run data was never
  committed anywhere", HANDOFF §3).
- Rejected: committing `raw/`. It would leak machine paths, add megabytes of transcripts per root,
  and force redaction work no reader has asked for. Also rejected: leaving §11 open, since the task
  explicitly says T7's first results root decides it.

### probe-is-reported-done

- Decision: the plan §9a live probe **has been reported done** and this task does not re-run it. The
  operator ran it on 2026-09-04 against the merged `cmd/quarry-mcp` built from `main`, harness-
  faithfully, and both halves were green: with the allowlist, the server connected and an
  `mcp__quarry__toc` call returned the §4 envelope; without it, the call was refused and landed in
  `permission_denials`. That report is written up as
  `bench/loomyard-eval/ladder/results/2026-09-04-toc/probe.md` — the operator's report, not fresh
  transcripts — and the matrix proceeds. The task's constraint stands as written: the probe runs
  before the matrix, and if it had not been reported done the correct action would have been to stop
  and ask.
- What `probe.md` contains: it **transcribes the orchestrator's round-1 review answer verbatim** —
  quoted, attributed to the round-1 discussion review at
  `_mill/reviews/20260904-110823-discussion-review-r1.md`, and labelled as an operator report rather
  than a transcript this task captured. No operator artifact exists in the tree to copy, so the
  review file is the source of record and must be named as such. Minimum fields: the date
  (2026-09-04); what was probed (the merged `cmd/quarry-mcp`, built from `main`); the three §9a
  properties and their outcomes (server connected and listed in the `system` record; an
  `mcp__quarry__toc` call returned the §4 envelope; a call outside the allowlist refused and present
  in `permission_denials`); the flags that made the denial half meaningful, in particular
  `--setting-sources ""`; and the `claude` CLI version the probe ran under, or an explicit statement
  that it was not recorded — plan §9a's table was taken on 2.1.259 and this host is on 2.1.236, so a
  silent omission would read as an agreement it has not earned. The file says plainly that it is a
  report, not evidence this task generated.
- Rationale: the constraint asks a question, and the answer arrived through the round-1 review: the
  probe is done. Re-running it would spend an API call to re-derive a fact already established, and
  self-authorising past a stop-and-ask constraint on a premise the orchestrator can disprove is the
  wrong instinct regardless of how cheap the command is. `probe.md` still exists because plan §12's
  T6 done-when and this task's constraints both name the probe as a gate, and a results root with no
  record of it read as unverified.
- Kept from the earlier draft: `env -u CLAUDECODE -u CLAUDE_CODE_ENTRYPOINT LADDER_LIVE_TEST=1 go
  test ./bench/loomyard-eval/ladder/internal/ladder -run TestLive -v` runs once before the matrix.
  That is the harness's own guarded smoke test — the worker's tooling, not the operator's probe — and
  it is cheap insurance that the `claude -p` seam works on this host before ten repetitions depend on
  it. It carries the same `env -u` prefix as the matrix (see `matrix-runs-backgrounded`): the test
  exists to make a claim about that seam under the conditions the matrix runs in, and running it
  under the very markers the matrix strips would test something else. A failure blocks the matrix.
- Rejected: re-running the operator's `claude -p` probe (duplicates a reported-done gate). Rejected:
  proceeding with no `probe.md` at all (the results root would carry no record of a gate the plan
  names). Rejected: rewriting the constraint to say the worker runs the probe — the artifact must
  state the constraint as the task wrote it.

### matrix-runs-backgrounded

- Decision: the matrix runs as a detached background process with its stdout and stderr tee'd to a
  log under `.scratch/`, and the driving batch polls that log rather than blocking on the command.
  The invocation strips the driving session's own Claude Code markers:

  ```
  cd <worktree-root> && \
    env -u CLAUDECODE -u CLAUDE_CODE_ENTRYPOINT \
      go run ./bench/loomyard-eval/ladder/cmd/ladder run \
        --config bench/loomyard-eval/ladder/ladder-toc.yaml \
        --results bench/loomyard-eval/ladder/results/2026-09-04-toc \
        --cells a0-none,a2-toc-dir
  ```

- Rationale: ten measured runs at ~60 s each plus ten Opus scorer calls is 20–40 minutes of
  wall-clock, far beyond any single foreground tool call's ceiling; a killed invocation would leave
  the run lock held and half a root on disk. Backgrounding with a polled log is the pattern the rest
  of this repository's long operations already use. The `env -u` is because the harness's measured
  `claude -p` children inherit the driving agent's environment: `CLAUDECODE` and
  `CLAUDE_CODE_ENTRYPOINT` announce "you are running inside a Claude Code session", which is not a
  condition the V1 measurement ran under and not one this run wants to introduce silently.
- Rejected: a foreground call (times out). Rejected: handing the command to the operator (the task
  is dispatched precisely so it runs unattended). Rejected: inheriting the environment unchanged
  (an unrecorded difference from the thing being reproduced).

### clean-tree-before-the-matrix

- Decision: every `_mill/` artifact and every source change is committed before the matrix's first
  invocation, and the run batch aborts if `git status --porcelain` is non-empty at that moment. No
  file in the repository is edited between the first repetition and the last — including
  `conclusion.md`, which is written only after the run finishes.
- Rationale: this is the harness rule that carried over from V1 verbatim (HANDOFF §3, task
  constraints): never edit the code under test mid-matrix. A dirty tree makes the committed record
  say "measured against something not in git", which is the exact failure the 2026-09-02 run's
  post-mortem identified and fixed.
- **`WarnOnServerHashDrift` is not the safety net here, and the plan must not treat it as one.**
  `run.go` builds the server once per invocation and then writes that one hash into
  `prov.ServerHashes[repKey(cell, rep)]` for *every* rep `1..reps_effective` of every non-control
  cell — including reps a previous invocation already completed — so a resumed invocation overwrites
  the earlier invocation's hashes with its own. `CollectInvocation` leaves `Invocation.ServerHashes`
  nil, so no per-invocation history survives either. After any resume the map holds exactly one
  distinct hash and the drift warning cannot fire, by construction. Since `resume-not-restart` makes
  resuming the normal path, the detector is silent in precisely the workflow this task uses.
- What the run does instead, two independent checks the plan must perform explicitly:
  1. **`quarry_commit` equality across `provenance.json`'s `invocations[]`.** Every invocation
     records its own; if two differ, the root mixed two versions of the code under test and is void.
     This is the real drift check under resume and it survives the overwrite above, since
     `invocations[]` is appended to, never rewritten.
  2. **The server hash captured out-of-band, per invocation.** Before each invocation of the matrix
     command, record `sha256sum` of the built server binary (`<worktree-root>/bin/quarry` under the
     resolved worktree root, `~/.cache/ladder-eval` on this host — the harness's own build target)
     into the run log, and transcribe every reading into the conclusion. Two readings that differ
     mean the same thing as a `quarry_commit` difference: the root is void and the matrix restarts.
  Both checks are cheap, and the conclusion states the values it checked rather than asserting the
  warning stayed silent.
- Rejected: tolerating a dirty tree and relying on the hash warning to catch trouble. The warning is
  a detector, and under resume it is not even that.

### resume-not-restart

- Decision: an incomplete or blinding-invalidated repetition is re-attempted by re-running the same
  command against the same results root — the harness resumes on `RepIsComplete` and renames a
  discarded repetition to `<n>.invalid-<k>`. Repeat until each cell reports 5 complete repetitions
  or the harness's `MaxAttempts` (3) ceiling is reached for a repetition. If a cell finishes short,
  the conclusion reports its real `n` and names the cause, exactly as the `v1-final` toc conclusion
  did for its three excluded repetitions.
- Rationale: resume is a designed feature of the results-root contract and costs nothing; a fresh
  root would throw away good repetitions and, worse, would make the cost numbers of the surviving
  root incomparable with the discarded ones. A short cell is an honest outcome — the task's
  done-when explicitly permits "or the conclusion says why not"; only a matrix that never ran fails.
- **`MaxAttempts` does not cover a blinding failure, so the run supplies its own ceiling.** The
  harness's `MaxAttempts = 3` governs only the `InvalidateRep` path — an infrastructure failure or a
  formatting miss, where the repetition directory is renamed `<n>.invalid-<k>` and the counter is the
  number of those renames. A fatal gate-2 finding takes a different path: `writeCompleteState(...,
  blindingFailed=true)` writes the repetition as `complete` with the flag set, so it is never
  invalidated and no counter ever increments. `RepIsComplete` returns false for it, which means the
  next invocation re-attempts it — and would do so without limit. `run.go`'s own comment says such a
  repetition is re-attempted "once the operator fixes the cause", a step no unattended loop performs.
  **The rule for this run: a blinding-failed repetition is re-attempted exactly once, and only after
  the cause has been diagnosed and named. A second failure of the same repetition stops the matrix;
  the conclusion records the finding verbatim and reports the cell at its real `n`.** Never re-run a
  blinding failure blind.
- What "fixing the cause" means, per fatal check — the plan needs this because three of the four are
  not transient:
  - **Check (a), the MCP prefix in a control transcript.** A control cell was granted a server it
    should not have had, or the prompt leaked the prefix. Non-transient: inspect the cell's config
    and the rendered prompt; stop until it is explained.
  - **Check (b), the quarry repository root path in a control transcript.** Almost always the
    worktree root resolving somewhere it should not; re-check `LADDER_WORKTREE_ROOT` against
    `ResolveWorktreeRoot`'s two invariants. Non-transient.
  - **Check (d), `CheckRenderedControlPrompt`.** Deterministic and pre-dispatch — see the gate
    inventory in Technical context. Re-running can never clear it, so it gets **zero** re-attempts:
    stop immediately.
  - **The memory-path scan, `ScanMemoryPaths`.** Aborts the whole invocation, not just the
    repetition. Fixing it means removing the offending file from the measured session's auto-memory
    directory (or accepting that this host cannot run the matrix cleanly) — and re-running only
    after that. One re-attempt after the fix, then stop.
- Rejected: starting a fresh root on any failure. Rejected: pushing past `MaxAttempts` by hand,
  which turns a systematic fault into an expensive loop. Rejected: relying on `MaxAttempts` to bound
  blinding re-attempts — it does not reach that path at all.

### separation-verdict-from-the-harness

- Decision: the conclusion's headline claim — does `a2-toc-dir` separate from `a0-none` on turns and
  cache-read — is taken from the harness's own `summary.json` `comparisons[]` entries and their
  `separated` field (ranges fully disjoint), quoted alongside `median [min–max]` for each metric, in
  a table shaped like the `v1-final` toc conclusion's. Every number in that table comes from this
  results root.
- **Mind the two spellings.** In `summary.json` the metric keys are `costMetricNames`' own — `turns`
  and **`cache_read_input_tokens`** (likewise `cache_creation_input_tokens`, `quarry_tool_uses`,
  `grep_fallback_total`). `cache_read` and `cache_creation` are only `table.txt`'s column headers
  (`tableColumnNames`). Read `comparisons[].metric == "cache_read_input_tokens"` out of the JSON;
  read `cache_read` off the rendered table. The conclusion may use the short names in prose, as the
  `v1-final` one does, provided the artifact it cites is named.
- Rationale: `Summarize` already recomputes each cost metric per cell over the present-and-not-void
  repetitions and builds one comparison per (rung cell, metric) pair with a disjointness verdict;
  `RenderTable` prints those lines. Re-deriving the verdict by hand in prose is how a conclusion
  starts disagreeing with the artifacts committed beside it. Note that `separated` is a strict
  no-overlap test on the min–max ranges: at n=5 a real effect can be present without it firing, so
  the conclusion reports the medians and the ranges and treats `separated` as evidence, not as the
  sole criterion.
- Rejected: inventing a significance test. n=5 on two cells does not support one, and the prior
  record makes no such claim either.

### prior-numbers-are-cited-not-merged

- Decision: the `v1-final` figures appear in the conclusion in one clearly-labelled section titled
  as the prior record, naming the branch, root and reps they come from, and never in this root's
  metric table.
- Rationale: the standing rule (HANDOFF §3, task constraints) is that cost numbers compare only
  within one results root — different host, different harness, different CLI version, different
  cache behaviour. The comparison the conclusion is entitled to make is qualitative: does the same
  direction and rough magnitude of effect appear again? Correctness metrics (`recall`, `precision`)
  are the exception the `v1-final` conclusion itself names — they may be compared by id across
  roots, and the conclusion may say so where it does that.
- Rejected: a single merged table with a "v1" column, which reads as commensurable no matter how
  many footnotes it carries.

### committed-artifacts

- Decision: the results root commits exactly `conclusion.md`, `summary.json`, `provenance.json`,
  `table.txt` and `probe.md`. Everything under `raw/` stays untracked.
- Rationale: the first three are what every `v1-final` root committed; `table.txt` is the new
  harness's rendered table and is the artifact a reader looks at first; `probe.md` is the §9a
  evidence this task is required to produce. All five are small, machine-path-free and describe the
  run completely.
- Rejected: committing only `conclusion.md` (the conclusion's numbers would then be unverifiable
  against anything).

### harness-fixes-restart-the-matrix

- Decision: if a harness defect blocks the run, it is fixed in `bench/loomyard-eval/ladder/` with a
  test, committed, and the matrix restarts in a **fresh** results root. Repetitions measured by the
  pre-fix harness are never mixed with post-fix ones in the same root. The conclusion names the fix
  and the abandoned root. If the defect is in the code under test instead
  (`internal/`, `quarry/`, `cmd/`), the run stops and the conclusion records it; fixing it is a
  separate task.
- Rationale: the harness is not the code under test, so fixing it does not break the "never edit the
  code under test mid-matrix" rule — but it does change how a repetition is measured, which makes
  pre- and post-fix repetitions incomparable in a way `provenance.json`'s per-rep server hash does
  not capture (that hash covers the server binary, not the harness). A fresh root is the only honest
  boundary. The abandoned root gets an `ABANDONED.md`, following the precedent of
  `v1-final:.../2026-09-02-compact/`.
- Rejected: patching and resuming (silently mixes two measurement regimes). Rejected: forbidding
  harness fixes outright (would block the task on its own tooling).

### docs-updated-in-this-task

- Decision: the same task updates `docs/rewrite-plan.md` §11 (the `raw/` bullet) and `HANDOFF.md`
  (§3's table row for the toc finding, §4's "next" text, §5's open-decision list) to reflect what
  T7 measured and decided.
- Rationale: HANDOFF §3 currently says "nothing on `main` reproduces them (the T7 rerun will)", and
  §5 lists the `raw/` question as open. Leaving both stale after the run is the single most
  misleading state the repository could be left in, and the edits are three short paragraphs.
- Rejected: a follow-up task. There is nothing to defer — the facts land with the conclusion.

### done-gate-runs-offline

- Decision: the done gate stays `go test ./... && golangci-lint run` with `LADDER_LIVE_TEST` unset,
  so the guarded live test skips. The live test runs once, explicitly, as part of the probe batch.
- Rationale: `live_test.go` skips unless `LADDER_LIVE_TEST=1` precisely so the repeated gate stays
  free, deterministic and network-free; making the gate spend API budget on every invocation is the
  thing that guard exists to prevent.
- Rejected: setting `LADDER_LIVE_TEST=1` in the gate.

## Technical context

**The harness.** `bench/loomyard-eval/ladder/` is T2's Go program: `cmd/ladder` (two subcommands,
`run` and `report`) over `internal/ladder`. `run` drives a sequential, cell-minor loop and then
summarises, writes and prints the table; `report` re-derives the summary and table from an existing
raw tree without running or scoring anything — useful for re-rendering after a partial run. Both
resolve the quarry repository root from the process's own working directory, so the command is run
from the worktree root.

Files worth reading before planning:

- `bench/loomyard-eval/ladder/cmd/ladder/main.go` — the flags: `--config`, `--results`, `--cells`,
  `--reps`, `--claude-bin`. `run` exits non-zero when the summary reports an incomplete or invalid
  cell, which is how the planner should detect "the matrix is not finished yet".
- `internal/ladder/run.go` — the per-repetition loop and the five-outcome failure taxonomy
  (infrastructure failure, formatting miss, max-turns completion, fatal gate-2 finding, scorer
  failure). A `max_turns` completion is *complete*, not a failure.
- `internal/ladder/runstate.go` — the six per-repetition files (`transcript.jsonl`, `answer.json`,
  `answer.redacted.json`, `usage.json`, `score.json`, `run.json`), `RepDir` =
  `<root>/raw/<cell>/<rep>`, `RepIsComplete` (state `complete` **and** not blinding-failed),
  `InvalidateRep`'s `.invalid-<n>` rename, and `MaxAttempts = 3`.
- `internal/ladder/gates.go` — the gates. Gate 1 (`CheckGrantedToolUsed`) is per cell and never
  fatal: it fires when a cell was granted tools and no repetition used one, which means the cell
  measured the tool's prompt cost only. Gate 2 (`CheckBlinding`) is per repetition, post-dispatch,
  and only for control cells: check (a) the MCP prefix appears in the transcript (fatal), check (b)
  the quarry repository root path appears (fatal), check (c) the bare token `quarry` appears
  (observation, never fatal — expect it to fire and treat it as information). **Check (d),
  `CheckRenderedControlPrompt`, is separate and easy to miss:** it runs *pre*-dispatch on the
  rendered control prompt and is fatal on the bare token `quarry`, the server name, any
  `quarry_tools` entry (here `toc`), or the MCP prefix. It is deterministic — the same prompt fails
  it every time — so a re-run can never clear it; see `resume-not-restart` for its zero-re-attempt
  rule.
- `internal/ladder/provenance.go`'s `ScanMemoryPaths` is the fifth fatal mechanism and the only one
  that stops the **whole invocation** rather than one repetition. On the first repetition whose
  session-init record names auto-memory directories, the harness writes them to
  `raw/memory-paths.json`, hashes them into `memory_path_hashes`, and walks them: a named directory
  that does not exist, or any file under one whose content matches the bare token `quarry`, is a
  fatal finding, and `run.go` turns it into `abortRun` — the repetition is written blinding-failed
  and the invocation returns. Note *whose* memory is scanned: the paths come from the measured
  session's own init record, and that session runs with its working directory in the pinned Loomyard
  worktree under the resolved worktree root, so it is Loomyard's project memory that is walked, not
  this quarry worktree's. That is a narrower blast radius than it first looks, but it is still a
  live abort path on any host whose Loomyard memory mentions quarry, and the plan must handle it.
- `internal/ladder/worktree.go` — the `Runner` seam every external process goes through;
  `ResolveLoomyardRepo` reads `LADDER_LOOMYARD_REPO` from the environment and falls back to parsing
  `KEY=VALUE` lines out of `<quarry-root>/.scratch/ladder.env`; `ResolveWorktreeRoot` resolves
  `LADDER_WORKTREE_ROOT`, else `$XDG_CACHE_HOME/ladder-eval`, else `~/.cache/ladder-eval`, and
  refuses any path that is the quarry root, under it, or merely contains the substring `quarry`
  (because gate 2 check (b)/(c) scan the cell's working directory out of the transcript);
  `PrepareWorktree` / `RestoreWorktree` create and reset the detached pinned worktree; and the
  advisory single-run lock `.ladder.lock` lives directly under the worktree root — a stale lock
  after a killed run must be cleared by hand, and the error message names its holder's pid and
  results root.
- `internal/ladder/provenance.go` — `provenance.json`: `quarry_commit`, `quarry_dirty`,
  `quarry_dirty_files`, `loomyard_commit`, `loomyard_repo_sha256` (a hash, never the path),
  `hostname`, `go_version`, `claude_version`, `server_hashes` keyed by cell-and-rep,
  `selected_cells`, `reps_effective`, `memory_path_hashes`, `session_fingerprints`, and an
  `invocations` list that merges across resumed invocations.
- `internal/ladder/summarize.go` / `report.go` — `summary.json` (per-cell `metrics`, `comparisons`
  with `separated`, `incomplete`, `invalid`) and `table.txt`, whose columns are `cell, ladder, n,
  turns, duration_ms, cost_usd, cache_read, cache_creation, output_tokens, input_tokens_total,
  tool_uses, prefixed_tool_uses, grep_fallback, tool_result_bytes, read_bytes, recall, recall_n,
  precision, precision_n, flags`. The table's header states the cache caveat in the harness's own
  words: cache-read and cache-creation are reported separately, never summed, and the median over
  repetitions is the honest statistic.
- `internal/ladder/live_test.go` — the guarded live smoke test (`LADDER_LIVE_TEST=1`,
  `LADDER_LIVE_CLAUDE_BIN` for a non-default binary). It runs one repetition of the control cell in
  a fresh worktree and asserts the advertised tool list is exactly the four built-ins, that no MCP
  server loaded, and that a granted tool actually executes.

**The configuration.** `bench/loomyard-eval/ladder/ladder-toc.yaml` — `run_model: claude-sonnet-5`,
`reps: 5`, `run_effort: medium`, `max_turns: 60`, scorer `claude-opus-5` / effort `high`,
`quarry_tools: [toc]`, `server: {name: quarry, build: ./cmd/quarry-mcp}`,
`source_repo: env:LADDER_LOOMYARD_REPO` (the loader rejects any other literal). Its two tasks pin
Loomyard at `975578cda8d6f3a81580bd4e73725e060211b766` — that is the *task worktree* pin the harness
checks out per cell, and it is a different thing from the `72c23d9` pin the task constraints name,
which is the commit the operator's Loomyard **checkout** must be sitting at (the same pin T5a's
golden test hard-fails on). Both are satisfied on this host today.

**The environment on this host, verified during exploration:**

- `.scratch/ladder.env` exists and reads `LADDER_LOOMYARD_REPO=/home/knatte/Code/loomyard/wts/loomyard`.
- That checkout's HEAD is `72c23d9ee` — the required pin — and the task pin `975578cd` is a
  reachable commit in it.
- `LADDER_WORKTREE_ROOT` is unset, so worktrees land under `~/.cache/ladder-eval`, which satisfies
  both of `ResolveWorktreeRoot`'s invariants.
- `claude --version` is `2.1.236`; plan §9a's probe table was recorded on `2.1.259`. The version is
  captured per root in `provenance.json` and per repetition in `session_fingerprints`, so the
  difference is recorded rather than hidden — but the probe batch is also where a behavioural
  difference in the flags §9a depends on would surface, which is another reason to run it first.
- `go version go1.26.0 linux/amd64`; the working tree is clean at the branch tip; `cmd/quarry-mcp`
  and `.mcp.json` are present from T6.

**Gotchas.**

- The quarry repository root on this host is `/home/knatte/Code/quarry/wts/ladder-toc-rerun`, which
  contains the substring `quarry`. That is fine for the *repository* — gate 2 check (b) scans for
  that path appearing in a control cell's transcript, and the cell runs with its working directory
  in the Loomyard worktree — but it is exactly why `ResolveWorktreeRoot` must not be pointed at
  anything under `Code/quarry/`. Do not set `LADDER_WORKTREE_ROOT` to a path containing `quarry`.
- Gate 2 check (c) counts bare occurrences of the token `quarry` in a control transcript as a
  non-fatal observation. Expect it to appear in the run's findings and treat it as information for
  the conclusion, not as a failure.
- Any hand-run `claude -p` probe against this server must pass `--setting-sources ""`, which is what
  `internal/ladder/run.go` already passes for a measured repetition. Without it the operator's global
  `defaultMode: "auto"` auto-approves read-only MCP calls, so the allowlist-denial half of the probe
  silently does not test what it appears to. This is recorded because it is the kind of difference
  that makes a hand probe disagree with the harness; the harness itself is already correct.
- `modelUsage` in the final `result` record includes Claude Code's own Haiku overhead; metrics come
  from the assistant records. The harness already does this — do not re-derive numbers from
  `modelUsage`.
- `output_tokens` was unusable on the V1 host. The new harness records it as a column; whether it is
  usable here is something the run will show, and the conclusion should say which columns it trusts.

## Constraints

There is no `CONSTRAINTS.md` at the hub root. The constraints below come from the task entry, the
plan and `HANDOFF.md`:

- **Never edit the code under test mid-matrix.** The per-rep binary hash in `provenance.json` must
  hold across the whole run; `WarnOnServerHashDrift` flags a root where it does not.
- **Cost numbers compare only within one results root.** `v1-final`'s figures are cited as the prior
  record, never pasted into this run's tables as commensurable.
- **No tracked file carries a machine path.** The Loomyard checkout comes from `.scratch/ladder.env`
  (gitignored, per machine); `provenance.json` stores hashes where a path would otherwise appear.
- **Plan §11's `results/**/raw/` decision is made by this task's first results root** and recorded in
  the conclusion.
- **The operator's §9a live probe runs before the matrix; if it has not been reported done, stop and
  ask.** It has been reported done — 2026-09-04, against the merged `cmd/quarry-mcp` built from
  `main`, both halves green (see the `probe-is-reported-done` decision) — so the matrix proceeds and
  the report is recorded as `probe.md`.
- **The done gate is `go test ./... && golangci-lint run`**, green from the repository root, with no
  pre-existing debt to inherit (`HANDOFF.md` §1).
- Go only; no Python (`CLAUDE.md`).
- The task branch is `ladder-toc-rerun`, parent `main`; task-state files live under `_mill/` and are
  committed on the branch.

## Testing

Most of this task is a run, not new code, so the test strategy is mostly about what gates the run
rather than what unit tests get written.

- **The pre-matrix smoke test is the first gate.**
  `env -u CLAUDECODE -u CLAUDE_CODE_ENTRYPOINT LADDER_LIVE_TEST=1 go test
  ./bench/loomyard-eval/ladder/internal/ladder -run TestLive -v` — the same `env -u` prefix the
  matrix uses — must pass before the matrix starts; a failure blocks it. This is the harness's own guarded live test —
  the `claude -p` seam, the fresh-worktree tool grant, the no-MCP-server assertion for a control
  cell — and it is the worker's tooling check, not the operator's §9a probe. The §9a probe is
  reported done (see `probe-is-reported-done`) and is written up in `probe.md`, not re-run.
- **The offline suite is the standing gate.** `go test ./...` and `golangci-lint run` from the
  repository root, with `LADDER_LIVE_TEST` unset, before the matrix starts and again before the task
  is marked done. The tree is green today, so any failure is something this task introduced.
- **The matrix's own gates are the run's tests.** Gate 1 tells the conclusion whether `a2-toc-dir`
  used the tool it was granted — a cell that never called `toc` measured prompt cost only and the
  conclusion must say so. Gate 2 tells it whether any control repetition was contaminated. The
  `incomplete` and `invalid` lists in `summary.json`, and `run`'s non-zero exit, are the machine
  check that the matrix actually finished.
- **A hand-verification pass on one repetition.** T2's own done-when was "the metrics match the
  transcript by hand" for a single `reps: 1` control run. Repeat that spot check once here against
  one `a2-toc-dir` repetition — read `raw/a2-toc-dir/1/transcript.jsonl` and confirm the turn count,
  the `toc` call count and the `cache_read` figure in `usage.json` match what the transcript shows.
  It is the only defence against a summariser that is confidently wrong, and it costs one file read.
- **TDD candidates:** none by default, because no new production code is planned. If a harness
  defect surfaces and must be fixed (see `harness-fixes-restart-the-matrix`), that fix is a genuine
  TDD candidate: a failing table test in the relevant `internal/ladder/*_test.go` file first, then
  the fix, then a fresh results root. `internal/ladder` is thoroughly table-tested with an offline
  fake-runner layer (`testdata/fakeclaude`, `testdata/stubmcp`), so a new case has an obvious home
  and needs no new scaffolding.
- **Scenarios the conclusion must cover regardless of outcome:** a cell short of 5 repetitions and
  why; any repetition invalidated by gate 2 (checks (a), (b)) and why; any check (d)
  `CheckRenderedControlPrompt` failure, which is deterministic and stops the run outright; any
  `ScanMemoryPaths` abort, including which file tripped it and what was done about it; any gate-1
  finding; the `quarry_commit` equality check across `invocations[]` and the out-of-band server-hash
  readings, stated as values rather than as "no warning fired"; any session-fingerprint drift
  observation; and whether `output_tokens` is usable on this host.
- **Negative-coverage note.** Two of the checks above are ones a passing run reports nothing about:
  `WarnOnServerHashDrift` cannot fire after a resume (see `clean-tree-before-the-matrix`), and check
  (c)'s `target_origin_quarry_mention` is expected to fire and means nothing on its own. Silence from
  either is not evidence, and the conclusion must not present it as such.

## Q&A log

- **Q:** What is the results root called? **A:** [auto-pick] `bench/loomyard-eval/ladder/results/2026-09-04-toc`. **Why:** the plan and task both spell `results/<date>-toc`, and every `v1-final` root uses a `YYYY-MM-DD` prefix; the date is the run's, so a later first run renames it.
- **Q:** Which cells and how many repetitions? **A:** [auto-pick] `--cells a0-none,a2-toc-dir` with no `--reps` override, so the file's `reps: 5` applies. **Why:** that is verbatim plan §12's T7 row and `ladder-toc.yaml`'s closing comment; ladder b answers a different question and would double the cost of the gate.
- **Q:** Is `results/**/raw/` committed or ignored? **A:** [auto-pick] Ignored, confirming the `results/*/raw/` entry T2 already shipped, recorded in the conclusion and in plan §11. **Why:** `raw/memory-paths.json` carries machine paths that no tracked file may hold — the reason `provenance.json` stores hashes instead — and the raw tree is ten 60-turn transcripts fully summarised by the committed artifacts.
- **Q:** The §9a live probe is a precondition — has it been reported done? **A:** Yes — answered by the orchestrator at discussion-review round 1: the operator ran it on 2026-09-04 against the merged `cmd/quarry-mcp` built from `main`, harness-faithfully, and both halves were green. The task does not re-run it; `probe.md` records the operator's report, and the harness's guarded `TestLive` still runs as the worker's own pre-matrix smoke test. **Why:** the constraint asks a question whose answer the orchestrator holds. The draft answered it from the absence of an artifact in the tree (T6's `_mill/` was cleaned at merge) and self-authorised past a stop-and-ask constraint on that inference, which was wrong twice over — the premise was false and the constraint was not the worker's to rewrite.
- **Q:** Who executes the matrix, and how? **A:** [auto-pick] A mill-go batch in this worktree, backgrounded with a tee'd log under `.scratch/` and polled. **Why:** ten runs plus ten scorer calls is 20–40 minutes, well past any foreground call's ceiling, and a killed invocation would strand the advisory run lock.
- **Q:** Must the tree be clean when the matrix starts? **A:** [auto-pick] Yes — commit everything first and abort if `git status --porcelain` is non-empty. **Why:** `provenance.json` records `quarry_dirty` and the per-rep server hash; a dirty tree makes the committed record describe something that is not in git, which is the exact fault the 2026-09-02 post-mortem fixed.
- **Q:** What happens to incomplete or blinding-invalidated repetitions? **A:** [auto-pick] Resume into the same root until each cell has 5 complete repetitions or `MaxAttempts` is exhausted, then report the real `n` and the cause. **Why:** resume is a designed property of the results-root contract, a fresh root would discard good repetitions, and the done-when explicitly allows an honest shortfall.
- **Q:** How is "separates from the control" decided? **A:** [auto-pick] From the harness's own `comparisons[].separated` on `turns` and `cache_read`, quoted with `median [min–max]`. **Why:** `Summarize` already computes the disjointness verdict that `table.txt` prints; a hand-derived verdict would eventually contradict the artifacts committed beside it. `separated` is a strict no-overlap test, so medians and ranges are reported too and it is treated as evidence, not as the sole criterion.
- **Q:** How are the `v1-final` numbers presented? **A:** [auto-pick] In one clearly-labelled prior-record section naming branch, root and reps — never in this root's metric table. **Why:** the standing rule is that cost numbers compare only within one root; correctness metrics are the stated exception and may be compared by id.
- **Q:** Should the measured `claude -p` children inherit this session's environment? **A:** [auto-pick] No — invoke the harness under `env -u CLAUDECODE -u CLAUDE_CODE_ENTRYPOINT`. **Why:** those variables announce "running inside a Claude Code session", a condition the V1 measurement did not run under and not one to introduce silently into a regression gate.
- **Q:** What if a harness bug blocks the run? **A:** [auto-pick] Fix it with a test, commit, and restart the matrix in a fresh results root, naming the fix and the abandoned root in the conclusion. **Why:** the harness is not the code under test, so fixing it breaks no rule — but it changes how a repetition is measured, and the per-rep server hash does not cover the harness, so a fresh root is the only honest boundary. A defect in the code under test stops the run instead and becomes a separate task.
- **Q:** Does this task update the surrounding documents? **A:** [auto-pick] Yes — `docs/rewrite-plan.md` §11 and `HANDOFF.md` §3/§4/§5. **Why:** HANDOFF currently states that nothing on `main` reproduces the number and lists the `raw/` question as open; leaving both stale after the run is the most misleading state the repository could be left in.
- **Q:** Which files in the results root are committed? **A:** [auto-pick] `conclusion.md`, `summary.json`, `provenance.json`, `table.txt`, `probe.md`. **Why:** the first three match every `v1-final` root, `table.txt` is what a reader looks at first, `probe.md` is the required §9a evidence, and all five are small and machine-path-free.
- **Q:** Does the done gate run the live test? **A:** [auto-pick] No — `go test ./... && golangci-lint run` with `LADDER_LIVE_TEST` unset. **Why:** the guard exists so the repeated gate stays free and deterministic; the live test runs once, explicitly, in the probe batch.
- **Q:** The server-hash drift warning is defeated by the resume workflow this task mandates — what checks drift instead? **A:** [auto-pick] `quarry_commit` equality across `provenance.json`'s `invocations[]`, plus an out-of-band `sha256sum` of the built server binary recorded per invocation and transcribed into the conclusion. **Why:** `run.go` rewrites `server_hashes` for every rep on every invocation and `CollectInvocation` leaves the per-invocation copy nil, so after any resume the map holds one hash and `WarnOnServerHashDrift` cannot fire; `invocations[]` is appended to rather than rewritten, so it survives.
- **Q:** `MaxAttempts` does not bound blinding-failed repetitions — what stops the loop? **A:** [auto-pick] One re-attempt, and only after the cause is diagnosed and named; a second failure of the same repetition stops the matrix and the conclusion reports the cell at its real `n`. Check (d) gets zero re-attempts because it is deterministic. **Why:** a blinding failure is written `complete` with the flag set rather than invalidated, so no counter increments and an unattended loop would re-attempt it without limit.
