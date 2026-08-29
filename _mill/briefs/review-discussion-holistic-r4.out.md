MILL_REVIEW_BEGIN
# Review: Per-capability quarry-mcp benchmark suite

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic)
reviewed_file: _mill/discussion.md
date: 2026-08-29
```

## Findings

### [BLOCKING:design] Control and rungs differ in preamble, not just tools
**Section:** Technical context ("Reusable assets from #006") + "Reporting discipline"
**Issue:** `none` uses the B preamble (verified at `bench/loomyard-eval/README.md:168-205`: standard tools, no steering), while every quarry rung gets a generated A preamble carrying "prefer quarry over grep, do not re-verify a quarry answer with grep" — so every rung-vs-`none` delta the disjoint-range rule certifies mixes capability with prompt steering, and `bash_grep_count`/`grep_tool_count` are directly suppressed by an instruction the control never receives.
**Fix:** State the disposition — either give `none` an equivalent-length, tool-neutral preamble, or record explicitly that rung-vs-control deltas and all grep counts are confounded by preamble and may not be attributed to the capability.

### [BLOCKING:design] Cleanliness gate discards task 04's own reference method
**Section:** Technical context ("The shared worktree is writable")
**Issue:** Task 04's fasit reaches its answer by a compiler experiment (`results/2026-08-28/04-.../c.json` lines 18/23: edit the interface, `go build -gcflags=-e`), and the task text (`tasks/04-...md:83-96`) frames the run as "you are about to change `Shuttle.Run`"; every run has Bash, so a run that does this dirties the shared worktree and is invalidated — systematically discarding exactly the runs a low-capability rung is most likely to produce, which biases the Ladder B efficiency comparison the ladder exists to make.
**Fix:** Decide whether a compiler experiment is a legitimate scored route (restore the worktree, keep the run) or a disqualifying action, and state which.

### [BLOCKING:design] Invalidate-and-rerun has no bound or abort rule
**Section:** "Runs dispatch sequentially" + "Protocol validation gates"
**Issue:** A failed gate is "re-run, never reported", but several gates can fail deterministically — `permissions.deny` not actually blocking `mcp__quarry__*`, every task-04 run dirtying the tree, a model mismatch — producing unbounded paid re-runs of one cell; only the cold cell has an escape ("reported as not-run"). The pre-matrix probe is also scoped only to hidden-vs-advertised, never to whether denial blocks at all.
**Fix:** Add a per-cell attempt cap with a stated terminal disposition, and extend the probe to assert that a denied `mcp__quarry__*` call actually fails.

### [BLOCKING:design] Scoring calls are unpinned while runs are pinned
**Section:** "Correctness scoring against the existing fasit" + Layout (`score_run.py`)
**Issue:** 45 blinded scoring calls decide every recall/precision/`decoy_admitted` number, yet no model, effort, or prompt pinning is stated for them — while the same discussion argues at length that "a matrix spanning two models is not a matrix" for the runs; a scorer that drifts across the matrix shifts scores between rungs graded at different times, which is the same defect one axis over.
**Fix:** Pin the scorer's model and prompt in `ladder.yaml` alongside the run model, and record it in every `score.json`.

### [NIT:scope] Tests and fixtures have no home in the declared layout
**Section:** "Layout under `bench/loomyard-eval/ladder/`" vs "Testing"
**Issue:** Testing requires committed fixture transcripts and four TDD units, but the layout tree — declared as the file inventory — has no tests or fixtures directory, and the repo carries no Python test infrastructure (only `bench/loomyard-eval/scripts/gen_compact_toc.py`; no conftest/pyproject/pytest.ini/Makefile).
**Fix:** Name the test directory, the fixture location and its tracking status, and the runner.

### [NIT:design] Scorer input set excludes the task text
**Section:** "Correctness scoring against the existing fasit"
**Issue:** The scorer is specified as seeing "only that run's answer and the fasit", but Ladder A's rule requires judging whether `summary` describes the same mechanism — a judgement that needs the task text, which is not config-revealing.
**Fix:** State whether the task text is part of the scorer's input.

### [NIT:design] Transcript acquisition mechanism unstated
**Section:** "Metrics — full benchmark accounting"
**Issue:** Every number is extracted from `transcript.jsonl`, but how the harness obtains it (streamed output vs. locating the session file by `session_id`) is never stated, and that choice determines which fields exist and hence the shape of `extract_usage.py`'s fixtures.
**Fix:** Name the capture mechanism in the metrics decision.

## Verdict

REQUEST_CHANGES
Four unresolved design decisions: control confound, worktree gate bias, unbounded reruns, unpinned scorer.
MILL_REVIEW_END
