# Discussion: Ladder breadth (M1)

```yaml
task: Ladder breadth (M1)
slug: ladder-breadth
status: discussing
parent: main
```

## Problem

T7 reran the one measured win the whole rewrite leaned on — a directory table of contents halving
exploration cost on task 01 — against the merged rewrite, and it came back flat-to-reversed at n=5
(`bench/loomyard-eval/ladder/results/2026-09-04-toc/conclusion.md`). No cost metric separated, the
tool was demonstrably called twice in every repetition, and correctness did not move either
direction. What that root explicitly does **not** answer is whether a toc pays under a different
task or scope: ladder b (task 04, the negative control) was declared in `ladder-toc.yaml` but never
selected, and no task shape other than task 01 has ever been measured against the rewrite.

`docs/roadmap.md`'s standing rule is that nothing is built without a measured win behind it, so T8
(the type checker: `impact`, `assert-no-callers`, `verified`, the §8.2 DAG tightening) is parked
until measurement says where — or whether — the surface pays. This task is that measurement. Folded
in ahead of it is M2, a harness observability gap the same T7 root exposed: three attempts across
two repetitions of `a0-none` were silently retried and succeeded, and nothing on disk says why the
first attempts were rejected, because the harness writes a reason file only ahead of the
server-not-connected path — which by design never runs for a control cell.

**Why now:** the build queue is stopped behind this answer. M2 goes first because it is a harness
change, and a harness change cannot land mid-matrix.

## Scope

**In:**

- **M2:** persist the invalidation cause into a discarded attempt's directory before
  `InvalidateRep` renames it away, covering *every* path that invalidates an attempt, not only the
  server-not-connected one. Proven by test.
- **Ladder b:** run `ladder-toc.yaml` task 04's cells `b0-none` and `b8-toc-dir` — already declared,
  never run — against `main`'s `cmd/quarry-mcp` at reps 5.
- **Ladder c (new shape, multi-package navigation):** finish the drafted but unrunnable
  `bench/loomyard-eval/tasks/02-shedadapters-exploration.md` (add its missing `## Output schema`
  heading), author its fasit, and add cells `c0-none` / `c1-toc-dir`.
- **Ladder d (new shape, cold-start orientation):** author a new task file, prompt and fasit for a
  whole-repository question that names no package, and add cells `d0-none` / `d1-toc-dir`.
- **The matrix:** one invocation, six cells × 5 reps, into a single results root
  `bench/loomyard-eval/ladder/results/<YYYY-MM-DD>-breadth/`.
- **The conclusion:** `results/<YYYY-MM-DD>-breadth/conclusion.md`, naming per task shape whether any
  cost or correctness metric separates, or stating plainly that none does anywhere.
- **Doc propagation:** update `docs/roadmap.md`'s parked-T8 section and `HANDOFF.md` §3's measured
  record from the conclusion's own finding.

**Out:**

- **The OSL-1033 host rerun.** Isolating the host variable behind T7's task-01 discrepancy is
  operator-coordinated. Do not block on it, do not schedule it, do not mention it as a prerequisite.
- **Re-running ladder a.** Task 01 was measured by T7 on this host and harness; re-measuring it here
  buys nothing and costs ten real `claude -p` runs.
- **Widening the matrix.** No third rung per ladder, no extra task, no reps above 5, without the
  operator's explicit word.
- **Unparking T8, or writing any part of it.** This task produces the input to that decision and
  stops there.
- **Any change to `cmd/quarry-mcp`, `quarry/`, `internal/engine` or the CLI.** The code under test is
  `main`'s, unmodified; the only source this task edits is the ladder harness and the ladder's own
  task/config files.
- **A grep-toc control cell, annex delivery, the compact output form, and the file-level toc tool.**
  All four are named as deliberately absent in `ladder-toc.yaml`'s own header and each needs harness
  or engine work this task does not do.
- **Committing `results/**/raw/`.** Settled by T7; the `.gitignore` entry stays.

## Decisions

### m2-one-reason-file-for-every-cause

- Decision: replace `ServerConnectFailureFile` (`server_connect_failure.txt`) with a single
  `InvalidReasonFile = "invalid_reason.txt"`, written into the attempt directory immediately before
  `InvalidateRep` renames it, on **every** path that invalidates an attempt. The file is plain text,
  one `key: value` per line, carrying at least: `cell`, `repetition`, `attempt`, `cause`, and a
  free-form `detail`; plus `exit_code` when one is recoverable. `cause` is a fixed enumeration:
  `runner_error`, `result_error`, `unparseable_answer`, `server_not_connected`.
