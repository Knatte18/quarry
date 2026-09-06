# Batch: matrix-and-writeup

```yaml
task: 'M4 matrix run: execute the descoped kick-start batch (cards 29-32)'
batch: 'matrix-and-writeup'
number: 1
cards: 4
verify: go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestPreMatrix'
depends-on: []
```

## Batch Scope

This batch executes the descoped kick-start measurement verbatim: generate the pack (card 29), run
the 3x10 matrix (card 30), write the conclusion (card 31), and record the result in the roadmap
(card 32). It is one batch because the four steps are strictly sequential against a single results
root and share a single resolved date; splitting them would let the roadmap point at a results root
whose conclusion had not been written yet. There is no new Go code and no interface for a later
batch to consume — the deliverable is a committed results root plus a roadmap edit.

Batch-local notes beyond `## Shared Decisions` in the overview:

- **`<RUN_DATE>` is a placeholder, not a directory name.** Every path in this batch spelled
  `bench/loomyard-eval/ladder/results/<RUN_DATE>-kickstart/...` means the results root card 29
  resolves. Card 29 resolves it once and records it; cards 30-32 read the recorded value.
- **Card 30 spends real money** (~$10-15 against the live API) and is the only card that does. It is
  resumable but not repeatable: once repetition 1 has run, the glyph list, the three cards, the task
  file, the fasit and n are frozen for the whole results root.
- **The fasit is frozen.** `bench/loomyard-eval/tasks/07-fabric-merge-state-tracing.fasit.json` is
  never edited by this batch — not its `relevant_files`, not its `key_symbols`, not its `summary`.
- **Every harness command runs from this worktree's root.** The quarry repository root is resolved by
  walking up from the working directory to the nearest `.git`, which in this worktree is a gitfile
  that resolves here.

## Cards

