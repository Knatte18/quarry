MILL_REVIEW_BEGIN
# Review: Resolve self-target path: per-call dirPackage memo (GH #34)

```yaml
verdict: APPROVE
reviewer_model: opushigh
reviewed_file: _mill/discussion.md
date: 2026-10-03
```

## Findings

### [NIT:design] "Total equals files touched" is true by construction
**Section:** Decisions › Test seam; Testing › TDD candidate
**Issue:** "Files touched" is defined as the paths the memo saw requested, so total builds equal it by construction — the vacuous shape `TestResolve_ParsesEachUnitOnce`'s own comment warns against.
**Fix:** Say the test lists the expected touched paths from the fixture (the files in the touched directories), which also pins the no-eager-parse rule.

### [NIT:consistency] Whole-repo rejection cites the wrong repository
**Section:** Decisions › Lazy, per-directory, single-parse extraction › Rejected
**Issue:** The 616 ms / 469-file figure is quarry's, but `TestResolve_TwentyGlyphsUnder150ms` runs against the pinned Loomyard checkout (`loomyard_timing_test.go`).
**Fix:** Ground the "breaks the 150 ms test" claim in Loomyard's size, or drop the test reference.

### [NIT:design] "When that call wants them" is per-call or per-file?
**Section:** Decisions › Threading through TOC › Symbol extraction cost
**Issue:** `fileTargetAnswer` wants symbols only for the target file, so an exported symbols-on file-target TOC would extract symbols for every sibling if this is read per call.
**Fix:** State that a TOC-built record extracts symbols only when its own file wants them.

### [NIT:consistency] "Parse count" vs "record build" and "unchanged" vs comment edits
**Section:** Scope; Testing › Gitignored explicit target; Testing › Existing suites unchanged
**Issue:** Scope and the gitignored test still say "parse counter/count" after the Test seam decision redefined the unit as record builds. "Unchanged" also lists `TestResolve_ParsesEachUnitOnce`, whose comment Scope rewrites and whose field it may rename.
**Fix:** Use "record build" throughout, and phrase the existing-suites line as "still pass".

## Verdict

APPROVE
Source claims hold. The remaining findings are non-blocking test-precision and wording issues.
MILL_REVIEW_END
