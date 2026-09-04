# Review: Ladder breadth (M1)

```yaml
verdict: APPROVE
reviewer_model: orchestrator
reviewed_file: _mill/discussion.md
date: 2026-09-04
```

## Findings

### [NIT:decision] Smoke run rejected outright, not offered to the operator
**Section:** Testing, "A dry sanity run of one new cell at `--reps 1` before the full matrix is **not** taken"
**Issue:** The rejection is correct under the cost constraint as written (extra runs need the operator's word), but the biggest residual risk in this task is burning 30 real runs on a subtly bad new prompt, and loader tests plus offline e2e cannot catch prompt quality; a reps-1 smoke into a throwaway root outside `results/` is exactly two runs and is available *with* the operator's word.
**Suggested fix:** Record the smoke run as an option the operator may authorise before the matrix (throwaway root, never the measured one), rather than a closed rejection — one sentence in the plan's pre-matrix step.

### [NIT:design] Attempt-number mechanism named as needed, not decided
**Section:** Technical context, "The loop currently has no explicit attempt counter of its own — one is needed for the reason file's `attempt` field, and it must agree with the suffix `InvalidateRep` produces"
**Issue:** The reason file is written *before* `InvalidateRep` produces the suffix (today `attempts` only exists as `InvalidateRep`'s return, run.go:447), so the agreement between the file's `attempt` field and the directory's `.invalid-<n>` suffix has to be guaranteed by a mechanism the discussion does not pick.
**Suggested fix:** Have the plan pin it — e.g. a loop-local counter incremented per rejection, plus an assertion (in the e2e tests already specified) that each file's `attempt` equals its own directory's suffix, which the Testing section in fact promises to assert.

## Verdict

APPROVE
Thorough, code-verified, and scope-faithful; two plan-level details to pin, nothing blocking.
