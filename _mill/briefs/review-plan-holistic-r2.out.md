MILL_REVIEW_BEGIN
# Review: Batch-answer contract: per-target coverage + fail-closed Status helpers — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5
reviewed_file: plan/
date: 2026-09-09
```

## Findings

### [BLOCKING:design] Card 9's cross-reference warning rests on a false premise
**Location:** batch 3 / card 9
**Issue:** The card singles out the paragraph opening `runExpand's own pipeline, continuing from step 4 above` as a cross-reference the renumbering may invalidate; verified against `internal/cli/cli.go`'s `Run` doc comment, that "step 4" (and the identical phrase in the runTOC, runDelta and runResolve paragraph openers at the same comment) refers to `Run`'s own shared step 4, "Resolve the repository root by calling internal/repopath.ResolveRoot" — not to runResolve's inner 1..6 list. Deleting runResolve's step 4 invalidates no cross-reference at all, and naming that sentence invites an implementer to "correct" four correct references to `step 3`.
**Fix:** State the fact instead of the suspicion — that the four `continuing from step 4 above` openers cite `Run`'s shared step 4 and must be left unchanged, and that renumbering runResolve's list 1..6 → 1..5 invalidates no cross-reference in the comment.

### [NIT:consistency] Shared Decision says "one package var"; the plan adds two
**Location:** overview / `### Decision: additive-only, semver-minor` vs batch 1 cards 1 and 3
**Issue:** The decision fixes the new exported surface at "one package var and two methods on existing types", but card 1 adds `engine.Statuses` and card 3 adds `quarry.Statuses` — two package vars across the two packages the plan treats as one API surface.
**Fix:** Reword the decision to "one package var per package (`engine.Statuses`, re-exported as `quarry.Statuses`) and two methods", so the decision and the card inventory agree.

### [NIT:consistency] Card 2 misstates answer_test.go's existing test style
**Location:** batch 1 / card 2
**Issue:** The card says to follow "the file's existing table-test style: a `tests` slice of anonymous structs with a `name` field" and that `TestSubject_Case` is "the file's own shape, which every existing test in it carries". Verified against `internal/engine/answer_test.go`: it contains no `tests := []struct{...}` table at all (its subtests are inline `t.Run("Case", func...)` calls), every existing test's subject token is `Answer` (`TestAnswerJSON_*`, `TestAnswerKnobs_*`), and `TestAnswerExtensionlessFileHeaderRule` and `TestAnswerIgnoreSetFreshness` carry no underscore.
**Fix:** Cite `internal/engine/name_test.go` (already in the card's `Context:`) as the table-test precedent rather than answer_test.go, and drop the claim about answer_test.go's own naming.

### [NIT:consistency] Card 10 puts the `name` verb into a doc that never names it
**Location:** batch 3 / card 10
**Issue:** The card requires the coverage paragraph to state the rule "for both batch verbs" and to add "`name` has no such path and never fails batch-wide" to `docs/glyph.md` §5. Verified against `docs/glyph.md`: the document mentions `toc`, `resolve` and `expand` only — the verb `name` appears nowhere in any section, so §5 would assert a property of an undefined term, and the anchor paragraph the card names (`toc` takes paths; `resolve` takes glyphs) is about those two verbs.
**Fix:** Either have the card require one introductory clause naming what the `name` verb is before the coverage sentence, or scope the §5 addition to `resolve` and leave `name`'s batch contract to its godoc in `internal/engine/name.go`.

## Verdict

REQUEST_CHANGES
One false-premise instruction in card 9; three consistency nits elsewhere.
MILL_REVIEW_END