- Rationale: the T7 conclusion's own ask is "a future harness change that also persists the
  runner-level error (or the process exit code) into the discarded attempt directory". One file with
  a `cause` field answers that for all four paths and keeps a reader of a `.invalid-N/` directory
  looking in one place. `server_connect_failure.txt` becomes the `cause: server_not_connected` case
  of the same file, so nothing is lost and no second file has to be explained.
- Rejected: keeping `server_connect_failure.txt` and adding a second file for the other causes — two
  files with overlapping purpose, and a reader who checks only the one they know about. Also
  rejected: recording the cause into `run.json` instead — the invalidating paths never reach
  `writeCompleteState`, so there is no `run.json` to write into; the attempt directory is the only
  surviving place.

### m2-cause-taxonomy-and-exit-code

- Decision: derive `cause` from where the attempt loop in `run.go` actually rejects the attempt:
  `invokeErr != nil` → `runner_error`; a transcript whose result record carries `IsError` (or has no
  result record at all) → `result_error`; a missing or undecodable fenced answer → `unparseable_answer`;
  a non-nil `CheckServerConnected` finding → `server_not_connected`. Recover `exit_code` by
  `errors.As` down to `*exec.ExitError` and calling `ExitCode()`; when no exit status is recoverable,
  omit the `exit_code` line rather than writing a sentinel.
- Rationale: these are exactly the four branches the existing loop already distinguishes — the change
  records a decision the code already makes rather than inventing a new classification. `ExecRunner`
  wraps its failure with `%w`, and `invokeMeasuredProcess` wraps again with `%w`, so the exit error
  survives to the call site. Omitting an unknown field is honest; `exit_code: -1` reads like a real
  exit status.
- Rejected: a single free-text reason line with no `cause` key — greppable classification across a
  results root is the whole point. Also rejected: parsing the exit code out of the wrapped error's
  text.

### m2-proven-by-test

- Decision: prove M2 with e2e tests in `bench/loomyard-eval/ladder/internal/ladder/e2e_test.go`
  driven by the existing `testdata/fakeclaude` fixture: a new case using `FAKE_CLAUDE_STREAM=partial_fail`
  (which flushes a partial stream and `os.Exit(1)`) asserting that each `N.invalid-k/` carries
  `invalid_reason.txt` naming `cause: runner_error`, the cell, the repetition and `exit_code: 1`; a
  case using `no_fence` asserting `cause: unparseable_answer`; and the existing
  `GrantedCellServerNeverConnects` case retargeted from `ServerConnectFailureFile` to
  `InvalidReasonFile` with `cause: server_not_connected`.
- Rationale: the roadmap's done-when for M2 is literally "an artificially failed attempt's
  `N.invalid-1/` carries a readable reason file"; `partial_fail` is that artificial failure and it
  already exists in the fixture. The offline runner seam means these tests cost nothing and need no
  network.
- Rejected: a unit test on the writer function alone — it would not prove the writer is called on the
  paths that actually invalidate, which is the entire gap T7 hit. Also rejected: demonstrating the
  file by hand during the matrix — the matrix must not be perturbed to produce evidence.

### m2-lands-before-the-matrix

- Decision: M2 is complete, tested, linted and **committed** before the matrix invocation starts.
  The matrix then runs against the post-M2 harness, and the breadth matrix's own non-control cells
  (`b8-toc-dir`, `c1-toc-dir`, `d1-toc-dir`) completing end to end are what satisfies the third
  harness rule for this harness change.
- Rationale: `docs/rewrite-plan.md` §2 forbids editing the code under test mid-matrix and holds a
  harness change proven only by a non-control cell completing end to end. Sequencing M2 first
  satisfies both with no extra runs. A clean tree at invocation time also keeps `provenance.json`'s
  `quarry_dirty` honest (the only tolerated dirty entry is the narrow
  `_mill/briefs/<currently-executing-batch>*.md` carve-out T7 established).
- Rejected: running the matrix first and landing M2 after — it would leave the very run that most
  needs invalidation diagnostics running on the blind harness. Also rejected: splitting M2 out to a
  separate `mill-quick` task — the roadmap allows it, but then this task's matrix either waits on
  another task's merge or runs on the old harness.

