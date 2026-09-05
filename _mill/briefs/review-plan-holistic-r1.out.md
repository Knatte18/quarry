MILL_REVIEW_BEGIN
# Review: Kick-start pack bench: pre-resolved glyph spans in the prompt (M4) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5
reviewed_file: plan/
date: 2026-09-05
```

## Findings

### [BLOCKING:consistency] Card 12's pairing assertion cannot hold
**Location:** batch 2 / card 12, subtest `TwoRungsPairAgainstOneControl`
**Issue:** Verified against `summarize.go`'s `buildComparisons`: it emits one `Comparison` per (non-control cell, metric) over `comparisonMetricNames()` — 12 cost metrics plus recall/precision — so a root yields ~12–14 rows per rung, never "two comparison rows"; and the card says to summarise "the same results root", which the preceding subtest builds with two configs (one `control: true`, one `control: false`), i.e. **one** rung, not two.
**Fix:** State the fixture explicitly (three tool-less configs under one letter: one control, two rungs) and assert on the distinct rung ids appearing in `Comparisons` with `Control` equal to the control id, not on a row count.

### [BLOCKING:scope] Card 28 cannot reach the identifiers it must call
**Location:** batch 6 / card 28
**Issue:** `Requirements:` asks for extraction "through the pack block extractor" and for rendering the prompt "card included", but `Context:` lists only data files — `pack.go` (`ExtractPackBlock`, `PackSentinelBegin`/`End`) and `prompt.go` (`LoadCardFile`, the new fourth `RenderPrompt` parameter) are in neither `Context:` nor `Edits:`, and neither is a later card's `Creates:` target.
**Fix:** Add `bench/loomyard-eval/ladder/internal/ladder/pack.go` and `bench/loomyard-eval/ladder/internal/ladder/prompt.go` to card 28's `Context:`.

### [BLOCKING:scope] No check ties the cards' `Uses:` list to `pack_targets`
**Location:** batch 6 / card 28 and batch 7 / card 29
**Issue:** Card 28 adds `TestPreMatrix_KickstartCardsShareOneUsesList` (three cards identical to each other) but nothing asserts that list equals the ladder file's `pack_targets` in order — exactly the invariant card 29's substitution procedure puts at risk, since it hand-edits the glyph list in the ladder file and the `Uses:` list in three cards as separate steps. A drift leaves the treatment card's generated block naming glyphs its own `Uses:` list does not.
**Fix:** Extend the pre-matrix suite with an assertion that each card's `Uses:` entries equal `Ladder.PackTargets` element-for-element in order.

### [NIT:consistency] `RenderPrompt` call-site count is wrong three ways
**Location:** batch 2 / card 8 and its `## Batch Tests`
**Issue:** Card 8 says "the four existing call sites" then lists five files; `## Batch Tests` says "five call sites across three test files and one production file". Grep of the tree finds six existing callers — `run.go:358`, `prompt_test.go:241,269,283` (three in that one file), `gates_test.go:199`, `prematrix_test.go:69`, `live_test.go:96` — seven counting `live_test.go` separately from `prematrix_test.go`, across four test files.
**Fix:** Say "every caller in `run.go`, `prompt_test.go` (three), `gates_test.go`, `prematrix_test.go` and `live_test.go`" and drop the count.

### [NIT:consistency] `go build ./...` is credited with catching test-file breaks
**Location:** batch 2 / `## Batch Tests` (and the same sentence pattern in batch 3)
**Issue:** `go build` does not compile `_test.go` files, so the module-wide `verify:` cannot catch a missed `RenderPrompt` call site in `prompt_test.go`/`gates_test.go`/`prematrix_test.go`/`live_test.go`. The batch's own `go test` is what compiles them (confirmed: `live_test.go` carries no build tag, only an env-guarded skip, so it does compile).
**Fix:** Attribute the compile-break net to the batch's own `go test`, not to `go build ./...`.

### [NIT:consistency] Card 5's "exactly the eleven" does not match its second grep
**Location:** batch 2 / card 5
**Issue:** The `IsControl()` grep yields eleven non-test sites (`mcp.go:47`; `run.go:164,182,368,415,558,664,902,912`; `config.go:131,232`), but the `\.Allowed` grep additionally yields nine non-test sites (`run.go:262,665,666,911`, `gates.go:62`, `summarize.go:257,306`, `config.go:73,226`), none of which card 5 classifies — so the confirmation step as written fails on its own terms.
**Fix:** Say the eleven refers to the `IsControl()` grep only, and that the `.Allowed` grep is read for new sites via the escape clause already stated.

### [NIT:decision] Batch 5's `verify:` departs from its own Shared Decision
**Location:** `00-overview.md` decision `verify-commands-are-go-scoped-to-the-harness-package` vs. batch 5
**Issue:** The decision states "every batch's `verify:` is a `go test` against `./bench/loomyard-eval/ladder/internal/ladder/`"; batch 5's is `go build ./bench/loomyard-eval/ladder/... && go test ...`. The build prefix is justified in batch 5's `## Batch Tests` (it is the only gate on the new `cmd/ladder` subcommand) but the decision does not admit it.
**Fix:** Amend the decision to allow a scoped `go build` prefix for the batch that touches `cmd/ladder`.

### [NIT:scope] Card 17 names a helper from a file it may not read
**Location:** batch 4 / card 17
**Issue:** `Requirements:` says to reuse "the package's existing hex-sha256 helper"; that is `sha256Hex` in `provenance.go`, and card 17's `Context:` is `none` (its only `Edits:` is `pack.go`).
**Fix:** Add `bench/loomyard-eval/ladder/internal/ladder/provenance.go` to card 17's `Context:` and name `sha256Hex` explicitly.

## Verdict

REQUEST_CHANGES
Three blocking gaps: an unsatisfiable assertion, a missing Context pair, and an unchecked invariant.
MILL_REVIEW_END
