MILL_REVIEW_BEGIN
# Review: The glyph-maker: declaration to glyph (P1, roadmap 2b)

```yaml
duration_s: 224.6
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: _mill/discussion.md
date: 2026-09-05
```

## Findings

### [NIT:scope] Inventory misses the `quarry` package's own counts
**Demoted-from:** BLOCKING
**Section:** Testing → "Docs and the verb-set inventory" **Issue:** The inventory claims "Every site found has a stated disposition", but omits `quarry/doc.go:7` ("three query methods, not one: TOC, Resolve and Expand"), `quarry/doc.go:11,13,14` ("seven renderers", "the three text renderers", "The three JSON success renderers ... share one encoder configuration"), and `quarry/render.go:2` ("the three successful envelopes") — five statements this task falsifies by adding `Name`, `RenderNameJSON` and `RenderNameText`. **Fix:** The search predicate is "statement of the verb set or its count", which structurally cannot find renderer-count or facade-method-count statements; state a predicate that also covers facade surface counts and give each of these five sites a disposition.

### [BLOCKING:design] No-root CLI test rests on a false premise
**Section:** Testing → `internal/cli` verb wiring, third bullet **Issue:** "Use the existing scratch-tree helper for a directory outside any repository if one is reachable" is not achievable — `writeScratchTree` (`internal/cli/scratchtree_test.go:39`) builds trees under the module root's `.scratch/`, i.e. inside this repository, and `Run` resolves the root from `os.Getwd()` (`cli.go:271`), not from any path argument; `cli.go:150-152` states the no-root case is unreachable without changing the process working directory, "which these tests never do". **Fix:** Drop the unreachable first branch and commit to one stated way of asserting that `name` dispatches before root resolution — the only change to existing behaviour is otherwise left with an undecided test strategy.

### [NIT:consistency] `internal` reason routes to exit 1, not exit 3
**Section:** Decisions → "Exit codes" / "The reason vocabulary" **Issue:** `codeForNameResult` maps every non-empty `Error` to `exitNegative`, so an entry carrying the `internal` reason (unwired grammar, nil tree) renders as a negative answer with exit 1, contradicting `usage.go:37`'s "3 internal error". Unreachable today, but the branch is deliberately spelled and then routed inconsistently. **Fix:** State whether `internal` is a per-entry negative answer (exit 1) or the internal-error envelope (exit 3).

### [NIT:decision] `after_test.go:4` has no actual disposition
**Section:** Testing → inventory, "Tests that pin the counts" **Issue:** That table pins the frozen `docs/research/output-formats/after/` goldens, which the discussion says twice must not be added to; with no `name` row added, "The table spans three verbs" stays true and the comment neither fails loudly nor goes stale, so its presence in a "fail loudly" list is misleading. **Fix:** Move it to "Not touched", or state the comment-only edit explicitly.

### [NIT:design] Round-trip walk reuse and runtime budget unstated
**Section:** Testing → "the round trip (the done-when gate)", step 1 **Issue:** Whether `TestRoundTrip_LoomyardNaming` calls `assertSymbolRoundTrip` (re-running the whole span round trip, one parse pass per unit) or performs its own `TOC` walk is unstated, and no runtime expectation is given — `assertSymbolRoundTrip`'s doc comment (`roundtrip_test.go:100-104`) names go test's default timeout as a live constraint at Loomyard scale, and the maker adds one or two fresh `WithTree` calls (each constructing a parser and a language) per harvested symbol. **Fix:** Say which walk the naming round trip consumes and acknowledge the timeout constraint.

## Verdict

REQUEST_CHANGES
Facade-package count statements unenumerated; the no-root CLI test rests on an unreachable premise.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 1._
MILL_REVIEW_END
