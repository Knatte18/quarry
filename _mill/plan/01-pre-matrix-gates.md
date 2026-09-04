# Batch: pre-matrix-gates

```yaml
task: "Ladder, toc rerun (T7)"
batch: "pre-matrix-gates"
number: 1
cards: 2
verify: null
depends-on: []
```

## Batch Scope

This batch produces the one committed artifact the matrix must be preceded by — the §9a probe
report — and confirms that the results root's raw tree will land untracked before anything writes
into it. It is one batch because both cards are cheap, offline, and must be finished and committed
before the matrix's first invocation: the clean-tree rule in `## Shared Decisions` makes "everything
committed" a hard gate, and a probe report written after the run would be a record produced after
the thing it gates. The interface the next batch consumes is a clean working tree plus an existing
results root directory containing `probe.md`.

Batch-local decision: the probe is **not** re-run here. The task constraint reads "the operator's
§9a live probe runs before the matrix; if it has not been reported done, stop and ask", and the
answer arrived through discussion review round 1 — it is reported done. This batch records that
report; it does not generate fresh evidence and must not claim to.

## Cards

### Card 1: Record the §9a probe report as `probe.md`

- **Context:**
  - `_mill/discussion.md`
  - `_mill/reviews/20260904-110823-discussion-review-r1.md`
  - `docs/rewrite-plan.md`
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/results/2026-09-04-toc/probe.md`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create the results root directory and write `probe.md` into it. The file records
  the operator's plan §9a live probe as an **operator report**, and says so in its own opening
  sentence — it is not a transcript this task captured, and no such transcript exists in the tree to
  copy. The file must contain, at minimum: (a) the date, 2026-09-04; (b) what was probed — the
  merged quarry MCP server built from the parent branch; (c) the three §9a properties and their
  outcomes, namely that the server connected and was listed in the session's `system` record, that
  an `mcp__quarry__toc` call returned the §4 envelope, and that a call outside the allowlist was
  refused and appeared in `permission_denials`; (d) the flags that made the denial half meaningful,
  in particular `--setting-sources ""`, without which the operator's global `defaultMode: "auto"`
  auto-approves read-only MCP calls and the denial half silently tests nothing — note that the
  harness passes the same flag for every measured repetition, which is what makes the hand probe
  harness-faithful; and (e) the `claude` CLI version the probe ran under, or an explicit statement
  that it was not recorded. This host is on 2.1.236 while plan §9a's own probe table was taken on
  2.1.259, and a silent omission of the version would read as an agreement the report has not
  earned, so state the discrepancy plainly. The body must **transcribe the round-1 discussion
  review's answer verbatim**, as a block quote, attributed to
  `archive/ladder-toc-rerun:_mill/reviews/20260904-110823-discussion-review-r1.md` — the tag form,
  not a bare `_mill/` path, because this task merges as one squash commit and its task-state tree
  survives only under that archive tag, so a bare path would be unresolvable on the parent branch.
  The verbatim source is the `[NIT:consistency]` finding titled "probe-before-matrix overrides the
  task's stop-and-ask constraint on a false premise" in that review file; quote its **Issue:**
  paragraph, which is where the probe outcome is stated. State the task constraint as the task wrote
  it — the probe runs before the matrix and a not-reported-done probe stops and asks — and then
  state that it was reported done, rather than paraphrasing the constraint into something the worker
  performs. Read `run.go` only to confirm the `--setting-sources ""` claim about what a measured
  repetition passes; do not change it.
- **Commit:** `bench(ladder): record the section 9a probe report in the 2026-09-04-toc results root`

### Card 2: Confirm the results root's raw tree is ignored

- **Context:**
  - `bench/loomyard-eval/ladder/.gitignore`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** A zero-diff verification card. Confirm that the ignore rule shipped by T2 covers
  the new results root before the matrix writes ten transcripts into it. Run
  `git check-ignore -v bench/loomyard-eval/ladder/results/2026-09-04-toc/raw/a0-none/1/transcript.jsonl`
  and confirm it reports the `results/*/raw/` pattern from the ladder directory's own ignore file as
  the matching rule. Then run `git status --porcelain` and confirm the tree is clean after card 1's
  commit — this is the clean-tree gate the matrix batch depends on, checked here while it is still
  cheap to fix. If either check fails, stop and report: an unignored raw tree would put resolved
  auto-memory machine paths on the branch, which the repository forbids, and the fix is a
  `.gitignore` change that must land before any measurement runs. Record both command outputs in the
  batch's report so the write-up batch can cite the rule that was verified. Make no file change in
  this card.
- **Commit:** none

## Batch Tests

`verify: null`. Nothing in this batch has a runnable surface: card 1 writes one markdown file and
card 2 runs two read-only git queries whose output is the test. The repository-wide gate
(`go test ./... && golangci-lint run`, with the live-test guard unset) still runs at the end of the
task and covers this batch as it covers every other; running it per implementer round here would
verify Go code no card in the batch touches. Card 2 is itself the batch's real gate — it proves the
ignore rule holds and the tree is clean, which is the precondition batch 2 is entitled to assume.
