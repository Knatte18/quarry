MILL_REVIEW_BEGIN
# Review: resolve + expand (T4) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (1M context)
reviewed_file: plan/
date: 2026-09-04
```

## Findings

### [BLOCKING:scope] Marshal coverage claim for ExpandAnswer is false
**Location:** 04-expand card 17 (closing sentence) and 01-answer-types `## Batch Tests`
**Issue:** Both state card 16's `found` marshal plus card 17's `not_found` marshal "cover all six of that type's keys in both their present and their omitted state", but card 16 asserts `members`/`candidates` **absent** and card 17 asserts them **absent** too — neither key's present spelling is ever pinned by a marshal, so a tag typo (`member`, `candidate`) or a wrong `omitempty` on those two keys passes every assertion in the plan.
**Fix:** Either add a marshalled assertion over an answer that carries `members` (card 16's `TestExpand_Struct`) and one that carries `candidates` (card 17's `TestExpand_AmbiguousBuildTags`), or correct both claims to state which two keys stay unpinned and why.

### [BLOCKING:scope] ResolveResult's present-state tags unpinned by the same gap
**Location:** 03-resolve card 11; 01-answer-types `## Batch Tests`
**Issue:** Card 11's only marshal is of a `not_found` result, which asserts `symbols`, `candidates`, `dir`, `error` and `reason` absent; no card marshals a `found`, `multipart`, `ambiguous`, path or error result, so five of `ResolveResult`'s nine tags are never observed in their emitted form. Card 2's own reasoning — `go vet` catches tag syntax and nothing else — applies here identically.
**Fix:** Extend card 11 (or card 13) to marshal one `ambiguous` and one path/error result, or narrow batch 1's "three marshalled assertions, one per new type's shape" claim to say what those assertions do and do not pin.

### [NIT:consistency] Gap 1 is recorded twice in one file
**Location:** 03-resolve cards 7 and 9; `internal/engine/resolve.go:50-55`
**Issue:** `unitDirs`' existing doc comment already records the external-test-unit-versus-real-directory gap almost verbatim ("docs/glyph.md §2 gives the external test unit its pseudo-path without saying what happens when a real directory spells the same string — that is a gap in the identifier contract"); card 7 requires the same gap recorded again at the collision read, and card 9 rewrites that very comment without noting the overlap.
**Fix:** Have card 7 cross-reference `unitDirs`' existing statement instead of restating it, or have card 9 say which of the two copies survives.

### [NIT:consistency] Card 2 miscounts card 9's comment edits
**Location:** 01-answer-types card 2 vs 03-resolve card 9
**Issue:** Card 2 says "card 9 re-tenses three in the resolve file"; card 9 states five clauses across four comments, of which four are re-tensings and one is a substantive rewrite. All five clauses were verified present in `internal/engine/resolve.go`.
**Fix:** Make card 2's cross-reference agree with card 9's own count.

### [NIT:consistency] Symbol.File's "batch 5 adds" clause left undispositioned
**Location:** 01-answer-types card 2, item 3
**Issue:** The quoted sentence at `answer.go:46-48` is "The span lookup **batch 5 adds**, and the later resolve and expand verbs, fill File…"; card 2 only names the "later verb" phrasing, leaving it unstated whether T3's batch reference — which now collides with this plan's own batch 5 — stays.
**Fix:** State explicitly whether the "batch 5 adds" clause is rewritten or preserved.

## Verdict

REQUEST_CHANGES
Two false wire-contract coverage claims leave six JSON tags unpinned by any test.
MILL_REVIEW_END