### Card 29: generate the kick-start pack for the e1 card

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/pack.go`
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
  - `bench/loomyard-eval/ladder/internal/ladder/prematrix_test.go`
  - `bench/loomyard-eval/ladder/cmd/ladder/main.go`
  - `bench/loomyard-eval/ladder/.gitignore`
  - `bench/loomyard-eval/tasks/07-fabric-merge-state-tracing.fasit.json`
  - `.scratch/ladder.env`
- **Edits:**
  - `bench/loomyard-eval/cards/07-e1-pack.md`
  - `bench/loomyard-eval/cards/07-e0-names.md`
  - `bench/loomyard-eval/cards/07-e2-files.md`
  - `bench/loomyard-eval/ladder/ladder-kickstart.yaml`
  - `bench/loomyard-eval/tasks/07-fabric-merge-state-tracing.md`
- **Creates:**
  - `.scratch/kickstart-results-root.txt`
  - `bench/loomyard-eval/ladder/results/<RUN_DATE>-kickstart/pack-resolve.json`
  - `bench/loomyard-eval/ladder/results/<RUN_DATE>-kickstart/provenance.json`
- **Deletes:** none
- **Moves:** none
- **Requirements:**

  **Preflight, in this order.**

  1. Confirm `.scratch/ladder.env` exists in this worktree and its `LADDER_LOOMYARD_REPO` value names
     the loomyard checkout. Verify it; do not re-copy it, and do not `export`
     `LADDER_LOOMYARD_REPO` into the shell environment at any point in this batch.
  2. Resolve the worktree root and bind it as `<worktree-root>` for the rest of this batch. The
     harness resolves it as `LADDER_WORKTREE_ROOT` if set, else `$XDG_CACHE_HOME/ladder-eval`, else
     `~/.cache/ladder-eval` — and it refuses any path that is the quarry repository root, is under
     it, or contains the case-insensitive substring `quarry`. Do not assume the third alternative;
     read the environment and record which one applies. Then confirm there is no stale lock file at
     `<worktree-root>/.ladder.lock`: the lock is opened with `O_CREATE|O_EXCL`, is never reaped, and
     never tests process liveness, so a leftover file from an earlier dead run blocks this one. If
     one is present, read it (it records `pid=` and `results=`), confirm the pid is dead with
     `kill -0`, and delete it.
  3. Resolve the date once: `RUN_DATE=$(date -u +%F)` and
     `RESULTS_ROOT=bench/loomyard-eval/ladder/results/$RUN_DATE-kickstart`. Write two lines into
     `.scratch/kickstart-results-root.txt`: one spelled `RESULTS_ROOT=` followed by the
     worktree-relative results-root path, and one spelled `LADDER_WORKTREE_ROOT_RESOLVED=` followed
     by the worktree root resolved in step 2. Cards 30, 31 and 32 read that file rather than
     re-deriving the date or re-resolving the worktree root.
  4. Commit any pending worktree state so `git status --porcelain` is empty outside ignored paths,
     per the `commit-clean-before-each-harness-invocation` Shared Decision.

  **Run the pack subcommand** from this worktree's root:

  ```
  go run ./bench/loomyard-eval/ladder/cmd/ladder pack \
    --config bench/loomyard-eval/ladder/ladder-kickstart.yaml \
    --results $RESULTS_ROOT
  ```

  `RenderKickstartPack` is fatal on any resolve result whose status is not `StatusFound`, whose
  `Error` is non-empty, or which carries other than exactly one symbol; it names the offending target
  and emits no partial output. A clean exit therefore is itself the confirmation that all nine
  `pack_targets` resolved `found` — there is no separate check to run. On a clean exit the command
  has rewritten the block between `<!-- KICKSTART-PACK:BEGIN -->` and `<!-- KICKSTART-PACK:END -->`
  in `bench/loomyard-eval/cards/07-e1-pack.md`, and written `pack-resolve.json` plus a provenance
  invocation recording `reps_effective: 10` under the results root.

  **Contingency branch — enumerate the failures first.** The pack subcommand's error message names
  exactly one target no matter how many failed: the renderer returns on the *first* result whose
  `Error` is non-empty, whose status is not `StatusFound`, or which carries other than one symbol,
  and it returns before the resolve record is written, so no `pack-resolve.json` exists to read the
  rest out of. The pack command's own error is therefore not a count and must never be used as one.

  On a pack failure, enumerate the full resolve set independently before deciding anything. The
  pinned worktree the pack command prepared is at `<worktree-root>/worktrees/07-fabric-merge-state-tracing`,
  where `<worktree-root>` is the resolved worktree root defined in the preflight above; it survives
  the failed pack. For each of the nine entries in `pack_targets` in
  `bench/loomyard-eval/ladder/ladder-kickstart.yaml`, run the repository's own resolver against that
  worktree — `go run ./cmd/quarry resolve <glyph> --root <pinned worktree> --text` — and judge each
  answer by the same gate the pack renderer applies, not by the command's exit code: the target
  passes only when the `--text` output's first line ends in the status word `found` **and** exactly
  one symbol line follows it. Anything else — `not_found`, `ambiguous`, `error`, `multipart`, or a
  `found` answer carrying zero or more than one symbol — is a failure. Keying on the exit code alone
  would undercount: the resolver exits 0 for a multipart answer, which the pack renderer treats as
  fatal. The count of failures in that enumeration, never the pack command's message, is what the
  halt test below is applied to.

  **Zero failures in the enumeration is itself a halt.** The pack command failed but every target
  resolves cleanly on its own, which means the fault is in the pack path — the batched resolve, the
  pinned worktree, the card's sentinels, or the write — and not in a glyph. Substituting a glyph
  would paper over it. Record what the pack command reported and stop.

  **Substitution, only when the enumeration shows exactly one failing target.** Apply the
  deterministic substitution table below. The three reserve candidates are
  `internal/fabricengine#Fabric.MergeAbort`,
  `internal/fabricengine#Fabric.mergeStateOrForeignErr` and `internal/gitrepo#Repo.ConflictedFiles`.
  A reserve already present in `pack_targets` is skipped — it cannot substitute for itself.

  | offending glyph's package | action |
  |---|---|
  | `internal/fabricengine` | substitute `Fabric.mergeStateOrForeignErr`; if that itself does not resolve `found`, `Fabric.MergeAbort`; if both fail, halt |
  | `internal/gitrepo` | substitute `Repo.ConflictedFiles`; if it does not resolve `found`, halt |
  | `internal/fabriccli` | halt — no same-package reserve exists |
  | `internal/mergeresolve` | halt — no same-package reserve exists |

  Test each candidate reserve with the same `go run ./cmd/quarry resolve <glyph> --root <pinned worktree> --text`
  invocation and the same status-word-plus-one-symbol gate the enumeration uses, before editing
  anything — that is what "does not resolve `found`" means in the table's cascade.

  Two or more targets failing simultaneously in that enumeration is also a halt regardless of
  package: the reserve list
  holds three entries for one expected failure, and multiple failures indicate a pin, checkout or
  extractor problem rather than a bad glyph. Halt means halt — do not substitute across packages, do
  not relax the `found` gate, do not touch the loomyard checkout, and do not proceed to card 30.
  Record what failed and why under the `## Notes for whoever prepares C's fasit / scores this`
  section of `bench/loomyard-eval/tasks/07-fabric-merge-state-tracing.md`, commit, and stop.

  When the table yields a substitute, perform these six steps in this order:

  1. edit `pack_targets:` in `bench/loomyard-eval/ladder/ladder-kickstart.yaml`;
  2. hand-edit the `Uses:` list in all three cards — `bench/loomyard-eval/cards/07-e0-names.md`,
     `bench/loomyard-eval/cards/07-e1-pack.md` and `bench/loomyard-eval/cards/07-e2-files.md` —
     because the pack subcommand rewrites only the pack cell's sentinel block, so the other two cards
     would otherwise keep naming a symbol the treatment card no longer lists, which is an arm
     difference in precisely the dimension under test;
  3. re-derive the `Files:` list in `bench/loomyard-eval/cards/07-e2-files.md` from the new glyph set,
     deduplicated;
  4. re-run the same pack command, which rewrites the pack block and the provenance record;
  5. record the substitution and its reason under the
     `## Notes for whoever prepares C's fasit / scores this` section of
     `bench/loomyard-eval/tasks/07-fabric-merge-state-tracing.md`;
  6. re-run the fasit's own cross-check for the substitute symbol against the read-only loomyard
     checkout named by `LADDER_LOOMYARD_REPO` in `.scratch/ladder.env` — a `git -C <that checkout>
     show 72c23d9:<file>`, never a bare `git show`, since `72c23d9` is the loomyard pin from
     `pinned_sha` and does not exist in this quarry worktree — confirming the symbol exists at the
     pinned SHA in the file the resolve reports.

  All of this happens before repetition 1 or not at all; the run loop indexes repetitions from 1 and
  gates them behind `verifyCardsAndPack`.

  **Inspection.** Read the rendered block in `bench/loomyard-eval/cards/07-e1-pack.md` by eye: two
  lines per target, the first naming a plausible file and span, the second the signature with
  internal newlines collapsed, and no docstring prose leaked in. Read the generated
  `pack-resolve.json` under the results root and confirm it names all nine targets. Then confirm by
  eye that the three cards' `Uses:` lists are still identical to each other and still match
  `pack_targets` in `bench/loomyard-eval/ladder/ladder-kickstart.yaml`, and run the batch verify
  command as the mechanical backstop against a half-applied substitution.

  **Commit** the edited treatment card, `pack-resolve.json` and `provenance.json` under the results
  root, plus any file the contingency branch touched. `.scratch/kickstart-results-root.txt` is
  ignored and is not committed.
