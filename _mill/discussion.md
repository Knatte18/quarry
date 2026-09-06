# Discussion: M4 matrix run: execute the descoped kick-start batch (cards 29-32)

```yaml
task: 'M4 matrix run: execute the descoped kick-start batch (cards 29-32)'
slug: kickstart-matrix-run
status: discussing
parent: main
```

## Problem

The `ladder-kickstart` task built a benchmark harness for the *push-mode* half of the toc
hypothesis — whether pre-resolving a plan card's glyphs into file+span locations and injecting them
into the prompt before the agent starts pays for itself — and then descoped its last batch, the one
that actually spends money. The harness, the ladder file, the three cards, the task and its fasit are
all merged (`e1553fa`). What is missing is the measurement: nobody has run the 3x10 matrix, and so
`docs/roadmap.md` still carries "the M4 matrix run" as its point 1, blocking the one remaining
adoption decision (whether kick-start pack injection joins the Loomyard adoption).

**Why now:** every other blocker cleared. All three plan-alphabet primitives are on main, Loomyard
issue #226 is fully unblocked, and the only thing in the adoption still waiting on quarry is the
kick-start-pack piece — which waits on this number. The measurement rule was predeclared and frozen
before the harness was written precisely so that this run could be executed later, by someone else,
without renegotiation.

This task IS that descoped batch, executed verbatim. Its batch spec was recovered word-for-word from
the archived branch (`git show a69c999:_mill/descoped/07-matrix-and-writeup.md`) and is reproduced
in full inside the wiki task body. **Nothing about the measurement is open.** The discussion below
settles only the operational plan: environment, ordering, commit boundaries, execution mechanics,
and failure handling.

## Scope

**In:**

- Card 29 — run `ladder pack` against `ladder-kickstart.yaml` and a new results root; inspect its
  output; commit the generated treatment card plus `pack-resolve.json` and `provenance.json`.
- Card 30 — run `ladder run` (all three cells, no cell selection, no rep override); record the four
  per-cell dispositions; commit `summary.json`, `table.txt`, and the updated `provenance.json`.
- Card 31 — write `conclusion.md` in the new results root: hand-computed one-sided Mann-Whitney U on
  turns and cost_usd, e1-pack vs e0-names, with the arithmetic shown and recomputed a second time.
- Card 32 — two edits to `docs/roadmap.md` in one commit: add the new results root to the
  standing-rule paragraph, remove the now-completed point 1 (renumbering the rest), update the
  "Updated" date.
- The contingency branch inside card 29 (glyph substitution) if and only if a glyph fails to resolve
  `found`.

**Out:**

- **Any change to the measurement design.** The rule is predeclared: primary comparison e1-pack vs
  e0-names on `turns` and `cost_usd`, one-sided Mann-Whitney U (alternative: treatment lower),
  n = 10 per arm, alpha = 0.05, critical U <= 27; `e2-files` descriptive only, no test. No optional
  stopping, no dropped or added repetitions, no re-running an arm because its numbers look wrong.
- **Any harness code change.** No new Go code, no new fields in `summary.json`, no new subcommand,
  no test changes. `bench/loomyard-eval/ladder/internal/ladder/**` is read-only for this task, and
  so is `quarry/`, `glyph/`, `internal/`, `cmd/`.
- Any edit to the target repository (`/home/knatte/Code/loomyard/wts/loomyard`). It is read-only
  input; the harness pins its own worktree from `pinned_sha`.
- The "small and independent" roadmap bullet about creating the results root before writing the
  provenance record. Card 32's spec says to strike it; it is **already gone** (commit `9f3096a`
  restructured the file). Skip that edit — see Decision "roadmap-card-32-adaptation".
- Whether T8 unparks, whether the parked task stays parked, and whether an M4b edit-task variant is
  written. Card 32's own text says this card reports a measurement and does not decide that
  question.
- Any change to `.gitignore` or to the `results/*/raw/` ignore rule. The raw tree stays untracked.

## Decisions

### one-batch-four-cards

- **Decision:** the plan is a single batch, `matrix-and-writeup`, holding cards 29, 30, 31, 32 in
  that order, with `verify: go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestPreMatrix'`.
- **Rationale:** the descoped spec's own justification still holds verbatim: the four steps are
  strictly sequential against one results root and share one date, and splitting them would let the
  roadmap point at a root whose conclusion had not been written. The task body's instruction is "do
  not redesign anything".
- **Rejected:** two batches (29+30 measure, 31+32 write up) — introduces a batch boundary in the
  middle of a single results root's lifetime; four single-card batches — same problem, four times.

### live-matrix-runs-detached

- **Decision:** card 30 launches the run **detached**, not as a foreground Bash call:

  ```
  nohup go run ./bench/loomyard-eval/ladder/cmd/ladder run \
    --config bench/loomyard-eval/ladder/ladder-kickstart.yaml \
    --results <RESULTS_ROOT> \
    > .scratch/kickstart-run.log 2>&1 &
  ```

  then polls `.scratch/kickstart-run.log` (`tail`) until the harness's own rendered table appears at
  the end of the log and the process has exited.
- **Rationale:** ~30 measured claude invocations plus up to 30 scorer invocations is on the order of
  an hour of wall-clock; a foreground Bash call is capped at 600 s and would be killed. Detached +
  log-polling behaves identically whether the caller is this orchestrator or a mill-go implementer
  subagent, and does not depend on any harness background-notification semantics. The run is
  resumable by design: completed repetitions are skipped and the resumed invocation is appended to
  the provenance record.
