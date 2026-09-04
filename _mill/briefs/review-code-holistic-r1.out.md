MILL_REVIEW_BEGIN
# Review: resolve + expand (T4) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: sonnethigh
reviewed_file: plan/ + source
date: 2026-09-04
```

## Findings

### [BLOCKING:design] openQuarryRoot reimplements the existing openModuleRepo helper
**Location:** `internal/engine/resolve_test.go:30-41`
**Issue:** Card 10 adds `openQuarryRoot`, which duplicates `repoModuleRoot`+`openModuleRepo` already
declared in `internal/engine/walk_test.go:17-31` (identical `runtime.Caller(0)` +
`filepath.Dir(filepath.Dir(filepath.Dir(...)))` climb, identical purpose, same `openRepo` call at the
end). Both are visible in the same test binary since `_test.go` files in one package share scope, so
`openQuarryRoot` could simply have called `openModuleRepo(t)`. This is exactly the "global utility
duplication" the review criteria calls out, and the plan (card 10) never checked for the existing
helper before specifying a new one.
**Fix:** Have `openQuarryRoot` delegate to (or be replaced by) the existing `openModuleRepo`; remove
the reimplemented `runtime.Caller` climb.

### [NIT:consistency] "grouping is realised in unitMemo, below" points the wrong direction
**Location:** `internal/engine/resolve.go:429-430`
**Issue:** `symbolsOfUnit`'s doc comment (re-tensed by card 9, item 3) says the grouping "is realised in
unitMemo, below," but `unitMemo`'s declaration is earlier in the file (line ~95), above
`symbolsOfUnit` (line ~424) — the reverse of what the comment claims.
**Fix:** Change "below" to "above" in that sentence.

## Verdict

REQUEST_CHANGES
Duplicated test-root helper (openQuarryRoot vs. openModuleRepo) is the one real defect; everything else checks out.
MILL_REVIEW_END