- **Commit:** `feat(bench): generate the kick-start pack for the e1 card`

### Card 30: run the kick-start 3x10 matrix

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
  - `bench/loomyard-eval/ladder/internal/ladder/report.go`
  - `bench/loomyard-eval/ladder/internal/ladder/score.go`
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
  - `bench/loomyard-eval/ladder/cmd/ladder/main.go`
  - `bench/loomyard-eval/ladder/ladder-kickstart.yaml`
  - `.scratch/kickstart-results-root.txt`
- **Edits:**
  - `bench/loomyard-eval/ladder/results/<RUN_DATE>-kickstart/provenance.json`
- **Creates:**
  - `.scratch/kickstart-run.log`
  - `bench/loomyard-eval/ladder/results/<RUN_DATE>-kickstart/summary.json`
  - `bench/loomyard-eval/ladder/results/<RUN_DATE>-kickstart/table.txt`
- **Deletes:** none
- **Moves:** none
- **Requirements:**

  Read `.scratch/kickstart-results-root.txt` and bind both recorded values: the results root as
  `$RESULTS_ROOT` and the resolved worktree root as `<worktree-root>`.
  Confirm the tree is clean again (`git status --porcelain` empty outside ignored paths) before
  launching, per the `commit-clean-before-each-harness-invocation` Shared Decision.

  **Launch the run detached**, not as a foreground call — roughly thirty measured agent invocations
  plus up to thirty scorer invocations is on the order of an hour of wall clock, and a foreground
  shell call would be killed at its timeout:

  ```
  nohup sh -c 'go run ./bench/loomyard-eval/ladder/cmd/ladder run \
    --config bench/loomyard-eval/ladder/ladder-kickstart.yaml \
    --results <RESULTS_ROOT>; echo "LADDER_EXIT=$?"' \
    > .scratch/kickstart-run.log 2>&1 &
  ```

  Pass no `--cells` and no `--reps`. The run loop refuses an invocation whose effective repetition
  count differs from the one the root was written with, and under a locked n that is correct.

  **`Deletes: none` above is literal.** No file is deleted by this card, tracked or ignored, on any
  branch — including the aborted-matrix branch below. The raw tree under `$RESULTS_ROOT/raw/` is
  git-ignored and the harness writes and rewrites it freely; nothing in this card removes any part of
  it. The one exception in the whole batch is a stale run-lock file outside the results root, cleared
  under resume precondition 2.

  **Poll for completion.** Tail `.scratch/kickstart-run.log` until the trailing `LADDER_EXIT=<code>`
  line appears. Tailing alone cannot distinguish "still running" from "died" — the log goes quiet in
  both cases — so the authoritative liveness check is the lock file's own `pid=` line at
  `<worktree-root>/.ladder.lock` tested with `kill -0`, together with
  `pgrep -f 'cmd/ladder|ladder run'`. The `$!` handle from the launch is the `go run` parent shell,
  a different process from the ladder child the lock records; it shows whether the launch survived at
  all and is never proof that the ladder process is gone.

  **Completion predicate — three outcomes.** Read no number from any file until this branch resolves;
  a printed table does not prove a full matrix, because an aborted run still writes and prints
  `summary.json` and `table.txt` through `summarizeAndReport`.

  - `LADDER_EXIT=0` — normal completion. Proceed to the reading step below. No resume is permitted
     from here.
  - No `LADDER_EXIT` line at all — abnormal process death (kill, logout, OOM). Walk the three resume
     preconditions below and re-run the identical command against the same results root.
  - `LADDER_EXIT=1` — three different things produce this code and they are not interchangeable.
     Classify it under the branch below before touching any number. A non-zero exit is **not** by
     itself evidence of an aborted matrix: the run loop sets its non-zero-exit flag for any repetition
     that came back `incomplete` or `blindingFailed`, and the summariser exits non-zero whenever the
     `incomplete` or `invalid` list is non-empty — none of which stops the matrix.

  **Resume preconditions — all three, in this order.** A resume is not a bare re-run.

  1. Confirm nothing is live: `kill -0 <pid>` fails and `pgrep -f` finds no remaining ladder process.
     Never resume alongside a live run.
  2. Clear the stale lock at `<worktree-root>/.ladder.lock`. Read it first, confirm the recorded
     pid is dead and the recorded results root is this one, then delete it.
  3. Commit the tree clean again. The run loop rewrites the provenance record after every repetition
     and card 29 committed that file, so by resume time it is modified in the working tree; committing
     it first is what keeps `quarry_dirty: false`. Use the message
     `chore(bench): checkpoint provenance before resuming the kick-start matrix`.

  Then re-run the identical command against the same results root, never with a `--reps` override.

  **Classifying a `LADDER_EXIT=1` run — two tests, three outcomes.** Read nothing else — no median,
  no rank, no `n` — until this branch has resolved.

  **Test 1: did this invocation reach the summariser at all?** Look for a rendered table in this
  invocation's own `.scratch/kickstart-run.log` (the launch truncates the log each time, so what it
  holds is this invocation only). The summariser that writes and prints the table is reached only
  when the run function returns a nil error; when it returns an error, the command wraps it, prints
  it and exits 1 without ever writing `summary.json` or `table.txt`. Checking for the file on disk is
  not a substitute: on a resumed root both files may already exist from an earlier invocation.

  - **No rendered table, an error line instead — a hard error, not a matrix outcome.** The run
    function returns an error for a held or stale run lock, an effective-repetition-count mismatch, a
    pack-hash mismatch in the pre-repetition card verification, a resume-time memory-path finding, or
    any mid-matrix repetition error. The matrix is truncated or never started, and nothing about it
    may be read as a result. Read the error line and act on the fault it names:
    - held or stale lock — walk resume precondition 2, then the other two, then re-run;
    - repetition-count mismatch — the invocation passed a differing `--reps`; re-run the identical
      command with no `--reps` at all;
    - pack-hash mismatch — **stop**. The pack and the committed cards have diverged. Do not
      regenerate the pack after the cards have been committed; surface this to the operator;
    - memory-path finding or a mid-matrix repetition error — fix the fault the message names, then
      walk the three resume preconditions and re-run the identical command.

  **Test 2, only when a table was rendered: was the matrix aborted?** The discriminator is whether
  some repetition's own `run.json` under `$RESULTS_ROOT/raw/<cell>/<rep>/` carries a fatal finding
  whose gate is `memory_path_scan`. Grep for that gate name across the raw tree's `run.json` files.

  - **A `memory_path_scan` fatal finding is present — the matrix was aborted.** This is the only
    reachable abort in this configuration: the memory-path scanner returns a fatal finding when any
    memory file matches the bare token `quarry`, or when a named memory path does not exist, and that
    branch stops the whole invocation after writing the repetition complete with the blinding-failed
    flag. (The other abort branch — every attempt failing to connect the same server — cannot occur
    here, since no cell grants tools and no server is ever built.) The taint is an environment fault
    about the machine, detected without reading a single score, and the run loop's own comment says
    the abort exists so a resumed invocation cannot skip past the repetition that revealed it — so
    resume is the designed recovery, not optional stopping. Fix the environment (remove the offending
    token from the named memory file, or create the missing memory path), then walk the three resume
    preconditions and re-run the identical command. **Delete nothing.** The completeness predicate the
    resume loop guards on is state `complete` *and not* blinding-failed, so a repetition written
    complete-with-the-blinding-failed-flag is already not counted complete and is re-attempted in
    place; deleting its directory would destroy the transcript and the finding the harness wrote
    there deliberately. Record the resume and the finding text for card 31's coverage section.
  - **No `memory_path_scan` fatal finding — the matrix ran to completion and the exit code is
    reporting short or invalidated repetitions, not an abort.** Both reachable causes leave the run
    loop running: a repetition that exhausted its attempt ceiling comes back `incomplete` with no
    abort flag, and a repetition the post-run blinding check fails comes back `blindingFailed` with
    no abort flag. Neither is an environment fault and neither licenses a resume or a deletion.
    Treat the run as complete, read the results, and let card 31 disposition the realised n under the
    `no-optional-stopping` Shared Decision. Do not re-run, re-sample, extend or drop an arm.

  Because a completed matrix can exit 1 this way, a `blinding_failed_count` above zero in
  `summary.json` on its own never establishes an abort, and neither does an `n` column below ten in
  `table.txt`. Only the gate name in `run.json` does.

  **No repetition directory is deleted by this card, under any branch.** The frozen batch spec
  anticipated one under the taint disposition; the harness's own completeness predicate makes it
  unnecessary, and card 31's coverage section records that fact rather than a deletion.

  **Read the results, once the completion predicate has resolved to a completed matrix** — either
  `LADDER_EXIT=0`, or `LADDER_EXIT=1` with no `memory_path_scan` fatal finding. Read the printed table and
  `summary.json` under the results root, and record per cell:

  - turn-ceiling count — the per-cell `max_turns_count`;
  - unscored count — the per-cell `unscored_count`;
  - harness events — `blinding_failed_count` plus the `incomplete` and `invalid` lists, and the
    `raw/<cell>/<rep>.invalid-*/invalid_reason.txt` files;
  - gate passed and gate failed — hand-tallied with `grep` over each repetition's
    `raw/<cell>/<rep>/score.json` `"summary_matches"` value. Neither the summarizer nor the reporter
    aggregates that field, so it has to come from the per-repetition score files. Use `grep`, not a
    script — this is a Go repository and no Python is introduced.

  Also record each arm's realised n from the `n` column of `table.txt`. A cell appearing in the
  `incomplete` list realised fewer than ten repetitions; that is a legitimate outcome and card 31
  dispositions it. Do not re-run, re-sample, extend or drop an arm to repair it.

  **Commit** `summary.json`, `table.txt` and `provenance.json` under the results root. The raw tree is
  ignored and is never committed.