- **Liveness test.** `tail` alone cannot distinguish "still running" from "died"; the log goes quiet
  in both cases. Record the launched shell's pid (`$!`) at launch and test it with `kill -0 <pid>`,
  or, equivalently, `pgrep -f 'ladder(/| ).*run'`. The run is finished only when the process is gone
  **and** the log's last lines carry the rendered table; the process gone without a table is a death.
- **Resume preconditions — all three, in this order, before re-running.** A resume is not a bare
  re-run of the same command:
  1. **Confirm nothing is live.** `kill -0 <pid>` fails and no `go run .../cmd/ladder` or `ladder`
     process remains (`pgrep -f`). Never resume alongside a live run.
  2. **Clear the stale lock.** `AcquireRunLock` (`worktree.go:293-320`) opens
     `~/.cache/ladder-eval/.ladder.lock` with `O_CREATE|O_EXCL`, **never reaps it, and never tests
     process liveness** — its own doc comment says a stale lock is cleared by the operator. An
     abnormally dead run therefore always leaves one behind, and the resume fails with
     "another ladder run holds ...". Read the file first (it records `pid=` and `results=`), confirm
     the pid is dead and the results root is this one, then delete it.
  3. **Commit the tree clean again.** `Run` calls `WriteProvenance` after **every** repetition
     (`run.go:224`), and card 29 committed `provenance.json`, so by resume time that tracked file is
     modified in the working tree. Committing it before the resumed invocation is what keeps
     `quarry_dirty: false` — the property `commit-before-each-invocation` exists to buy, which a
     naive resume would otherwise silently forfeit. Commit message:
     `chore(bench): checkpoint provenance before resuming the kick-start matrix`.

  Then re-run the **identical** command against the **same** results root — never with a `--reps`
  override (`run.go:100-103` refuses a differing effective count, which is correct under a locked n).
- **A resume is permitted only for an abnormal process death, and only before any result has been
  read.** Resuming after the invocation completed normally — to refill a repetition the harness
  ended `incomplete`, once `table.txt` and `summary.json` exist — is optional stopping through a side
  door and is forbidden. See Decision `short-arm-disposition`.
- **If `quarry_dirty: true` is nevertheless recorded** for any invocation in this root (a resume that
  skipped step 3, or any other cause), it is **not** grounds to discard the root. The conclusion's
  coverage section must then name the invocation, quote `quarry_dirty_files` verbatim, and state
  whether any listed path could affect the measurement — mirroring how the reference root's
  conclusion handles its own `quarry_dirty: false` claim explicitly rather than silently.
