MILL_REVIEW_BEGIN
# Review: Resolve self-target path: per-call dirPackage memo (GH #34) — holistic

```yaml
verdict: APPROVE
reviewer_model: sonnethigh
reviewed_file: plan/ + source
date: 2026-10-03
```

## Findings

### [NIT:consistency] Over-long comment lines left by the retarget edits
**Location:** `internal/engine/units.go:4`, `internal/engine/units.go:112-114`, `internal/engine/resolve.go:385-387`
**Issue:** Replacing `dirPackage` references produced lines far past the ~100-column wrap used in the rest of these comments (units.go:4 runs the clause onto one ~150-column line, and resolve.go:386 trails a half-wrapped line).
**Fix:** Re-wrap those paragraphs to the surrounding width.

### [NIT:consistency] `SpansOf` doc comment omits the throwaway record memo
**Location:** `internal/engine/resolve.go:631-633`
**Issue:** The comment says `SpansOf` "calls symbolsOfUnit", but the call at line 653 now also builds `newFileMemo(r, true)`, which the Shared Decisions name as a SpansOf behaviour.
**Fix:** Add one clause saying SpansOf passes a throwaway record memo.

## Verdict

APPROVE
Plan cards 1-7 are realised in source, memo seams match the tests, and no BLOCKING issues were found.
MILL_REVIEW_END
