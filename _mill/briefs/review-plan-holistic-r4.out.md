MILL_REVIEW_BEGIN
# Review: Ladder breadth (M1) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic), reasoning effort high
reviewed_file: plan/
date: 2026-09-04
```

## Findings

### [BLOCKING:decision] Card 10's `--results` example escapes date substitution
**Location:** batch 4 / card 10; overview `results-root-date-substitution`
**Issue:** The decision enumerates every site the `2026-09-04` date must be substituted at — "batch 5's cards 15 and 16, batch 6's cards 17 and 18, and the overview's `## All Files Touched`" — but card 10 also writes a `-breadth` results root into the tracked `ladder-toc.yaml` header ("Update the run command's `--results` example to a `-breadth` root") and is not on that list; card 10 never says whether that example carries a literal date or keeps the existing header's `<date>` placeholder.
**Fix:** State card 10's disposition explicitly (keep the `<date>` placeholder, as the current header uses) or add card 10 to the decision's substitution list.

### [NIT:scope] Card 13 does not say how to resolve `task_file` from the test directory
**Location:** batch 4 / card 13, `TestPreMatrix_ControlPromptsAreBlind`
**Issue:** The card pins the ladder path as `../../ladder-toc.yaml` and pins the fasit paths as "relative to the test's own directory", but for the task file it only says "load its task's `task_file` with `LoadTaskFile`". `ladder-toc.yaml`'s `task_file` values are repository-relative (`bench/loomyard-eval/tasks/…`), which `run.go` resolves via the unexported `resolveRepoRelative(quarryRepoRoot, …)`; passing them straight to `LoadTaskFile` from `internal/ladder/` fails. `prompt_test.go` uses `../../../tasks/…`.
**Fix:** Name the resolution the test uses (e.g. join `../../../../` onto `task.TaskFile`), matching how the card already pins the ladder and fasit paths.

### [NIT:consistency] Task-02 swap contingency omits the blinding re-check
**Location:** batch 2 / card 7 vs batch 3 / card 9
**Issue:** Card 9 states that a ladder-d subject swap "re-opens card 8's blinding constraint — the replacement prompt must again contain none of the bare tokens `quarry` or `toc`". Card 7's task-02 swap contingency changes "the prompt body, the `## Scope` section and the notes section" with no equivalent sentence, yet `c0-none` is a control and `CheckRenderedControlPrompt` is equally fatal for it.
**Fix:** Add the same one-line re-check to card 7's swap paragraph; card 13's gate already backstops it, so this is prose symmetry, not a new mechanism.

## Verdict

REQUEST_CHANGES
One decision gap on card 10's results-root example; the rest is sound.
MILL_REVIEW_END
