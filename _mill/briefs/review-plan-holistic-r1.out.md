MILL_REVIEW_BEGIN
# Review: P3 — the glyphs verb: the planner flat index as a frozen toc preset (roadmap 2a) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: plan/
date: 2026-09-05
```

## Findings

### [BLOCKING:scope] runTOC's glyphs branch has no machine-independent test
**Location:** batch 2, card 10 (and cards 12, 14–15)
**Issue:** Card 10 is the only card that changes CLI runtime behaviour and its `Edits:`/`Creates:` name no test file; the sole coverage of that branch is card 12's `TestGlyphsIsByteIdenticalToItsExpansion` and batch 3's five goldens, both gated on `loomyardRepo(t)`, which *skips* when `LADDER_LOOMYARD_REPO` is unset (`internal/cli/loomyard_test.go:55-58`) — and the done gate `go test ./... && golangci-lint run` (overview, `verify-scope-and-the-done-gate`) sets no such variable. On CI and on every machine without a checkout, nothing asserts that `quarry glyphs <t>` renders the glyphs view at all.
**Fix:** Add a card (or extend card 10) with a `writeScratchTree`-based `Run` test in `internal/cli/cli_test.go`'s existing no-checkout pattern (`cli_test.go:41`) covering `glyphs` and `toc --view glyphs` for a file and a directory target in both formats, plus `--view full` rendering byte-identically to a viewless `toc`, which `view-vocabulary` promises and no card asserts.

### [NIT:scope] The glyphs pre-scan's `--root`-with-no-value case is unspecified
**Location:** batch 2, card 9
**Issue:** The pre-scan enumerates every rejection but not `glyphs --root` with no following token; falling through to the target count would emit `glyphs takes exactly one target, got 0` where `toc --root` emits `--root requires a value` (`internal/cli/flags.go:146-149`).
**Fix:** State the disposition in `Requirements:` — reject with the existing `"%s requires a value"` message — and add the row to `TestParseArgs_Glyphs`.

### [NIT:consistency] `TestParseArgs_FourVerbGate` is left stale by the fifth verb
**Location:** batch 2, card 9
**Issue:** `internal/cli/flags_test.go:229-231` names and documents itself as pinning "all four verbs"; card 9 makes the gate five-way and updates only the two `no verb given` rows, contradicting the overview's `doc-comments-carry-the-reasoning` decision that a card updates the comments it makes incomplete.
**Fix:** Name that test's rename and its `glyphs` row in card 9's `Requirements:`.

### [NIT:consistency] Card 18 rests on a roadmap state that is not in the file
**Location:** batch 3, card 18
**Issue:** The card says to leave 2b and 2c "exactly as they are, including their already-merged status"; `docs/roadmap.md` lines 47-73 carry no status marker on either point — 2b is written as future work despite having been merged (`486d416`), so the premise is false and the instruction is unactionable as written.
**Fix:** Drop the status clause and state plainly that only point 2a is removed and the point-2 intro's "three"/"Any order among the three" is adjusted to what remains.

### [NIT:scope] The engine half of `viewless-output-unchanged` is never checked
**Location:** overview `## Shared Decisions`; batch 3, card 15
**Issue:** The discussion's decision and its regression gate cover goldens under `internal/cli/testdata/` **and** `internal/engine/`; the overview's restatement drops the engine, card 15's `git status` gate scopes to `internal/cli/testdata/`, and no batch verify or done-gate invocation runs `./internal/engine/` with the pinned checkout, so those goldens only ever skip. Low materiality — no card edits `internal/engine/` — but the gate the discussion names is never executed.
**Fix:** Either restate the narrowing and why it is safe, or widen card 15's status check and one verify to `./internal/engine/`.

### [NIT:scope] Card 6 confirms the goldens run, not that they pass
**Location:** batch 2, card 6
**Issue:** The card requires `-run TestAfterGoldens -v` to report cases "as run, not skipped" but states no disposition for a failure; these fifteen goldens have never executed on this machine (no `LADDER_LOOMYARD_REPO`), so a pre-existing mismatch at the pin would first surface as a red batch-2 verify and would be misattributed by card 15's rule that any pre-existing golden change is "a defect introduced by this task's product code".
**Fix:** Have card 6 require the fifteen existing cases to pass before any code card lands, and name the pre-existing-failure disposition (stop and report, not `-update`).

### [NIT:decision] The hub's `PYTHONPATH=` verify-prefix rule has no stated disposition
**Location:** overview `## Batch Index` / `verify-scope-and-the-done-gate`
**Issue:** `mill-config.yaml:226-236` states every non-null per-batch `verify:` must start with the literal `PYTHONPATH= ` and names `_plan_validate.verify-not-isolated` as the enforcer; none of the three batch verify commands does. The sibling plan in this hub (`wts/diff-to-symbols/_mill/plan/00-overview.md:162`) carries an explicit Shared Decision "Go verify commands carry no `PYTHONPATH=` prefix" for exactly this; this plan is silent.
**Fix:** Add the same Shared Decision with its rationale so the omission is a recorded disposition rather than an oversight.

## Verdict

REQUEST_CHANGES
The CLI's new render branch is untested on any machine without a Loomyard checkout.
MILL_REVIEW_END
