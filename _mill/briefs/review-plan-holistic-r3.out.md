MILL_REVIEW_BEGIN
# Review: Facade + CLI, resolve + expand (T5b) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic); self-reported, not independently verifiable
reviewed_file: plan/
date: 2026-09-04
```

## Findings

### [BLOCKING:consistency] Card 14 fixture makes its own "found" row impossible
**Location:** batch 4 / card 17 numbering — card 14, fixture paragraph vs. case table
**Issue:** The fixture is specified as "one free function" plus "a second file ... declaring the same function name under a different build tag", so the only free function's glyph matches two declarations; `statusForMatches` (internal/engine/resolve.go) then returns `StatusAmbiguous`, yet the table's first row asserts "the resolve verb on a glyph naming the free function → exit 0, status found, one symbol" and the fourth row asserts ambiguous on that same glyph.
**Fix:** Spell two distinct free functions in the fixture — one undeclared elsewhere for the found row, one duplicated under a build tag for the ambiguous row — the same hazard card 11 already names explicitly for its own fixture.

### [BLOCKING:design] RenderResolveText's path branch has no nil-Dir disposition
**Location:** batch 1 / card 5, `RenderResolveText` branch 3
**Issue:** Branch 3 joins `dirBlocks` of "the result's directory answer" on found; `ResolveResult.Dir` is `*DirAnswer`, so an externally constructed `ResolveResult{Status: StatusFound}` with a nil `Dir` dereferences nil — while the same card, for `RenderExpandText`, spells a fall-through branch precisely because "an external caller can hand it a zero value" and the one-newline promise "must hold for every input".
**Fix:** State branch 3's behaviour when `Dir` is nil (line 1 alone, no block) and pin it in the card's text table, so the two exported renderers make the same totality promise.

### [NIT:consistency] Card 5 says three branches, then lists four
**Location:** batch 1 / card 5, `RenderExpandText`
**Issue:** The lead-in reads "`RenderExpandText` has three branches:" and is followed by items 1–4 (not-found, found, ambiguous, otherwise), where item 4 is the deliberately-added fall-through.
**Fix:** Change the count to four.

### [NIT:consistency] Shared Decision over-claims the evidence goldens
**Location:** overview / "the payload's error field is engine text, emitted verbatim"
**Issue:** The decision states "The evidence goldens and the end-to-end tests pin that exact string byte for byte", but card 15's eight new rows carry no outside-repository resolve case, so no golden holds a payload `error` field at all; only card 11's end-to-end bullet pins it, and it says "emitted verbatim" rather than byte-for-byte.
**Fix:** Either drop "The evidence goldens and" from the decision, or add the outside-repo row to card 15's table and to card 17's index.

### [NIT:scope] Card 2's fixture guidance points at an unspellable unit
**Location:** batch 1 / card 2, `quarry/repo_test.go` tests
**Issue:** The card says to use "the same `writeScratchTree` fixture pattern those tests already use", but the existing TOC tests write Go files at the fixture root (`quarry/repo_test.go` `"a.go": "package p\n"`), and a file directly under the root has the empty, unspellable unit — every glyph-based `Resolve`/`Expand` case in this card needs a nested package directory.
**Fix:** Name the nested package directory in the card, as cards 11 and 14 do for their own fixtures.

### [NIT:scope] Card 11 does not say where the shared fixture grows
**Location:** batch 3 / card 11, "extend the fixture the pipeline tests share"
**Issue:** `newPipelineFixture` currently has one file in `pkg/sub`, and `TestRun_FlagPassThrough/symbols-reaches-named-subdirectory` asserts `len(answer.Files) != 1` for `pkg/sub`; adding the new free function and type there breaks an existing assertion the same card requires to keep passing unchanged.
**Fix:** Name a new sibling directory (not `pkg/sub`) as the home of the added package.

## Verdict

REQUEST_CHANGES
Card 14's fixture contradicts its own found row; card 5's path branch omits a nil case.
MILL_REVIEW_END
