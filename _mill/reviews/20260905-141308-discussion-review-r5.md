MILL_REVIEW_BEGIN
# Review: The glyph-maker: declaration to glyph (P1, roadmap 2b)

```yaml
duration_s: 324.7
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude (Opus-class, exact version not self-verifiable from inside the run)
reviewed_file: /home/knatte/Code/quarry/wts/glyph-maker/_mill/discussion.md
date: 2026-09-05
```

## Findings

### [BLOCKING:design] compareGolden cannot take the counts triple
**Section:** Testing → round trip, step 5 **Issue:** `compareGolden` is `func compareGolden(t *testing.T, name string, got DirAnswer)` (`internal/engine/golden_test.go:104`) — it marshals a `DirAnswer`, so a three-integer counts value cannot be passed to "the same `compareGolden` mechanism"; the cited lines 113–119 are only its path/update block. **Fix:** State the disposition explicitly — widen `compareGolden`'s parameter to `any` (an edit to an existing helper the additive constraint does not currently cover), or declare a sibling helper sharing the same `testdata/loomyard` path and `-update` flag.

### [BLOCKING:design] Inventory predicate counts surfaces, but this task falsifies uncounted prose
**Section:** Testing → "Docs and the verb-set inventory" **Issue:** The stated predicate is "any statement that *counts or enumerates* a quarry surface". It structurally cannot reach prose this task falsifies without a count: `quarry/doc.go:20-23` says a negative answer "is a payload with a status word" — `NameResult`'s negative payload carries `error`/`reason` and no status — and the discussion disposes only of the `internal/cli/doc.go:22-29` twin, not this one. The predicate also leaves `internal/engine` outside the inventory entirely, though it holds counted sites: `expand.go:137` ("one symbol entry is what all three verbs return for a symbol"), `expand.go:145` and `answer.go:218-219` ("docs/rewrite-plan.md's three-queries section"). **Fix:** Widen the predicate to "any statement a new verb / new envelope shape falsifies, counted or not", name `internal/engine` as an inventory site, and give the four sites above a disposition.

### [NIT:consistency] Two rationale citations do not match the cited source
**Section:** Decisions → "CLI verb name is `name`"; "No MCP tool" **Issue:** `glyph/` exports no `Compose`; C1's self-form constructor is `glyph.Self` (`glyph/self.go:14`), so "`compose` is already taken" is not literally true. And "fixed by ... `docs/rewrite-plan.md` §9" is wrong — §9 is Non-goals and says nothing about tools; the "only tools a ladder cell measures" rule is in §7. **Fix:** Cite `glyph.Self` as the self-form API by name, and repoint the MCP rationale at §7.

### [NIT:design] The package-level-facade precedent does not cover a `Repo` method
**Section:** Decisions → "The facade entry point is package-level" **Issue:** The alias-types-carry-no-methods rule is stated in `quarry/render.go:3-6`, not `quarry/quarry.go`, and its reason is that Go forbids methods on aliases to engine types — which does not bear on `(*Repo).Name`, since `Repo` is locally defined. The decision's primary rationale (no I/O, no repository dependency) stands on its own. **Fix:** Drop the precedent clause or re-cite it as `render.go`'s rule about aliases only.

## Verdict

REQUEST_CHANGES
Two blocking items: the counts-golden helper mismatch and the still-too-narrow docs inventory predicate.
MILL_REVIEW_END