- **Commit:** `feat(bench): run the kick-start 3x10 matrix`

### Card 31: write the kick-start matrix conclusion

- **Context:**
  - `bench/loomyard-eval/ladder/results/2026-09-05-ladder-d/conclusion.md`
  - `bench/loomyard-eval/ladder/results/2026-09-05-ladder-d/table.txt`
  - `bench/loomyard-eval/ladder/results/2026-09-05-ladder-d/summary.json`
  - `bench/loomyard-eval/ladder/results/<RUN_DATE>-kickstart/summary.json`
  - `bench/loomyard-eval/ladder/results/<RUN_DATE>-kickstart/table.txt`
  - `bench/loomyard-eval/ladder/results/<RUN_DATE>-kickstart/provenance.json`
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
  - `bench/loomyard-eval/ladder/internal/ladder/report.go`
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
  - `bench/loomyard-eval/ladder/internal/ladder/metrics.go`
  - `bench/loomyard-eval/ladder/ladder-kickstart.yaml`
  - `bench/loomyard-eval/cards/07-e0-names.md`
  - `bench/loomyard-eval/cards/07-e1-pack.md`
  - `bench/loomyard-eval/cards/07-e2-files.md`
  - `bench/loomyard-eval/tasks/07-fabric-merge-state-tracing.fasit.json`
  - `.scratch/kickstart-results-root.txt`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/results/<RUN_DATE>-kickstart/conclusion.md`
- **Deletes:** none
- **Moves:** none
- **Requirements:**

  Write the conclusion into the results root recorded in `.scratch/kickstart-results-root.txt`,
  following the section order of `bench/loomyard-eval/ladder/results/2026-09-05-ladder-d/conclusion.md`:
  identity and invocation facts, "this root stands alone", the numbers table, the rule restated, the
  per-repetition table, per-metric rank sums and U values with the verdict, correctness, the verdict
  section, coverage, and what this settles. Write the same shape of document whichever way the result
  falls — a negative answer is a valid, publishable answer.

  It must contain, at minimum:

  - The run's own identity and invocation facts, and an explicit statement that this root stands alone
    and is not pooled with any other.
  - Per-cell medians and ranges quoted verbatim from `summary.json` and `table.txt` under this results
    root, never re-derived. The two files spell some columns differently — `cache_read` against
    `cache_read_input_tokens`, `cache_creation` against `cache_creation_input_tokens`,
    `prefixed_tool_uses` against `quarry_tool_uses`, `grep_fallback` against `grep_fallback_total` —
    so state which spelling is being quoted where they differ.
  - The predeclared rule restated before the numbers, transcribed rather than reinterpreted: primary
    comparison e1-pack against e0-names; primary metrics `turns` and `cost_usd`; one-sided
    Mann-Whitney U with the alternative that the treatment is lower; ten per arm; alpha of five
    percent; critical U at or below twenty-seven; reject the null for a metric only when its U is at
    or below that value; the descriptive arm gets medians and ranges only and no test.
  - The per-repetition table, values taken from each repetition's own `raw/<cell>/<rep>/usage.json`.
  - The rank sums, the U values, and both identity checks. At a full ten against ten those are
    `R1 + R0 = N(N+1)/2 = 210` and `U1 + U0 = n1*n2 = 100`; if either tested arm is short, recompute
    both identities for the realised sizes and check against those values instead. Ranks are computed
    with ties averaged.
  - The run's host, branch and commit, and — because the run is not from `main` — the equivalence
    check: run `git diff --stat main...<commit> -- . ':(exclude)_mill'` and confirm the branch differs
    from `main` by no file the harness or the target repository reads, so "from clean `main`" holds in
    substance even though the recorded commit is not on `main`. If that diff is non-empty outside
    `_mill/` and this results root, say so and name the files rather than asserting equivalence.
  - Per arm, unconditionally: the realised n, the turn-ceiling count and the unscored count, so a
    clean ten-of-ten matrix is visibly clean. Then the predeclared readings:
    - realised n below ten in either tested arm — the primary test is **void** for every comparison
      that arm participates in. Report the rank sums and U values computed on the realised sizes, do
      not compare them against 27 or any substituted critical value, say so in the verdict line, and
      draw no separation conclusion from them. No re-run, no re-sample, no dropped arm. A short
      `e2-files` arm voids nothing — state its realised n and continue;
    - more than two of ten at the ceiling in any arm — the turns test is reported as censored, said so
      in the verdict line, with cost carrying the primary comparison;
    - more than two of ten gate failures in any arm — that arm's cost numbers are called suspect and
      still reported;
    - more than two of ten unscored in any arm — that arm's correctness accounting is called
      incomplete while its cost metrics stand.
  - A plain statement that recall and precision are descriptive only and never compared across arms,
    and why: both non-control arms have file recall inflated by construction, since the treatment card
    names the seven files verbatim inside its pack block and the descriptive card names the same seven
    as a plain list. The control is the only arm whose file recall is earned. If card 29's contingency
    branch vacated a file, reword this paragraph to say that both non-control cards name the same six
    of the fasit's seven relevant files, naming which file was vacated and by which substitution, and
    add that the vacated seventh is named by no card in any arm — which narrows but does not remove
    the inflation.
  - The other known asymmetry: e1 minus e2 is not a clean spans contrast, because the treatment also
    carries the signature and the parallel-read instruction.
  - Secondary observations — read bytes, wall time, recall of the listed symbols in the answer —
    reported and never tested.
  - A verdict in one sentence naming which of two readings the numbers support: if the treatment does
    not separate either, the surface has now been measured from both directions and the parked
    condition is closed twice over; if it does separate, the win belongs to pre-resolution rather than
    to a tool, which is a different product from the one the parked task assumed.
  - One sentence on the M4b condition, in three branches matching the three possible readings: on a
    non-separating result, that the edit-task variant does not become a candidate and why; on a
    separating result, that it does, with card 32 adding it to the roadmap; and on a **void** result,
    that the condition is neither discharged nor met — the run could not test it — so it carries
    forward unchanged and awaits a future measurement.
  - In the coverage section: if the run was ever resumed, the invocation count from the provenance
    record's `invocations` and why each resume happened — and, for a resume after a memory-path taint,
    the cell, the repetition and the finding text, plus the fact that the repetition was re-attempted
    in place. Assert "No permitted repetition deletion" explicitly, matching the reference root: card
    30 deletes no repetition directory on any branch, so this root can make the same assertion the
    reference root makes. If any invocation recorded
    `quarry_dirty: true`, name the invocation, quote `quarry_dirty_files` verbatim, and state whether
    any listed path could affect the measurement — that is not grounds to discard the root.

  **Verification is the second pass.** No code computes any of this, so recompute the whole statistic
  a second time from the per-repetition table and check both identities before committing. If a
  recomputation disagrees with the first pass, find and fix the arithmetic rather than picking one.
- **Commit:** `docs(bench): write the kick-start matrix conclusion`

### Card 32: record the kick-start measurement in the roadmap

- **Context:**
  - `bench/loomyard-eval/ladder/results/<RUN_DATE>-kickstart/conclusion.md`
  - `.scratch/kickstart-results-root.txt`
- **Edits:**
  - `docs/roadmap.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**

  Four edits to `docs/roadmap.md`, in one commit.

  1. **Standing-rule paragraph.** Add one sentence naming the new results root and saying in one
     clause what it measured and what it found. Spell the root the way the paragraph already spells
     its three — relative to the ladder directory, as `results/<RUN_DATE>-kickstart`, not as the
     repository-relative `bench/loomyard-eval/ladder/results/<RUN_DATE>-kickstart` this plan uses.
     If card 31's primary test came out void, say that in the same sentence — the root measured the
     other direction but its predeclared test could not be applied for want of the predeclared n —
     rather than reporting a finding the run does not support. Word it as the *other direction* —
     push-mode
     pre-resolution, glyph spans resolved into the prompt before the agent starts — and not as a
     fourth entry in the paragraph's existing parenthesised list of three roots. That list closes a
     negative for quarry as a mid-session agent tool; this root does not test that claim, and folding
     it in would misreport it.
  2. **Remove point 1 from `## The order of work`** — the M4 matrix run — and renumber the remaining
     points 2, 3 and 4 to 1, 2 and 3. Point 2's closing clause, "Independent of point 1 — only the
     kick-start-pack piece waits on the matrix result", must be rewritten to refer to the conclusion
     rather than to a point that no longer exists.
  3. **Preserve the M4b conditional**, whose only record is the point being deleted. Its last sentence
     reads "If — and only if — e1 separates, an edit-task variant (M4b: agent revising code in a
     throwaway worktree) becomes a candidate follow-up." The file's charter is that it only ever says
     what is ahead, so the disposition depends on the measured result:
     - e1 does not separate — the condition is discharged, not lost. Delete it from the roadmap; card
       31's verdict section already records in one sentence that the M4b edit-task variant does not
       become a candidate, citing this measurement as the reason.
     - e1 separates — M4b is now genuinely ahead. Add it as a new numbered point in
       `## The order of work`, immediately after the Loomyard-adoption point, phrased as the deleted
       sentence phrased it, pointing at the new results root as its justification. Edit 2's
       renumbering then yields, in order: 1 Loomyard adopts glyphs, 2 M4b, 3 T8, 4 More languages —
       so the two T8 and More-languages points end at 3 and 4 rather than at 2 and 3.
     - the primary test is void (card 31 reports a realised n below ten in a tested arm) — the
       condition is neither discharged nor met, because the run could not test it. Carry it forward
       unchanged: keep the M4b sentence, verbatim, attached to whichever point survives the
       renumbering as the nearest home for it, and say in the same clause that it awaits a future
       measurement because this root's primary test was void. Do not delete it and do not promote M4b
       to its own numbered point.
  4. **Update the `Updated <date>` line** on line 3 to the date of this change.

  Do not touch the `## Small and independent, any time` section: the frozen spec's other edit — the
  bullet about creating the results root before writing the provenance record — was already removed by
  an earlier restructuring commit and cannot be re-applied.

  Verify by reading the rendered file: all four edits present, no dangling reference to the removed
  point, and nothing else changed.
