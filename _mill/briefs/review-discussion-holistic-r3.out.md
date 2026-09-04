MILL_REVIEW_BEGIN
# Review: Ladder, toc rerun (T7)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic), high reasoning effort
reviewed_file: _mill/discussion.md
date: 2026-09-04
```

## Findings

### [BLOCKING:design] MaxAttempts is not a stop condition across invocations
**Section:** `resume-not-restart`, Scope bullet 3
**Issue:** The stated bound — "repeat until 5 complete or `MaxAttempts` (3) is reached" — does not hold under resume: `InvalidateRep` counts `.invalid-<k>` directories already on disk, so a resumed invocation makes attempt 4, gets `attempts >= MaxAttempts` immediately (`run.go` ~L419-425), records the cell incomplete, and the next resume spends another measured `claude -p` call — indefinitely. `summary.json` and `run`'s exit code look identical whether a repetition was never attempted or is attempt-exhausted, so the polled loop cannot detect the ceiling.
**Fix:** State an explicit ceiling on resume invocations (as `probe-is-reported-done`/the blinding rule already do for gate-2 failures) and name the observable the driver checks to stop — this is the same class of defeated-detector the discussion caught for `WarnOnServerHashDrift`.

### [BLOCKING:decision] Scorer-failed and max-turns repetitions have no disposition
**Section:** `resume-not-restart`, Testing → scenarios list
**Issue:** The discussion names run.go's five-outcome taxonomy but gives "scorer failure" no disposition. `dispatchScorer` exhausting `MaxAttempts` writes the repetition `complete`, `scored=false`, `skip_reason="scorer_failed"` — so `RepIsComplete` is true, resume never retries it, and `summarizeCell` drops it from `recall`/`precision` only (`UnscoredCount`). Same for a max-turns repetition (`MaxTurnsCount`). A cell can therefore report turns/cache_read at n=5 and recall at n=2, and the conclusion's must-cover list mentions neither counter nor `recall_n`.
**Fix:** State the disposition of a scorer-failed and a max-turns repetition (retry, accept, or restart), and add divergent cost-N vs correctness-N — `recall_n`, `MAX_TURNS`/`UNSCORED` flags — to the scenarios the conclusion must report, since "recall unchanged" is the headline claim.

### [BLOCKING:design] Same-day harness-fix restart collides with the fixed root name
**Section:** `harness-fixes-restart-the-matrix` vs `results-root-name`
**Issue:** `results-root-name` pins `results/2026-09-04-toc` and only re-dates for a *later* first run; a harness fix landing the same day yields the identical path. Re-invoking against it is not a restart: `ReadProvenance` returns the existing record, `MergeProvenance` appends the invocation, and every pre-fix repetition satisfies `RepIsComplete` and is skipped — silently mixing two measurement regimes, the exact outcome the decision forbids.
**Fix:** Name the fresh root's naming rule for a same-day restart (e.g. a suffix) and state that the abandoned root's directory is never re-invoked against.

### [NIT:design] `TestLive` is a full measured repetition, not cheap insurance
**Section:** `probe-is-reported-done` ("Kept from the earlier draft"), Testing bullet 1
**Issue:** `TestLive_FreshWorktreeGrantsExactlyBuiltins` calls `invokeMeasuredProcess` with the file's own `run_model`, `run_effort` and `--max-turns 60` — one full `a0-none` repetition's API spend and wall-clock, not a smoke test's. The stated command also carries no `-timeout`, so `go test`'s 10-minute default applies to a run the discussion itself budgets at ~60 s–several minutes.
**Fix:** Describe its real cost (one measured repetition) and state a `-timeout` value, or say explicitly that the default suffices.

### [NIT:consistency] `probe.md`'s cited source of record dies at merge
**Section:** `probe-is-reported-done`
**Issue:** `probe.md` is required to attribute the probe to `_mill/reviews/20260904-110823-discussion-review-r1.md`, but `HANDOFF.md` §0 records that each task merges as one squash commit with `_mill/` retained only under `archive/<slug>` — the same cleaning the discussion cites for T6. The committed evidence file would carry a path unresolvable on `main`.
**Fix:** Have `probe.md` cite the `archive/ladder-toc-rerun` tag alongside the `_mill/` path.

### [NIT:consistency] "worktree root" names two different directories
**Section:** `matrix-runs-backgrounded` vs `clean-tree-before-the-matrix`, Technical context
**Issue:** `cd <worktree-root>` means the quarry git worktree (`.../wts/ladder-toc-rerun`), while `<worktree-root>/bin/quarry` means the harness's `ResolveWorktreeRoot` output (`~/.cache/ladder-eval`). The two are disjoint by construction — `ResolveWorktreeRoot` rejects any path under or containing `quarry`.
**Fix:** Use distinct terms (e.g. "quarry repo root" vs "ladder worktree root") throughout.

## Verdict

REQUEST_CHANGES
Resume bound, unscored-repetition disposition, and same-day restart naming are unresolved.
MILL_REVIEW_END
