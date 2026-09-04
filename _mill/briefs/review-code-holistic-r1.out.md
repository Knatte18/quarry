MILL_REVIEW_BEGIN
# Review: Ladder breadth (M1) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: sonnethigh
reviewed_file: plan/ + source
date: 2026-09-04
```

## Findings

### [BLOCKING:consistency] `--cells` run example re-includes the retired ladder-a cells
**Location:** `bench/loomyard-eval/ladder/ladder-toc.yaml:75` (the "Run with:" block's `--cells` line)
**Issue:** Card 10 requires "a `--cells` line naming the six cells this matrix runs," and both batch 5's external-interface statement and card 15's actual invocation name exactly `b0-none,b8-toc-dir,c0-none,c1-toc-dir,d0-none,d1-toc-dir` — ladder a is deliberately excluded ("Ladder a is not re-run; task 01 was measured by T7 ... re-measuring it here buys nothing and costs ten real calls"). `provenance.json`'s one invocation confirms `selected_cells` holds exactly those six. The committed line instead reads `--cells a0-none,a2-toc-dir,b0-none,b8-toc-dir,c0-none,c1-toc-dir,d0-none,d1-toc-dir`, silently re-including the two ladder-a ids.
**Fix:** Trim the example to the six ids actually run, matching card 15's own command and `provenance.json`, so an operator who copies this tracked, documented command does not re-spend ten real `claude -p` calls re-measuring a shape this task explicitly chose not to re-run.

## Verdict

REQUEST_CHANGES
One tracked-file inconsistency (`ladder-toc.yaml`'s run example re-adds ladder a); everything else checked out.
MILL_REVIEW_END
