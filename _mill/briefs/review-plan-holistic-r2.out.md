MILL_REVIEW_BEGIN
# Review: The glyph-maker: declaration to glyph (P1, roadmap 2b) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic)
reviewed_file: plan/
date: 2026-09-05
```

## Findings

### [BLOCKING:consistency] Batch 5's verify contradicts its own Shared Decision
**Location:** `00-overview.md` "Decision: verify scope is one package per batch" vs `05-docs-inventory.md` batch header.
**Issue:** The decision states "each batch's `verify:` runs `go test` over the single package it touches" and "Applies to: all batches", but batch 5's verify is `go build ./... && go vet ./...` — repo-wide, and no `go test` at all; batch 5 also touches four packages plus two Markdown files, so one-package scoping is impossible for it by construction.
**Fix:** Carve the docs batch out in the decision's own text (a prose-only batch is gated by build+vet, not by a package test run), so the decision and the batch index agree.

### [NIT:consistency] Card 1 step 6 describes a case step 3 makes unreachable
**Location:** `01-engine-maker.md` card 1, steps 3 and 6.
**Issue:** Step 3 has `nameExtract` append `strategy.Symbols(...)` "only when `partial` is false", so a partial parse yields zero symbols; step 6's justification — "even if it happened to yield one surviving symbol" — can therefore never fire, and reads as though symbols are collected on a partial parse.
**Fix:** Either drop step 6 as already-implied by step 3, or state it as the invariant it actually is (a partial retry returns no symbols, so the count check is never reached).

### [NIT:consistency] Card 6's failure-line spelling can emit trailing whitespace
**Location:** `02-facade-and-renderers.md` card 6, `RenderNameText` failure branch.
**Issue:** The card spells the line as `normalizeProse(r.Target)`, then `" error "`, then `r.Reason` — with the reason segment emitted only when non-empty — which leaves a trailing space on an empty-reason value, contradicting the same card's no-trailing-whitespace invariant; `RenderResolveText`'s branch 1 in `quarry/text.go`, which the card says to follow "exactly", writes `" error"` and then `" " + r.Reason` for precisely that reason.
**Fix:** Respell the segments as `" error"` followed by a conditional `" " + r.Reason`, matching the cited branch.

### [NIT:consistency] Batch 4 misstates who consumes `compareGolden`
**Location:** `04-naming-round-trip.md` "Batch Tests".
**Issue:** The section justifies the `-run 'TestRoundTrip'` narrowing by claiming cards 14 and 15 edit "helpers those three tests are the only consumers of", but `compareGolden` in `internal/engine/golden_test.go` is also called by `TestGolden_LoomyardRenderDir` and `TestGolden_LoomyardRenderLayoutFile`, neither of which the `-run` pattern reaches.
**Fix:** Reword to name the two golden tests as further consumers and note that compilation plus their own Loomyard gate is what covers them.

## Verdict

REQUEST_CHANGES
One shared decision is contradicted by batch 5; three smaller self-contradictions.
MILL_REVIEW_END
