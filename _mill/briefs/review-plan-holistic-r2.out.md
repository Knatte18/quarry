MILL_REVIEW_BEGIN
# Review: Thin quarry/ facade over internal/quarryengine — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: sonnetmax
reviewer_self_id: claude-sonnet-5
reviewed_file: plan/
date: 2026-08-27
```

## Findings

### [BLOCKING:scope] `builtins()` is used well beyond load.go/detect.go
**Location:** batch 2 cards 5, 6, 8 (and discussion.md "Identifiers that must become exported" / registry.go bullet) **Issue:** Card 5 claims `builtins` stays unexported because "load.go and detect.go are the only users," but `quarry/ensureserver_integration_test.go` (6 call sites: lines 44, 50, 68, 72, 132, 145, 176) and `quarry/toolchain_integration_test.go` (line 41) and `quarry/supervised_integration_test.go` (line 46) — all moving to `package daemon` per card 6 — plus `quarry/refs_integration_test.go` (lines 71, 93, 188, 207, 245) — moving to `package query` per card 8 — all call the unexported `builtins()`. Neither card's retargeting list mentions this, so these four `-tags lsp` files fail to compile (`undefined: builtins`) at exactly the point batch 2's tagged verify pass would hit them. **Fix:** add `builtins() -> registry.BuiltinRegistry()` to cards 6 and 8's retargeting lists for these four files.

### [BLOCKING:scope] `repoRoot(t)` test helper crosses the query→daemon boundary in the disallowed direction
**Location:** batch 2 cards 6, 8 **Issue:** `func repoRoot(t *testing.T) string` is defined exactly once, in `quarry/refs_integration_test.go:59`, which card 8 moves to `package query`. It is called from `quarry/ensureserver_integration_test.go` (lines 50, 71, 135) and `quarry/supervised_integration_test.go` (line 50), both moved to `package daemon` by card 6. The layering Decision forbids `daemon` importing anything from `query`, so a simple qualified-import fix is impossible, and no card names this seam at all. This is a second, un-enumerated instance of the exact problem the `daemontest` Decision exists to solve, so discussion.md's claim "one test seam does cross a package boundary" is inaccurate. **Fix:** add a card-level decision giving `daemon`'s and `query`'s integration tests a common home for this helper (e.g. add it to `daemontest`, which any `_test.go` file may already import per card 11), and retarget all four call sites.

### [NIT:design] card 4's `documentSymbol`/`DocumentSymbol` "collision" is not a real Go conflict
**Location:** batch 2 card 4 **Issue:** Card 4 asserts the method `documentSymbol` and the type `lspDocumentSymbol` (both slated to become `DocumentSymbol`) collide, and renames the method to `DocumentSymbols` to avoid it. Go method names live in the receiver's method set, not the package identifier scope, so `type DocumentSymbol struct{}` alongside `func (c *Client) DocumentSymbol(...) ([]DocumentSymbol, error)` compiles without conflict. **Fix:** correct the stated rationale; the pluralized name can still be kept as a style choice, so this does not block implementation as written.

### [NIT:consistency] `layering_test.go` has no package-count floor, unlike `seam_enforcement_test.go`
**Location:** batch 2 card 10 vs. batch 3 card 11 **Issue:** discussion.md's "Consequence for the layering guard" states "both guards' expected-package-count assertions include it [daemontest]," and card 10 does add "visited at least six distinct package directories" to `seam_enforcement_test.go`. Card 11's Requirements for `layering_test.go` only keep the zero-files fatal and never ask for an equivalent minimum-package-count assertion. **Fix:** add the same package-count floor to card 11, or correct discussion.md's claim that both guards carry one.

## Verdict

REQUEST_CHANGES
Two BLOCKING scope gaps (`builtins()`, `repoRoot()`) leave four moved integration-test files uncompilable under cards 6/8 as written.
MILL_REVIEW_END