### two-new-shapes-not-one

- Decision: author **two** new task shapes, ladders c and d.
- Rationale: T7 already demonstrated that one flat result on one shape settles nothing. The whole
  point of this task is breadth; a single new shape that comes back flat leaves the reader unable to
  distinguish "toc does not pay" from "we picked the wrong second shape". Two shapes plus ladder b
  gives three independent task shapes in one root.
- Rejected: one shape (cheaper, but a weaker answer to the exact question that parked T8); three or
  more (each shape costs a fasit plus ten real `claude -p` runs, and the task body caps this at two).

### ladder-c-reuses-task-02

- Decision: ladder c's task is the existing, drafted `bench/loomyard-eval/tasks/02-shedadapters-exploration.md`
  — the three-package shed-pipeline exploration — finished rather than replaced. Finishing it means
  adding the `## Output schema (exploration tasks)` section it lacks (the same fenced block task 01
  carries) and authoring `02-shedadapters-exploration.fasit.json`.
- Rationale: the shape wanted for "multi-package navigation" already exists in the repository, was
  drafted against the same pinned SHA, and its own header names the reason it is interesting
  (`shedadapters` alone is ~8.4k lines, "a directory too big to comfortably read file-by-file"). Its
  only blockers are the missing schema heading — without which `LoadTaskFile` hard-fails — and the
  missing fasit. Writing a fresh three-package prompt instead would discard drafted, reviewed work.
- Rejected: a brand-new multi-package prompt (duplicated effort, no better); using task 05
  (`05-mergeresolve-resolve-impact`) instead — it is impact-shaped, and `impact`-shaped tools have
  measured flat in every run since August (`docs/rewrite-plan.md` §2 point 2).

### ladder-d-cold-start-orientation

- Decision: ladder d's task is new: `bench/loomyard-eval/tasks/06-loomyard-cold-start-orientation.md`,
  an exploration-schema task whose prompt **names no package and no file** and asks the agent to
  locate, in an unfamiliar repository, which package(s) own a named behaviour and what the entry
  points into that behaviour are. The subject is chosen by the implementer from the pinned Loomyard
  checkout under three constraints: (a) the real answer spans at least two packages, (b) none of
  those package names appears in the prompt text, (c) it is answerable entirely from the pinned SHA.
  The chosen subject and the reason it satisfies (a)–(c) is recorded in the task file's own notes
  section.
- Rationale: this is the condition under which a directory-level toc is most plausibly worth its
  prompt cost — the agent does not know where to look, so the first cheap survey is the whole value.
  Every shape measured so far (01, 02, 04) hands the agent its scope in the prompt, which is exactly
  the confound; ladder b is the negative control at the other end (the task text names the file
  outright). Ladders b, c and d together therefore span the scope axis rather than sampling it once.
- Rejected: a one-package exploration task (the other end named in `ladder-toc.yaml`'s "deliberately
  NOT here" list, but it tests less than task 01 already does); reusing task 03 (a post-hoc review of
  a compile-breaking rename, whose own postmortem explains why it differentiates nothing).

### fasit-authored-by-a-reference-agent-card

- Decision: each new fasit (`02-...fasit.json`, `06-...fasit.json`) is authored by its own dedicated
  card whose agent reads the pinned Loomyard worktree exhaustively, with no turn or token discipline
  applied, and cross-checks its answer by a second independent method (e.g. `go build` / `go vet`
  experiment, `git log`/`git show` on the relevant history, or `grep` sweeps) before writing. The
  file follows tasks 01/04's shape exactly, including a `_meta` block carrying `task`, `type`,
  `pinned_sha`, `scope`, `date`, `arm: "C"`, and a `role` line naming the method used. The fasit's
  `key_symbols` entries each carry `name`, `file`, `role`; `summary` is 3–6 sentences.
- Rationale: this reproduces V1's arm-C protocol, which is what tasks 01, 04 and 05's committed
  fasits were produced by, so the new shapes are scored on the same footing as the old ones.
  `StripFasitMeta` drops `_meta` before the scorer ever sees it, so the block is free documentation.
- Rejected: hand-authoring a thin fasit from filenames alone (an under-specified fasit deflates
  recall uniformly across control and rung, which hides rather than reveals a separation); generating
  the fasit with the toc tool alone (it would bias the reference answer toward exactly what the
  measured rung can see).

### degenerate-fasit-is-a-pre-matrix-swap