- **Rejected:** foreground with a 600 s timeout and repeated resumes (works, but every truncation
  kills a live `claude` child mid-repetition and manufactures avoidable `.invalid-*` attempts);
  operator runs the matrix by hand outside mill-go (defeats the point of the task being in the
  pipeline, and the roadmap's "(operator-driven)" note describes who pays for it, not who types it).

### commit-before-each-invocation

- **Decision:** immediately before **each** of the two harness invocations (`pack`, then `run`), the
  worktree must be committed clean — `git status --porcelain` empty except for ignored paths.
- **Rationale:** `CollectInvocation` records `git status --porcelain` of the quarry repository root,
  and `ResolveQuarryRepoRoot` walks up from cwd to the nearest `.git`, which in this worktree is
  *this worktree*, not the hub. `_mill/**` is tracked (`.gitignore` ignores only `**/_mill/*.active`),
  so uncommitted mill bookkeeping would land in `provenance.json`'s `quarry_dirty_files` and force
  the conclusion to carry a carve-out paragraph. The reference root
  (`results/2026-09-05-ladder-d`) records `quarry_dirty: false`, and matching that costs one commit.
- **Deviation from `docs/roadmap.md` point 1, deliberate and recorded.** That point says the bench is
  run "from the hub against main". This task runs it from its own spawned worktree instead, because
  that is where the task lives and the pipeline executes; the alternative is a hub run whose results
  this branch could not commit. The measurable consequence is confined to provenance:
  `provenance.json`'s `quarry_commit` will name this task branch's tip rather than a `main` commit,
  unlike the reference root (`0ae4daa…`, which the ladder-d conclusion notes differs from `main` by a
  single `_mill/status.md` bookkeeping commit and by no source file). **The conclusion's coverage
  section must state this explicitly**: name the branch, name the commit, and confirm — with
  `git diff --stat main...<commit> -- . ':(exclude)_mill'` — that the branch differs from `main` by
  no file the harness or the target repository reads, so "from clean `main`" holds in substance even
  though the recorded commit is not on `main`. If that diff is ever non-empty outside `_mill/` and
  the new results root, the conclusion says so and names the files rather than asserting equivalence.
- **Rejected:** accepting `quarry_dirty: true` and documenting a `_mill`-only carve-out (real work
  avoided by a `git commit`); running from the hub worktree (the task's own branch state would then
  be invisible to the run, the hub is not this task's worktree, and card 29's and card 30's commits
  would have nowhere to land).

### short-arm-disposition

- **Decision, predeclared here and therefore before rep 1:** an arm can legitimately finish with
  fewer than ten measured repetitions. `runCellRepetition` returns `repOutcome{incomplete: true}`
  with **no further retry** once `attempts >= MaxAttempts` (`run.go:574-584`), and
  `summarize.go:232-239` does not count a blinding-failed repetition as present for completeness. In
  either case the cell appears in `summary.json`'s `incomplete` list and `run` exits non-zero. The
  disposition is:

  - **A short arm voids the primary test for every comparison it participates in.** If either
    `e0-names` or `e1-pack` realises fewer than ten repetitions in the cost sample, the conclusion
    still reports everything — medians, ranges, the full per-repetition table, both rank sums and
    both U values computed on the realised sample sizes, with the two identities checked against
    those sizes — but it **does not compare U against 27, nor against any substituted critical
    value**, and the verdict line states that the predeclared test is void for want of the
    predeclared n. No conclusion about separation is drawn from a voided test.
  - **The root is not re-run, not re-sampled, not extended, and no arm is dropped.** Refilling the
    missing repetition after `table.txt` exists is optional stopping through a side door, however
    innocent the cause.
  - **A short `e2-files` arm voids nothing**, since no test runs on it. Report its realised n
    alongside its medians and ranges and move on.
  - **Realised n is reported per arm unconditionally**, alongside the ceiling and unscored counts, so
    a full n = 10/10 matrix is visibly full rather than merely unremarked.
- **Rationale:** the predeclared rule fixes alpha = 0.05 one-sided and n = 10 per arm; `U <= 27` is
  the exact-table lookup *for n = 10 against n = 10*. Substituting the critical value for a realised
  n = 9 would be renegotiating the frozen rule after seeing the run, and re-running the missing
  repetition would be optional stopping. Voiding is the only reading that leaves the rule untouched,
  and it costs nothing that was not already lost — every cost measurement in the root still stands
  and is still reported.
- **Rejected:** recomputing against the correct critical value for the realised n (changes a frozen
  constant post hoc); declaring the whole root unusable (discards ~$12 of valid cost measurements
  over one harness event); resuming to refill the arm after the table exists (optional stopping).

### date-resolved-once-at-card-29

- **Decision:** card 29 resolves the date **once**, with `date -u +%F`, and uses it to spell the
  results root as `bench/loomyard-eval/ladder/results/<YYYY-MM-DD>-kickstart/`. It then records the
  resolved absolute results-root path in the batch's own working notes. Cards 30, 31 and 32 reuse
  that recorded path verbatim and never re-derive the date.
- **Rationale:** the spec is explicit — "the date is a fact about the run, not a plan constant" — and
  spells its own paths as `2026-09-05` only as a placeholder. Resolving once and threading the
  resolved value forward removes the one way this can go wrong (a run that crosses midnight UTC
  producing a card-31 path that does not exist).
- **Rejected:** hard-coding today's date in the plan (breaks if mill-go runs on a later day); keeping
  the spec's literal `2026-09-05` (wrong by construction — the ladder-d root already owns that date).

### glyph-substitution-contingency-is-fully-specified

- **Decision:** the three reserve candidates the spec alludes to are recovered and named here, so
  card 29's contingency branch is executable without operator input:

  - `internal/fabricengine#Fabric.MergeAbort`
  - `internal/fabricengine#Fabric.mergeStateOrForeignErr`
  - `internal/gitrepo#Repo.ConflictedFiles`

  If and only if `ladder pack` reports a target that did not resolve `found`, replace the offending
  glyph with one of those three (same package, same mechanism) and then, **in this order**:

  1. edit `pack_targets:` in `bench/loomyard-eval/ladder/ladder-kickstart.yaml`;
  2. **hand-edit the `Uses:` list in all three cards** — `ladder pack` rewrites only the pack cell's
     sentinel block, so `07-e0-names.md` and `07-e2-files.md` would otherwise keep naming a symbol
     `07-e1-pack.md` no longer lists, which is an arm difference in precisely the dimension under
     test;
  3. **re-derive `07-e2-files.md`'s `Files:` list** from the new glyph set, deduplicated — the
     substitute may live in a file no other glyph does, or may vacate one;
  4. re-run `ladder pack`, which rewrites the pack block and the provenance record;
  5. record the substitution and its reason under
     `bench/loomyard-eval/tasks/07-fabric-merge-state-tracing.md`'s
     `## Notes for whoever prepares C's fasit / scores this`;
  6. re-run the fasit's cross-check — `git show 72c23d9:<file>` for the substitute symbol, confirming
     it exists at the pinned SHA in the file the resolve reports, which is the same second-method
     check the fasit itself was authored under.

  All of it happens **before rep 1 or not at all** (the frozen spec spells this "before rep 0";
  it means the same instant — the harness indexes repetitions from 1, `for rep := 1; rep <=
  repsEffective`, and `verifyCardsAndPack` is the pre-rep-1 gate). After it, check by eye that the
  three cards' `Uses:` lists are still identical and still match the ladder file's `pack_targets`,
  then run `TestPreMatrix` as the mechanical backstop — the test catches the half-applied
  substitution where the ladder file has moved on and one card has not.

  **The fasit is frozen.** `07-fabric-merge-state-tracing.fasit.json` is never edited by this branch
  — not its `relevant_files`, not its `key_symbols`, not its `summary`. It defines what the scorer
  grades against, and the batch-local decision freezes it alongside the glyph list and n; editing it
  mid-substitution would move the correctness target as well as the treatment, and no test guards
  either list.

  **Consequence, and how the conclusion handles it.** All three reserves live in files already on the
  fasit's seven-file `relevant_files` list — `Fabric.MergeAbort` and `Fabric.mergeStateOrForeignErr`
  in `internal/fabricengine/mergelifecycle.go` (lines 366 and 221 at the pin),
  `Repo.ConflictedFiles` in `internal/gitrepo/merge.go` (line 157). A substitution can therefore only
  ever **vacate** a file, never add one. Step 3 still re-derives e2's `Files:` list from the new
  glyph set, exactly as the frozen spec directs — so if the substitution vacates a file, e2 ends up
  naming six files against the fasit's seven. When that happens, the conclusion's
  recall-inflation paragraph must be reworded rather than transcribed: it says that e1's card names
  the fasit's seven files verbatim inside its pack block while e2 names only the six its glyph set
  reaches, so **both** non-control arms remain inflated by construction and still uncomparable to the
  control, but e2's inflation is no longer maximal — and it names which file was vacated. If no
  substitution occurs, the paragraph stands as written in "What the conclusion must contain".
- **Rationale:** the source discussion (`git show a69c999:_mill/discussion.md`, lines 325-338) names
  the three candidates verbatim; without them the branch would require operator judgement mid-run,
  which under an autonomous pipeline means either a halt or an unlogged treatment redesign.
- **Rejected:** halting to the operator on any non-`found` glyph (unnecessary now that the reserve
  list is recovered); letting the implementer pick substitutes by its own judgement (an unrecorded
  change to the treatment).

### gate-dispositions-tallied-from-score-json

- **Decision:** card 30 records, per cell, four dispositions from two sources:

  - **turn-ceiling count** -> `summary.json`'s per-cell `max_turns_count`;
  - **unscored count** -> `summary.json`'s per-cell `unscored_count`;
  - **harness events** (blinding failures, attempt-exhausted repetitions, `.invalid-*` attempts) ->
    `summary.json`'s `blinding_failed_count` and `incomplete` / `invalid` lists, plus the
    `raw/<cell>/<rep>.invalid-*/invalid_reason.txt` files. A cell in the `incomplete` list means the
    arm realised fewer than ten repetitions — read its realised n from `table.txt`'s `n` column and
    apply Decision `short-arm-disposition`;
  - **gate passed / gate failed** -> hand-tallied from each repetition's
    `raw/<cell>/<rep>/score.json` `"summary_matches"` value, counted with `grep`.
- **Rationale:** `ScoreRecord` is a free-form `map[string]any`, and neither `summarize.go` nor
  `report.go` aggregates `summary_matches` into `summary.json` or `table.txt` — verified by grep;
  only `recall`, `precision` and the three exclusion counters are aggregated. The gate split the card
  demands therefore has to come from the per-repetition score files.
- **Rejected:** adding a `summary_matches` aggregate to `summarize.go` (a harness change, explicitly
  out of scope for a task whose job is to run the harness as merged); reporting only what
  `summary.json` carries and omitting the gate split (the card names four dispositions, not three).

### roadmap-card-32-adaptation

- **Decision:** card 32 becomes **three** edits in one commit, not the spec's two:

  1. Add **one sentence** to the standing-rule paragraph naming the new results root and saying in
     one clause what it measured and what it found — worded so it reads as the *other direction*
     (push-mode pre-resolution: glyph spans resolved into the prompt before the agent starts), not as
     a fourth entry in the paragraph's existing mid-session-agent-tool list. The paragraph today
     cites three roots (`results/2026-09-04-toc`, `results/2026-09-04-breadth`,
     `results/2026-09-05-ladder-d`) as closing negative "for quarry as a *mid-session agent tool*";
     this root does not test that claim, and folding it into that list would misreport it.
  2. Remove point 1 ("The M4 matrix run (operator-driven)") from `## The order of work`, and
     renumber points 2, 3, 4 to 1, 2, 3. Point 2's closing clause ("Independent of point 1 — only the
     kick-start-pack piece waits on the matrix result") must be rewritten to refer to the conclusion
     rather than to a point that no longer exists.
  3. Update the file's `Updated <date>` line on line 3 to the date of this change.
- **Rationale:** the spec's first edit — striking the small-and-independent bullet about creating the
  results root before writing the provenance record — is already done: commit `9f3096a` restructured
  `docs/roadmap.md` and that bullet is gone. Re-applying it is impossible. The spec's surviving
  intent is edits (1) and (3); the wiki task body's operational note 3 adds (2) explicitly.
  Renumbering forces the point-2 cross-reference fix, which is not optional — leaving it would leave
  a dangling reference.
- **Rejected:** appending the new root as a fourth item in the parenthesised list (conflates two
  different claims); giving it its own paragraph (the card says "add one line to the standing-rule
  paragraph"); leaving point 1 in place (the task body says to remove it).

### done-gate-runs-with-the-env-var-unset

- **Decision:** the repository-wide done gate is `go test ./... && golangci-lint run`, run with
  `LADDER_LOOMYARD_REPO` **unset in the shell environment**. Do not `export` it anywhere. The bench
  reaches the target repository through `.scratch/ladder.env` instead.
- **Rationale:** `internal/engine/loomyard_test.go:53` reads `os.Getenv("LADDER_LOOMYARD_REPO")`
  directly — never the `.scratch` fallback — and at `:55`/`:59` **skips** when it is unset or not a
  directory. At `:69` it **fails** when the checkout's HEAD is not the pin
  `72c23d9eecc1fa55add567622093a8bbbfba8c1d`; the operator's checkout is at `408b910`. Leaving the
  variable unset makes the golden test skip while `ResolveLoomyardRepo`
  (`bench/loomyard-eval/ladder/internal/ladder/worktree.go:112-144`) still finds the repository via
  the file. This is exactly what the wiki task body's operational note 1 asks for.
- **Rejected:** exporting it and tolerating the pin failure (a red done gate); skipping the engine
  package in the done gate (hides a real test).

## Technical context

### Where things are

- Harness entry point: `bench/loomyard-eval/ladder/cmd/ladder/main.go` — three subcommands,
  `run`, `report`, `pack`.
  - `pack --config <ladder> --results <root> [--claude-bin claude]`
  - `run --config <ladder> --results <root> [--cells a,b] [--reps N] [--claude-bin claude]`
  - `report --results <root>` — re-derives `summary.json` and `table.txt` from the raw tree without
    running or scoring anything. Useful for re-reading a completed root; **not** part of the spec's
    card sequence.
- Ladder file: `bench/loomyard-eval/ladder/ladder-kickstart.yaml`. Its header documents the exact
  two invocations to use, through a bare `go run` entry point. `run_model: claude-sonnet-5`,
  `run_effort: medium`, `max_turns: 60`, `reps: 10`; scorer `claude-opus-5` / `high`.
- Cards: `bench/loomyard-eval/cards/07-e0-names.md` (control, `Uses:` only),
  `07-e1-pack.md` (treatment: `Uses:`, an **empty** sentinel block, and the parallel-read
  instruction), `07-e2-files.md` (descriptive: `Uses:` plus a seven-entry `Files:` list). All three
  `Uses:` lists are currently identical and match the ladder file's nine `pack_targets`.
- Task + fasit: `bench/loomyard-eval/tasks/07-fabric-merge-state-tracing.md` and `.fasit.json`.
- Reference conclusion to imitate: `bench/loomyard-eval/ladder/results/2026-09-05-ladder-d/conclusion.md`.
  Its section order is the template: identity/invocation facts -> "this root stands alone" ->
  numbers table -> rule restated -> per-repetition table -> per-metric rank sums / U / verdict ->
  correctness -> verdict section -> coverage (gates, invalidations, drift, provenance) -> what this
  settles.
- Reference table: `.../2026-09-05-ladder-d/table.txt`. Column names differ from `summary.json`'s
  keys — `cache_read` is `cache_read_input_tokens`, `cache_creation` is
  `cache_creation_input_tokens`, `prefixed_tool_uses` is `quarry_tool_uses`, `grep_fallback` is
  `grep_fallback_total`. The conclusion must say which spelling it quotes.

### How the pack mechanism works (card 29)

- `Pack` (`pack.go:227`) loads the ladder file, finds the single `pack: true` cell, prepares the
  pinned worktree at `TaskWorktreePath(worktreeRoot, "07-fabric-merge-state-tracing")`, makes exactly
  **one** batched `(*quarry.Repo).Resolve` call over all nine `pack_targets`, renders the block, and
  writes it into `07-e1-pack.md` between `<!-- KICKSTART-PACK:BEGIN -->` and
  `<!-- KICKSTART-PACK:END -->`.
- `RenderKickstartPack` (`pack.go:47`) is **fatal** on any result whose status is not
  `StatusFound`, whose `Error` is non-empty, or which carries other than exactly one symbol. It
  returns an error naming the offending target and emits no partial output. A clean exit from
  `ladder pack` therefore *is* the confirmation that all nine glyphs resolved `found` — there is no
  separate check to run.
- Rendered shape is two lines per target: `<target> → <file> <start>-<end>` then four spaces and the
  signature with internal newlines collapsed. **No docstring** is ever emitted. Card 29's eyeball
  check is: each line names a plausible file and span, and no docstring prose leaked in.
- `Pack` writes `pack-resolve.json` (`PackResolveFile`) and merges a full provenance invocation
  naming every config id and `reps_effective: 10`. **Consequence, deliberate:** the effective
  repetition count is pinned from the pack onward, so a later `--reps` override against the same root
  is refused by `run.go:100`. Under a locked n that is correct behaviour, not an obstacle.
- `Pack` acquires the same advisory run lock `Run` does, directly under the worktree root, released
  on exit. A dead pack or run leaves a stale lock that is **never** reaped automatically and must be
  cleared by hand.

### How the run gate works (card 30)

- `verifyCardsAndPack` (`run.go:258-320`) runs before rep 1 and does two things: it re-reads every
  selected cell's card, and it re-hashes the pack cell's sentinel block and compares it to
  `provenance.json`'s `kickstart_pack.pack_sha256`. A mismatch is a hard stop with "run the pack
  subcommand again". It compares **only the pack hash, never repository state** — which is exactly
  why committing between card 29 and card 30 is the expected workflow and cannot brick the root.
- `run.go:100-103` refuses an invocation whose effective repetition count differs from the one the
  root was written with. Resume by re-running the identical command; never pass `--reps`.
- Raw tree layout: `<results-root>/raw/<cell>/<rep>/` holding `usage.json` (the per-repetition
  metrics card 31's table quotes) and `score.json` (the scorer's record, including
  `summary_matches`). Invalidated attempts land in `raw/<cell>/<rep>.invalid-<n>/` with an
  `invalid_reason.txt`. `bench/loomyard-eval/ladder/.gitignore` ignores `results/*/raw/`, anchored to
  that directory — the raw tree is never committed.
- `summarizeAndReport` (`main.go:172`) runs automatically at the end of `run`: it writes
  `summary.json`, writes and prints `table.txt`, and exits non-zero if any cell is incomplete or
  invalid. A non-zero exit is a real signal, not noise.

### Environment

- `ResolveQuarryRepoRoot` (`worktree.go:89`) walks up from cwd to the nearest `.git` — in this
  worktree that is a gitfile, and `os.Stat` accepts it, so the root resolves to
  `/home/knatte/Code/quarry/wts/kickstart-matrix-run`. **All harness commands must be run from this
  worktree's root**, and never after a `cd` into the hub.
- `ResolveLoomyardRepo` (`worktree.go:112`) reads `$LADDER_LOOMYARD_REPO`, else parses
  `<quarry-repo-root>/.scratch/ladder.env`. **That file already exists in this worktree** and matches
  the hub's: `LADDER_LOOMYARD_REPO=/home/knatte/Code/loomyard/wts/loomyard`. The wiki body's
  "copy it from the hub" step is already satisfied — verify, do not re-copy.
- `ResolveWorktreeRoot` (`worktree.go:163`) resolves `$LADDER_WORKTREE_ROOT`, else
  `$XDG_CACHE_HOME/ladder-eval`, else `~/.cache/ladder-eval`. It **refuses** any path that is the
  quarry repo root, is under it, or contains the case-insensitive substring `quarry`. On this
  machine it resolves to `~/.cache/ladder-eval`, which carries **no** stale lock file. Its
  `worktrees/` subdirectory currently holds only `probe` (three relocated log files, not a git
  worktree); task 07's pinned worktree does not exist, so `PrepareWorktree` creates it during
  card 29.
- Target repository state, verified: `/home/knatte/Code/loomyard/wts/loomyard` is clean, HEAD is
  `408b91033c34e4ec6af621f80cb3afcc40247e96`, and the pinned commit
  `72c23d9eecc1fa55add567622093a8bbbfba8c1d` **exists** in it (`git cat-file -t` -> `commit`). The
  checkout's own HEAD does not matter to the bench, which pins its own worktree.

### Roadmap current state (card 32 input)

`docs/roadmap.md` today: line 3 carries `Updated 2026-09-05`; lines 8-13 are the standing-rule
paragraph citing three results roots; `## The order of work` has four numbered points, of which
point 1 is the M4 matrix run and point 2 ends "Independent of point 1 — only the kick-start-pack
piece waits on the matrix result"; `## Small and independent, any time` has two bullets, neither of
them the results-root/provenance one the spec says to strike.

## Constraints

- No `CONSTRAINTS.md` at the hub root.
- `CLAUDE.md`: this is a Go repo; do not introduce Python. The gate tally in card 30 therefore uses
  `grep`, not a script.
- Never `cd` out of this worktree. Read parent state with `git -C <path> ...` only.
- Never write to `/tmp` or any system temporary directory; ephemeral files go under `.scratch/`
  (gitignored via the repo-root `**/.scratch/` entry).
- Do not use `sed`.
- **Money and irreversibility.** Card 30 spends real money (~$10-15) against the live API and is the
  only card in this task that does. It is resumable but not repeatable: once rep 1 has run, the
  glyph list, the three cards, the task file, the fasit and n are frozen for the whole results root.
  Every correction the substitution rule allows happens before rep 1 or not at all.
- **No optional stopping, in any form.** Do not drop repetitions after seeing their scores, do not
  add repetitions after seeing any result, do not re-run an arm because its numbers look wrong, and
  do not select cells. Each is optional stopping through a side door. This includes resuming a
  *completed* invocation to refill a repetition the harness ended `incomplete` — see Decision
  `short-arm-disposition`; the only legitimate resume is after an abnormal process death, before any
  result has been read.
- Write the same shape of conclusion whichever way the result falls. A negative answer is a valid,
  publishable answer.

## Testing

There is no new code in this task, so there is nothing to TDD. Verification is by procedure, and
each procedure is named in the card that owns it.

- **Batch verify:** `go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestPreMatrix'`.
  Confirmed green on the current tree. It is a **regression net over the committed artefacts**, not a
  test of this batch's work: `TestPreMatrix_KickstartCellPromptsAreBlind`,
  `TestPreMatrix_KickstartCardsShareOneUsesList`,
  `TestPreMatrix_KickstartUsesListMatchesPackTargets`,
  `TestPreMatrix_KickstartPackCellCardHasSentinels` and `TestPreMatrix_NewFasitsAreWellFormed`
  between them assert the identical-`Uses:`-lists property, the ladder-file/card glyph agreement, the
  sentinel protocol, the fasit's shape and the three prompts' blindness. Its job here is to catch a
  half-applied glyph substitution.
- **Card 29 is verified by the pack command's own pre-rep-0 gate.** A clean exit means all nine
  glyphs resolved `found`; there is no additional assertion to write. The human-judgement part —
  plausible file and span per line, no leaked docstring — is an eyeball check of the rendered block
  in `07-e1-pack.md`, plus a read of `pack-resolve.json`.
- **Card 30 is verified by the run's own pre-repetition verification** (`verifyCardsAndPack`), which
  must pass. A failure there is a **stop**, not something to work around by regenerating the pack
  after the cards have been committed.
- **Card 31 is verified by recomputing the whole statistic a second time** from the per-repetition
  table, with two identities as the check: the two rank sums must total `N(N+1)/2` = 210 for
  N = 20, and the two U values must total `n1 * n2` = 100. Ranks are computed with ties averaged. No
  code computes any of this, so the second pass is the only check on it.
- **Card 32 is verified by reading the rendered file**: three edits present, no dangling reference to
  the removed point, nothing else changed.
- **Done gate** (end of task, per `commit-before-each-invocation` and
  `done-gate-runs-with-the-env-var-unset`): `go test ./... && golangci-lint run` with
  `LADDER_LOOMYARD_REPO` unset.

## Execution order

The four cards are strictly sequential. Concretely:

1. Verify `.scratch/ladder.env` exists in this worktree and names the loomyard checkout. Verify no
   stale lock file directly under `~/.cache/ladder-eval/`. Do **not** export `LADDER_LOOMYARD_REPO`.
2. Resolve `RUN_DATE=$(date -u +%F)` and
   `RESULTS_ROOT=bench/loomyard-eval/ladder/results/$RUN_DATE-kickstart`. Record both.
3. Commit any pending worktree state so `git status --porcelain` is empty.
4. **Card 29** — `go run ./bench/loomyard-eval/ladder/cmd/ladder pack --config
   bench/loomyard-eval/ladder/ladder-kickstart.yaml --results $RESULTS_ROOT`. On a non-`found`
   glyph, take the contingency branch in `glyph-substitution-contingency-is-fully-specified` in full,
   then return here. On success, inspect the rendered block and `pack-resolve.json`, run
   `TestPreMatrix`, then commit `07-e1-pack.md`, `$RESULTS_ROOT/pack-resolve.json` and
   `$RESULTS_ROOT/provenance.json` (plus any file the contingency branch touched) with message
   `feat(bench): generate the kick-start pack for the e1 card`.
5. Confirm the tree is clean again. **Card 30** — launch the run detached per
   `live-matrix-runs-detached`, recording the pid; poll the log and the pid together. If the process
   died without printing the table, walk the three resume preconditions in that decision (nothing
   live, clear `~/.cache/ladder-eval/.ladder.lock`, commit the tree clean) and only then re-run the
   identical command. **Read no result until the run has completed normally** — after that point no
   resume is permitted, per `short-arm-disposition`.
   Then read the printed table and `summary.json`, tally the four dispositions plus the realised n
   per `gate-dispositions-tallied-from-score-json` and `short-arm-disposition`, and commit
   `$RESULTS_ROOT/summary.json`,
   `$RESULTS_ROOT/table.txt` and `$RESULTS_ROOT/provenance.json` with message
   `feat(bench): run the kick-start 3x10 matrix`.
6. **Card 31** — write `$RESULTS_ROOT/conclusion.md` per the requirements below, recompute the
   arithmetic a second time, commit with message `docs(bench): write the kick-start matrix conclusion`.
7. **Card 32** — the three roadmap edits in one commit, message
   `docs: record the kick-start measurement in the roadmap`.
8. Done gate.

## What the conclusion must contain (card 31)

Following `results/2026-09-05-ladder-d/conclusion.md`'s structure:

- The run's own identity and invocation facts, and an explicit statement that **this root stands
  alone** and is not pooled with any other.
- Per-cell medians and ranges **quoted verbatim** from `summary.json` / `table.txt`, never
  re-derived. Note which spelling is being quoted where the two files differ.
- The predeclared rule **restated before the numbers**, transcribed rather than reinterpreted:
  primary comparison is e1-pack against e0-names; primary metrics are `turns` and `cost_usd`;
  one-sided Mann-Whitney U with the alternative that the treatment is lower; ten per arm; alpha of
  five percent; critical U at or below twenty-seven. Reject the null for a metric only when its U is
  at or below that value. The descriptive arm gets medians and ranges only and no test.
- The per-repetition table, values taken from each repetition's own `usage.json`.
- The rank sums, the U values, and the two identity checks. At a full n = 10 against n = 10 those are
  `R1 + R0 = N(N+1)/2 = 210` and `U1 + U0 = n1*n2 = 100`; if either tested arm is short, both
  identities are recomputed for the realised sizes and checked against those values instead.
- The run's own host, branch and commit, and — because the run is not from `main` — the
  branch-versus-`main` equivalence check named in Decision `commit-before-each-invocation`.
- Per arm, regardless of whether any threshold is crossed: the **realised n**, the turn-ceiling count
  and the unscored count, so a clean 10/10 matrix is visibly clean. Then the predeclared readings:
  - **realised n below ten in either tested arm** -> the primary test is **void** for every
    comparison that arm participates in, per Decision `short-arm-disposition`: report the rank sums
    and U values computed on the realised sizes, do **not** compare them against 27 or any
    substituted critical value, say so in the verdict line, and draw no separation conclusion from
    them. No re-run, no re-sample, no dropped arm. A short `e2-files` arm voids nothing — state its
    realised n and continue;
  - more than two of ten at the ceiling in any arm -> the turns test is **reported as censored**, said
    so in the verdict line, with cost carrying the primary comparison — and no re-run, no re-sample,
    no dropped arm;
  - more than two of ten gate failures in any arm -> that arm's cost numbers are called **suspect**
    and still reported;
  - more than two of ten unscored in any arm -> that arm's correctness accounting is called
    **incomplete** while its cost metrics stand.
- A plain statement that recall and precision are **descriptive only and never compared across
  arms**, and why: both non-control arms have file recall inflated by construction — the treatment's
  card names the seven files verbatim inside its pack block, and the descriptive card names the same
  seven as a plain list derived from the fasit's own relevant-files list. The control is the only arm
  whose file recall is earned. This is a known property of the design, stated as such rather than
  discovered afterwards.
- The other known asymmetry: e1 minus e2 is **not** a clean spans contrast, because the treatment
  also carries the signature and the parallel-read instruction.
- Secondary observations — read bytes, wall time, recall of the listed symbols in the answer — are
  reported and never tested.
- A verdict that says, in one sentence, which of two readings the numbers support: if the treatment
  does not separate either, the surface has now been measured from both directions and the parked
  condition is closed twice over; if it does separate, the win belongs to **pre-resolution** rather
  than to a tool, which is a different product from the one the parked task assumed.

## Q&A log

- **Q:** Batch structure for the plan — one batch of four cards, or split? **A:** [auto-pick] One batch of four cards, exactly as the descoped spec declares. **Why:** the spec's own rationale still holds (strictly sequential against one results root, one shared date, roadmap must not point at an unwritten conclusion), and the task body says do not redesign.
- **Q:** How is card 30's ~1-hour live matrix executed given the 600 s Bash timeout? **A:** [auto-pick] Launch detached with `nohup … > .scratch/kickstart-run.log 2>&1 &` and poll the log; re-run the identical command if the process died. **Why:** survives the timeout, behaves the same in the orchestrator and in a mill-go implementer subagent, and relies on the harness's documented resume rather than on manufactured mid-repetition kills.
- **Q:** Should the worktree be committed clean before the pack and run invocations? **A:** [auto-pick] Yes — commit immediately before each of the two invocations so both record `quarry_dirty: false`. **Why:** `CollectInvocation` reads `git status --porcelain` of this worktree and `_mill/**` is tracked, so uncommitted mill bookkeeping would otherwise pollute `quarry_dirty_files` and force a carve-out paragraph in the conclusion.
- **Q:** How is the results-root date fixed? **A:** [auto-pick] Card 29 resolves `date -u +%F` once, records the resolved root path, and cards 30-32 reuse it verbatim. **Why:** the spec says the date is a fact about the run, not a plan constant; resolving once removes the midnight-crossing failure mode.
- **Q:** What happens if a glyph does not resolve `found`? **A:** [auto-pick] Execute the full six-step substitution branch autonomously, using the three reserve candidates recovered from `git show a69c999:_mill/discussion.md`. **Why:** the reserve list is no longer missing, so the contingency is fully specified; without it the branch would need operator judgement mid-run, meaning either a halt or an unrecorded treatment redesign.
- **Q:** How should card 32 word the new standing-rule line, given the paragraph already cites three roots? **A:** [auto-pick] One sentence in the same paragraph, worded as the *other direction* (push-mode pre-resolution), explicitly not a fourth entry in the mid-session-agent-tool list. **Why:** the three cited roots close a claim this root does not test; folding it in would misreport the measurement.
- **Q:** Where do card 30's gate pass/fail counts come from? **A:** [auto-pick] Hand-tallied with `grep` from each repetition's `raw/<cell>/<rep>/score.json` `summary_matches`; ceiling and unscored counts come from `summary.json`. **Why:** `ScoreRecord` is a free-form map and neither `summarize.go` nor `report.go` aggregates `summary_matches`; adding that aggregation would be a harness change, which is out of scope.
- **Q:** (review r2 gap) What exactly does a resume require after the run process dies? **A:** [auto-pick] Three preconditions before re-running: confirm nothing live via the recorded pid, delete the stale `~/.cache/ladder-eval/.ladder.lock`, and commit the tree clean. **Why:** `AcquireRunLock` is `O_CREATE|O_EXCL` and by its own doc comment never reaps and never tests liveness, so a dead run always blocks its own resume; and `Run` rewrites `provenance.json` after every repetition, so the file card 29 committed is modified by resume time and would record `quarry_dirty: true` — silently forfeiting the property the commit-before-invocation decision exists to buy.
- **Q:** (review r2 gap) What happens if an arm finishes with fewer than ten repetitions? **A:** [auto-pick] The primary test is reported **void** for every comparison that arm participates in — U computed and shown on the realised sizes, never compared against 27 or a substituted critical value — with no re-run, no re-sample and no dropped arm. **Why:** `attempts >= MaxAttempts` returns `incomplete: true` with no further retry, so a short arm is reachable; substituting the critical value for the realised n would renegotiate a frozen constant after seeing the run, and refilling the repetition would be optional stopping. Voiding leaves the rule untouched and still reports every cost measurement the root produced.
- **Q:** How is the repo-wide done gate run given the env-gated engine test? **A:** [auto-pick] `go test ./... && golangci-lint run` with `LADDER_LOOMYARD_REPO` unset in the shell. **Why:** `internal/engine/loomyard_test.go:53-59` reads the variable directly and skips when unset, but fails at `:69` when the checkout is not at the `72c23d9` pin (it is at `408b910`); the bench still finds the repository through `.scratch/ladder.env`.