- **Commit:** `docs: record the kick-start measurement in the roadmap`

## Batch Tests

`verify:` is `go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestPreMatrix'`, scoped by
`-run` to the pre-matrix regression net over the committed artefacts rather than the whole package.
It covers `bench/loomyard-eval/ladder/internal/ladder/prematrix_test.go`:
`TestPreMatrix_KickstartCellPromptsAreBlind`, `TestPreMatrix_KickstartCardsShareOneUsesList`,
`TestPreMatrix_KickstartUsesListMatchesPackTargets`,
`TestPreMatrix_KickstartPackCellCardHasSentinels`, `TestPreMatrix_NewFasitsAreWellFormed` and
`TestPreMatrix_ControlPromptsAreBlind`. The last of those is selected by the `-run` prefix but is
keyed on `ladder-toc.yaml` rather than on this batch's artefacts — the selector is deliberately
broader than the checks the batch relies on, and it costs nothing to leave it in. It is green on the
current tree and stays green throughout this batch; its job here is to catch a half-applied glyph
substitution in card 29, where the ladder file has moved on and one card has not.

There is no new code in this batch, so there is nothing to TDD. Everything else is verified by
procedure inside the card that owns it:

- Card 29 is verified by the pack subcommand's own pre-repetition gate — a clean exit means all nine
  glyphs resolved `found` — plus an eyeball check of the rendered block and a read of the generated
  resolve record.
- Card 30 is verified by the run's own pre-repetition card-and-pack verification, which must pass. A
  failure there is a stop, not something to work around by regenerating the pack after the cards have
  been committed.
- Card 31 is verified by recomputing the whole statistic a second time, with the two identities as the
  check.
- Card 32 is verified by reading the rendered file.

The repository-wide done gate is `go test ./... && golangci-lint run`, already configured in
`mill-config.yaml`, run with `LADDER_LOOMYARD_REPO` unset in the shell.