- Decision: if a new shape's fasit turns out degenerate — the answer is trivial, the packages barely
  interact, or the reference agent cannot reach a confident answer at the pin — swap the subject and
  re-author **before** the matrix invocation, and record the swap and its reason in the task file.
  Never after the matrix has started.
- Rationale: task 02's own drafting notes already anticipate this ("if C's answer turns out trivial …
  pick a different subsystem for a re-run rather than forcing a scorecard out of a degenerate case").
  Once the matrix starts, prompts and fasit are measured stimulus under the no-mid-matrix-edit rule
  exactly like source, so the swap window closes.
- Rejected: running a degenerate shape anyway and noting it in the conclusion (it burns ten real runs
  to produce an uninterpretable row).

### one-ladder-file-one-results-root

- Decision: add ladders c and d to the existing `bench/loomyard-eval/ladder/ladder-toc.yaml` rather
  than creating a second ladder file, and run all six cells in **one** invocation into **one** results
  root. Amend the file's header comment accordingly: the "two task groups per file" line and the
  "deliberately NOT here" entry for a whole-repo, no-scope-hint task are both superseded by this
  task and must be rewritten rather than left contradicting the rows below them.
- Rationale: cost numbers compare only within one results root, so putting every cell this task
  measures into one root is what makes the three shapes' results readable side by side under one
  provenance record, one binary hash set, and one host. Nothing in `config.go` limits the number of
  ladder letters — `validate` only requires exactly one control per letter that appears — so the
  two-groups-per-file line is a stale comment, not a format rule. Cell ids stay globally unique
  across the toc matrix, so a future cross-root comparison by id still works.
- Rejected: a second `ladder-breadth.yaml` re-declaring ladder b (duplicate ids across files, two
  invocations, two provenance records, and a reader who has to know which file a `b8-toc-dir` number
  came from). Also rejected: three separate results roots, one per shape (same problem, worse).

### cell-ids-and-shape

- Decision: `c0-none` / `c1-toc-dir` for ladder c, `d0-none` / `d1-toc-dir` for ladder d. Each new
  ladder letter gets exactly one control (`allowed: []`) and exactly one rung (`allowed: [toc]`).
  Ladder b's ids are unchanged (`b0-none`, `b8-toc-dir`) — they are already declared and already
  chosen to avoid the main matrix's `b1`..`b7`.
- Rationale: `<letter><n>-<rung>` is the established id shape, `0` is the control everywhere, and c
  and d are fresh letters so `1` is free. One control per letter is enforced by `validate`.
- Rejected: renaming `b8-toc-dir` to `b1-toc-dir` for tidiness — the id is deliberately chosen to
  stay unique against the main matrix's ids and cross-root by-id comparison depends on that.

### run-parameters-unchanged

- Decision: `reps: 5`, `run_model: claude-sonnet-5`, `run_effort: medium`, `max_turns: 60`, scorer
  `claude-opus-5` at `high`, `quarry_tools: [toc]`, server built from `./cmd/quarry-mcp`, new tasks
  pinned to the same `975578cda8d6f3a81580bd4e73725e060211b766` as tasks 01–05. All six cells are
  selected in one `--cells` list.
- Rationale: every one of these is either the file's existing value or the value T7 measured at; the
  task body fixes reps at 5 and forbids widening. Sharing the pin means the same Loomyard worktree
  content underlies all three shapes, so a difference between shapes is the shape, not the tree.
- Rejected: raising reps to sharpen the `separated` test (explicitly forbidden by the task's cost
  discipline); a different pin for the new tasks (a second checkout state to reason about, for
  nothing).

### separation-decision-rule

- Decision: a shape is reported as **separating** only when, within this results root, (1) at least
  one cost metric's comparison entry is `separated: true`, (2) that metric's median moves in the
  direction the toc hypothesis predicts (rung cheaper than control), and (3) neither recall nor
  precision degrades — i.e. the correctness comparison is not `separated: true` in the control's
  favour. Anything short of all three is reported as **no separation**, with the medians and ranges
  quoted so a reader can see how close it came. The conclusion states the verdict per shape and then
  states the aggregate: whether toc separates anywhere in this root, or nowhere.
- Rationale: `separated` is a strict no-overlap test on min–max ranges, which at n=5 can miss a real
  effect — so the median direction has to be read alongside it, exactly as T7's conclusion did. And a
  cost win bought with degraded correctness is not a win for a tool whose stated purpose is complete
  extraction. Naming the rule before the numbers exist is what keeps it from being fitted to them.
