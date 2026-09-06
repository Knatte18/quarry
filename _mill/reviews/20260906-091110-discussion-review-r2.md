MILL_REVIEW_BEGIN
# Review: M4 matrix run: execute the descoped kick-start batch (cards 29-32)

```yaml
duration_s: 274.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (1M-context variant)
reviewed_file: /home/knatte/Code/quarry/wts/kickstart-matrix-run/_mill/discussion.md
date: 2026-09-06
```

## Findings

### [BLOCKING:design] Resume path omits lock clearing and re-commit
**Section:** §Decisions / live-matrix-runs-detached (+ commit-before-each-invocation)
**Issue:** "Re-run the identical command" cannot work as written after an abnormal death: `AcquireRunLock` (worktree.go:293-306) is `O_CREATE|O_EXCL` and, as the discussion's own Technical Context states, is never reaped — so the resume fails with "another ladder run holds …"; and by resume time `$RESULTS_ROOT/provenance.json` is tracked-and-modified (card 29 committed it, `Run` rewrites it after every repetition, run.go:224), so `CollectInvocation` records `quarry_dirty: true` for the resumed invocation, defeating the very property `commit-before-each-invocation` exists to buy. No liveness test is specified either, so "the process died" is undetectable from `tail` alone.
**Fix:** Spell the resume preconditions — confirm no live `ladder`/`go run` process, clear `~/.cache/ladder-eval/.ladder.lock` if present, commit the tree clean — and state whether a resumed `quarry_dirty: true` is accepted with a conclusion carve-out or prevented.

### [BLOCKING:design] No disposition for an arm finishing with n < 10
**Section:** §Constraints (no optional stopping) and §What the conclusion must contain
**Issue:** A repetition can end permanently absent — `MaxAttempts` exhaustion returns `incomplete: true` without retry (run.go:574-584), and a blinding-failed repetition "is not counted present for completeness" (summarize.go:232-239) — so an arm can legitimately end with 9 usable repetitions. The predeclared rule fixes n = 10 per arm and critical U <= 27, and forbids adding repetitions; the conclusion spec gives readings for ceiling, gate-failure and unscored counts but none for a short arm.
**Fix:** State the predeclared disposition for a cell appearing in `summary.json`'s `incomplete` list — whether the U test is reported as void, recomputed against the correct critical value for the realised n, or the root is declared unusable — before rep 1.

### [NIT:decision] Fasit's disposition under glyph substitution unstated
**Demoted-from:** BLOCKING
**Section:** §Decisions / glyph-substitution-contingency-is-fully-specified
**Issue:** Steps 1-4 change `pack_targets`, all three `Uses:` lists and e2's `Files:` list; step 5 only records a note and step 6 "re-runs the cross-check" — nothing says whether `07-fabric-merge-state-tracing.fasit.json`'s `relevant_files`, `key_symbols` or `summary` change. All three named reserves live in files already on the seven-file list (`internal/fabricengine/mergelifecycle.go:221,366`, `internal/gitrepo/merge.go:157`), so a substitution can only *vacate* a file — leaving e2's re-derived `Files:` at six against the fasit's seven `relevant_files`, which is exactly what recall/precision score against, and falsifying the conclusion's mandated "e2 names the same seven the fasit does" sentence. `TestPreMatrix` checks neither list.
**Fix:** State explicitly whether the fasit is frozen (and e2's `Files:` therefore re-derived to match `relevant_files` rather than the glyph set) or edited alongside, and how the conclusion's recall-inflation paragraph is reworded if the two diverge.

### [NIT:consistency] "rep 0" versus "rep 1"
**Section:** §Decisions / glyph-substitution, §Testing
**Issue:** "Before rep 0 or not at all" and "pre-rep-0 gate" contradict §Technical Context's own "runs before rep 1", the run loop (`for rep := 1; rep <= repsEffective`), `verifyCardsAndPack`'s doc comment and the ladder file header, all of which index repetitions from 1.
**Fix:** Use "before rep 1" throughout.

### [NIT:consistency] Run host diverges from the roadmap point it deletes
**Section:** §Decisions / commit-before-each-invocation
**Issue:** `docs/roadmap.md` point 1 says the matrix is run "from the hub against main"; the decision rejects the hub, so `provenance.json`'s `quarry_commit` will name this worktree's branch tip rather than a main commit, unlike the reference root (`0ae4daa…`). The divergence is unremarked and the conclusion's provenance-coverage section has no instruction for it.
**Fix:** Note the deviation and say what the conclusion records about the non-main `quarry_commit`.

## Verdict

REQUEST_CHANGES
Three gaps in irreversible, money-spending paths: resume, short arms, fasit under substitution.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 2._
MILL_REVIEW_END