- Rejected: median-direction alone (too weak — T7's own medians moved, in the wrong direction, at
  `separated: false`); `separated: true` alone (would let a correctness regression through, and would
  count a wrong-direction separation as a result).

### conclusion-is-the-t8-input-and-propagates

- Decision: `results/<YYYY-MM-DD>-breadth/conclusion.md` follows T7's conclusion structure — a
  numbers table per ladder quoted verbatim from `summary.json`/`table.txt` and never re-derived, a
  per-shape verdict under the rule above, a gate/invalidation/drift/provenance coverage section, and
  a "what this settles" section that names the T8 unpark implication in the roadmap's own terms
  (either a measured win that re-establishes the surface's value, or not). It then updates
  `docs/roadmap.md`'s parked-T8 section and `HANDOFF.md` §3's table with the new row.
- Rationale: `docs/roadmap.md` names this conclusion as the input to the unpark decision, and T7's
  own finding was that the record must stop citing a superseded result — the same propagation is owed
  here. Quoting rather than re-deriving is what made T7's conclusion auditable.
- Rejected: writing the conclusion and leaving the roadmap and HANDOFF to a later task (the stale
  record is the exact failure T7 called out); making the unpark decision in this task (it is the
  operator's, and the roadmap says so).

### m2-observability-is-not-measured

- Decision: the `invalid_reason.txt` files produced during the matrix are read and reported in the
  conclusion's coverage section (how many attempts were invalidated, per cell, with their causes) but
  are not a measured metric and do not enter any comparison.
- Rationale: it closes the loop on why M2 was folded into this task — the reader of the breadth root
  gets the diagnostic T7's reader did not have — without pretending an infrastructure artifact is a
  benchmark result.
- Rejected: silently ignoring them (wastes the whole point of M2 on the first root that could use it).

## Technical context

**The harness.** `bench/loomyard-eval/ladder/`, one Go module with the repository root (`./go.mod`);
there is no nested module. `cmd/ladder` has two subcommands, `run` and `report`. The measured loop is
`internal/ladder/run.go`; the on-disk contract for one repetition is `internal/ladder/runstate.go`.

- `runstate.go` holds the six per-repetition file-name constants, `ServerConnectFailureFile`,
  `MaxAttempts = 3`, `RepDir`, `RunState`, `RepIsComplete`, and `InvalidateRep` (which renames
  `<dir>` to `<dir>.invalid-<n>` and returns `n`).
- `run.go`'s attempt loop is the M2 change site (around lines 386–460). Its shape: call
  `invokeMeasuredProcess`; accept only when `invokeErr == nil && t.Result != nil && !t.Result.IsError`
  **and** (for a non-control cell) `CheckServerConnected` returns nil **and** the answer is either a
  `max_turns` transcript or a decodable fenced JSON block. On rejection it calls
  `writeServerConnectFailure` (server-connect path only), then `InvalidateRep`, then retries until
  `attempts >= MaxAttempts`. The `connectFailures == attempts` case aborts the whole run and must keep
  working unchanged. The loop currently has no explicit attempt counter of its own — one is needed for
  the reason file's `attempt` field, and it must agree with the suffix `InvalidateRep` produces.
- `writeServerConnectFailure` (`run.go`, ~line 584) is the function the new writer replaces; its
  current content shape (`cell:`, `repetition:`, `expected_server:`, then the finding message) is the
  precedent for the new file's layout.
- `ExecRunner.Run` (`worktree.go`) wraps its `cmd.Run()` failure with `fmt.Errorf(... %w)`, and
  `invokeMeasuredProcess` wraps again with `%w`, so `errors.As(err, &exitErr)` down to
  `*exec.ExitError` reaches the real exit status. `os/exec` is already imported in `worktree.go`; the
  writer lives in `run.go`, which will need `errors` and `os/exec`.
- `summarize.go:251` carries a comment referencing `InvalidateRep`; nothing in `summarize.go` or
  `report.go` reads the reason file, and nothing needs to.

**The ladder file format.** `internal/ladder/config.go`. `LoadLadder` rejects unrecognised keys.
`validate` requires: `source_repo` exactly `env:LADDER_LOOMYARD_REPO`; non-zero `run_model`,
`run_effort`, `max_turns`, `reps`, `scorer.model`, `scorer.effort`; per task a non-empty `task_file`,
`pinned_sha`, `fasit` and a `schema` of exactly `"exploration"` or `"impact"`; unique config ids;
every `configs[].task` a key in `tasks`; every entry of `configs[].allowed` present in
`quarry_tools`; and **exactly one control (empty `allowed`) per ladder letter that appears**. There
is no cap on the number of ladder letters.

**The task-file contract.** `internal/ladder/prompt.go`. `LoadTaskFile` extracts exactly two things
and hard-fails without either: the blockquote under a `## ` heading containing the literal
`` `<TASK TEXT>` `` marker (dedented, `>` plus at most one space stripped), and the **first** fenced
JSON block after the first line starting with `## Output schema`. Extraction is inclusion-based on
purpose — it never looks for or excludes an answer-key heading, because those are spelt differently
across task files. Consequence for authoring: the schema heading must come **before** any answer-key
notes section, and the answer key may be spelt however is natural. `RenderPrompt` assembles
`PARALLEL_OPENING`, the identical body naming the target dir and granted tool names, the task text,
`PARALLEL_BLOCK`, the closing sentence, and the schema block — the same preamble for every cell.
`02-shedadapters-exploration.md` today has the `` `<TASK TEXT>` `` heading but **no** `## Output
schema` heading, which is why it cannot be loaded; task 01's `## Output schema (exploration tasks)`
section is the block to mirror.

**Scoring.** `internal/ladder/score.go`. `ExplorationRule` scores recall/precision over the fasit's
`relevant_files` and `key_symbols` plus a qualitative `summary_matches`; `ImpactRule` scores
`callers_to_update` on file *and* line plus `decoy_admitted` and `lookalikes_matched`. The rule is
selected by the task's `schema` key. `BuildScorerPrompt` embeds only the rule, the task text, the
`_meta`-stripped fasit and the redacted answer — never the cell id, ladder letter or tool list.
`RedactAnswer` strips the MCP prefix, the quarry repo root, the task worktree path, every entry of
`quarry_tools` and the bare server name. An exploration fasit therefore needs `relevant_files`,
`key_symbols` and `summary` to be genuinely good, since recall and precision are computed against
them.

**Provenance and the run.** `provenance.json` records `quarry_commit`, per-rep `server_hashes`,
`quarry_dirty` / `quarry_dirty_files`, `memory_path_hashes` and the `invocations` list. The tree must
be clean at invocation time apart from the established `_mill/briefs/<currently-executing-batch>*.md`
carve-out. `results/*/raw/` is gitignored by `bench/loomyard-eval/ladder/.gitignore` and stays that
way. `LADDER_LOOMYARD_REPO` is read from the process environment first and otherwise from
`<repo-root>/.scratch/ladder.env` (gitignored, per machine) by `ResolveLoomyardRepo`.

**The run command,** for reference:

```
go run ./bench/loomyard-eval/ladder/cmd/ladder run \
  --config bench/loomyard-eval/ladder/ladder-toc.yaml \
  --results bench/loomyard-eval/ladder/results/<YYYY-MM-DD>-breadth \
  --cells b0-none,b8-toc-dir,c0-none,c1-toc-dir,d0-none,d1-toc-dir
```

**The test fixtures.** `internal/ladder/testdata/fakeclaude` is a fake `claude` binary driven by
`FAKE_CLAUDE_*` environment variables; `FAKE_CLAUDE_STREAM` selects one of `normal`, `max_turns`,
`no_fence`, `leak_prefix`, `partial_fail`. `partial_fail` writes a partial stream then `os.Exit(1)` —
the runner-error injection M2's test needs. `FAKE_CLAUDE_SERVER_STATUS_OVERRIDE` (e.g.
`quarry=failed`) drives the server-not-connected path. `e2e_test.go` has helpers `newE2EEnv`,
`baseLadder`, `writeSyntheticLadderFile`, `setFakeClaudeEnv`, `setFakeClaudeEnvGranted`,
`runOpts`, `summarizeAndWriteReport`, and an existing `GrantedCellServerNeverConnects` case (~line
581) that already asserts a reason file in each `.invalid-N/` — that assertion is the one to retarget.

**Prior art to read before writing the conclusion:** `results/2026-09-04-toc/conclusion.md` is the
structural model (numbers quoted not re-derived, correctness reported with its own n, a coverage
section, a prior-record section that never merges numbers across roots).

## Constraints

There is no `CONSTRAINTS.md` at the hub root. The binding constraints come from the task body and
`docs/rewrite-plan.md` §2:

- **The three harness rules.** Never edit the code under test mid-matrix (a binary hash per rep is
  recorded in `provenance.json`); cost numbers compare only within one results root; a harness change
  is proven only by a non-control cell completing end to end.
- **New task prompts and fasit are measured stimulus.** Once the matrix has started they are under
  the no-mid-matrix-edit rule exactly like source.
- **`results/**/raw/` stays untracked** (settled by T7), and **no tracked file carries a machine
  path** — this is why `provenance.json` stores `memory_path_hashes` rather than paths, and it
  applies to every file this task writes, the conclusion included.
- **`LADDER_LOOMYARD_REPO` comes from `.scratch/ladder.env`**, per machine, gitignored. Never commit
  it, never hardcode the checkout path.
- **Cost discipline.** Real `claude -p` matrices. Reps stay at 5; six cells, no more, without the
  operator's word. Ladder a is not re-run.
- **Go only** (`CLAUDE.md`): no Python anywhere in this task.
- **The done gate** is `go test ./... && golangci-lint run`, green before the matrix invocation and
  green at handoff.
- **Never use `sed`** in any shell command (global operator rule); use the editing tools, or
  `awk`/`grep`/`cat`.

## Testing

**M2 — the invalidation reason file.** This is the TDD candidate; write the assertions first.

- *Unit, `runstate_test.go`:* `InvalidateRep`'s existing three-attempt suffix test is unchanged and
  must stay green — the rename semantics are not being touched. If the writer is factored as a pure
  function of (cell, rep, attempt, cause, exit code, detail), unit-test its rendered text for a
  present and an absent exit code.
- *e2e, `e2e_test.go` — `runner_error`:* a single-cell matrix under `FAKE_CLAUDE_STREAM=partial_fail`.
  Assert the repetition directory is gone, that each of the three `N.invalid-k/` directories carries
  `invalid_reason.txt`, and that the file names the cell id, the repetition, `cause: runner_error`
  and `exit_code: 1`. Assert the attempt number in each file matches its own directory's suffix.
- *e2e — `unparseable_answer`:* the same shape under `no_fence`, asserting `cause: unparseable_answer`
  and no `exit_code` line (the process exited zero).
- *e2e — `server_not_connected`:* retarget the existing `GrantedCellServerNeverConnects` case from
  `ServerConnectFailureFile` to `InvalidReasonFile`, keeping its existing assertions that the file
  names the cell and the server, and adding `cause: server_not_connected`.
- *Regression:* the happy path must write **no** `invalid_reason.txt` into a completed repetition's
  own directory — assert its absence, since the file belongs only inside a discarded attempt.
- *Regression:* the `connectFailures == attempts` whole-run abort still fires (the existing case's
  `summary.Incomplete` assertion covers it) and `MaxAttempts` is still 3.
- *Compile-time:* every reference to `ServerConnectFailureFile` is removed; the constant is deleted,
  not left orphaned.

**The ladder file and the new task files.** Not TDD, but gated before any real run:

- `LoadLadder` accepts the amended `ladder-toc.yaml` — add a config-level test asserting the six
  cells load and that ladders b, c and d each resolve exactly one control. This is the cheap guard
  against a typo costing thirty real runs.
- `LoadTaskFile` succeeds on `02-shedadapters-exploration.md` and on the new task 06 file, returning
  a non-empty task text and a schema block — the direct regression against task 02's current missing
  heading.
- Each new fasit parses as JSON, carries the exploration schema's three scored keys
  (`relevant_files`, `key_symbols`, `summary`) plus `confidence` and `open_questions`, and its
  `_meta.pinned_sha` equals the pin its ladder entry names.
- A dry sanity run of one new cell at `--reps 1` before the full matrix is **not** taken — it would
  consume real budget and, worse, produce a results root that is not the measured one. The offline
  e2e layer plus the loader tests are the pre-flight.

**The matrix itself** is not a test and has no assertions; its acceptance is the task's own done-when
— every measured cell a real MCP cell completing end to end, `5/5` complete non-blinding-failed
repetitions per cell, `unscored_count: 0`, and gate 1 (`granted_tool_used`) reported per rung cell so
the conclusion can say whether a flat rung actually called its tool.

## Q&A log

- **Q:** One new task shape or two? **A:** [auto-pick] Two. **Why:** T7 already showed one flat shape
  settles nothing; with a single new shape a flat result cannot be told apart from a badly chosen
  shape, and the task body authorises up to two.
- **Q:** Where does the multi-package shape come from — a new prompt or the drafted task 02?
  **A:** [auto-pick] Finish task 02. **Why:** the shape already exists at the same pin and its only
  blockers are a missing `## Output schema` heading and a missing fasit; writing a fresh three-package
  prompt would discard drafted work for no gain.
- **Q:** One ladder file and one results root, or a second ladder file? **A:** [auto-pick] Extend
  `ladder-toc.yaml` with ladders c and d, one invocation, one root. **Why:** cost numbers compare only
  within a root, so all six cells belong in one; `config.go` imposes no limit on ladder letters, so the
  file header's "two task groups per file" line is a stale comment to amend, not a rule.
- **Q:** How is each new fasit authored? **A:** [auto-pick] A dedicated reference-agent card,
  unlimited budget, cross-checked by a second independent method, `_meta` recording arm C and the
  method. **Why:** it reproduces the protocol tasks 01/04/05's committed fasits were produced by, so
  the new shapes are scored on the same footing.
- **Q:** What shape does the M2 reason file take? **A:** [auto-pick] One `invalid_reason.txt` covering
  every invalidation cause, replacing `server_connect_failure.txt`. **Why:** the T7 conclusion asked
  for the runner-level error and exit code specifically, and one file with a `cause` enumeration
  answers all four paths without leaving a reader to know which of two files to look in.
- **Q:** How is M2 proven by test? **A:** [auto-pick] e2e using `fakeclaude`'s `partial_fail` (exit 1)
  for `runner_error` and `no_fence` for `unparseable_answer`, plus the existing server-connect case
  retargeted. **Why:** the roadmap's done-when is literally an artificially failed attempt carrying a
  readable reason file, and the fixture already provides that failure offline.
- **Q:** Does M2 land before or after the matrix? **A:** [auto-pick] Before, committed, with the
  matrix's own non-control cells as its end-to-end proof. **Why:** a harness change cannot land
  mid-matrix, and the third harness rule is satisfied for free by the rung cells this task already runs.
- **Q:** What counts as "separates"? **A:** [auto-pick] `separated: true` on a cost metric **and** the
  median moving in the predicted direction **and** no correctness degradation. **Why:** `separated` is
  a strict range test that misses real effects at n=5, medians alone are what T7 showed can mislead,
  and a cost win bought with worse recall is not a win for this tool.
- **Q:** Does the conclusion propagate into the roadmap and HANDOFF? **A:** [auto-pick] Yes, both.
  **Why:** the roadmap names this conclusion as T8's unpark input, and T7's own finding was that the
  record must stop citing superseded results.
- **Q:** Same pin for the new tasks? **A:** [auto-pick] Yes, `975578cda8d6f3a81580bd4e73725e060211b766`.
  **Why:** one tree state under all three shapes means a difference between shapes is the shape.
- **Q:** What does the cold-start shape look like concretely? **A:** [auto-pick] Whole-repository,
  naming no package or file, exploration schema, subject chosen so the true answer spans at least two
  packages. **Why:** every shape measured so far hands the agent its scope, which is the confound; this
  is the condition where a first cheap survey is most plausibly worth its cost.
- **Q:** Cell ids? **A:** [auto-pick] `c0-none`/`c1-toc-dir`, `d0-none`/`d1-toc-dir`, ladder b's
  unchanged. **Why:** `<letter><n>-<rung>` is the established shape, `0` is the control everywhere, and
  `b8` is deliberately chosen to stay unique against the main matrix's ids.
- **Q:** What if a new shape's fasit turns out degenerate? **A:** [auto-pick] Swap the subject and
  re-author before the matrix starts, recording the swap; never mid-matrix. **Why:** task 02's own
  notes anticipate exactly this, and after the invocation the prompts are measured stimulus.
- **Q:** Reps? **A:** [auto-pick] 5. **Why:** fixed by the task body's cost discipline and by
  `ladder-toc.yaml`'s own `reps: 5`.
- **Q:** What happens at the discussion-review round cap if findings remain? **A:** [operator]
  Approve and hand off regardless. **Why:** the operator's explicit instruction this session — the
  round cap ends the review loop and proceeds to Handoff rather than blocking the task.
