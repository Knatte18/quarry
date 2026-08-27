MILL_REVIEW_BEGIN
# Review: Improve gopls query precision (build tags + scoping) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus-class model (Anthropic); exact version self-assessment uncertain, presented to me as claude-opus-5
reviewed_file: plan/
date: 2026-08-27
```

## Findings

### [BLOCKING:design] Spike cannot produce the figure card 25 asserts on
**Location:** batch 1 card 4 (with batch 6 card 25) **Issue:** Card 4 requires `references-verified:`/`callers-verified:` be computed "by applying the recorded widening mode's filter by hand against the logged definition and implementation results", but card 3's spike issues `definition`/`implementation`/`references` only at the two query positions and never a per-reference `textDocument/definition` — which is precisely the input `filterVerifiedReferences` consumes, so the filter cannot be applied to the logged output at all. **Fix:** Either extend card 3's spike to also issue and log one `textDocument/definition` per returned reference, or restate card 4's derivation as a hand-reading of the fixture source and drop card 25's "fix the code" instruction on disagreement.

### [BLOCKING:scope] Card 16 is mode-blind about the documentSymbol phase
**Location:** batch 4 card 16 (vs cards 14, 15) **Issue:** Card 15 makes `SupportsDocumentSymbol()` false, a `documentSymbol` error, an empty `documentSymbol` result, or a classification-phase deadline all skip-verification triggers under directional mode; card 16 never instructs the fake server to advertise `documentSymbolProvider` or answer `textDocument/documentSymbol`, so if card 4 records `mode: directional` every drop assertion in card 16 (both interface directions, "matches neither half is dropped") silently becomes a skip-and-keep and fails. **Fix:** Give card 16 the same conditional clause cards 14 and 15 carry — under directional mode the fake must advertise `documentSymbolProvider` and return a hierarchy classifying the implementation locations.

### [NIT:scope] Card 14 Context omits two files its Requirements name
**Location:** batch 4 card 14 **Issue:** Requirements cite `internal/cli/cli.go`'s declaration-exclusion comment and `quarryengine.Position.Character` (declared in `internal/quarryengine/position.go`); neither file is in `Context:` or `Edits:`, though card 15 does list `internal/cli/cli.go`. **Fix:** Add both paths to card 14's `Context:`, or drop the two citations from its Requirements.

### [NIT:consistency] Two lsp-tagged query test files may redeclare repoRoot
**Location:** batch 1 card 3, batch 6 card 24 **Issue:** Both create `//go:build lsp` files in package `query` and say to resolve the fixture root "the same way `refs_integration_test.go`'s own `repoRoot` helper does"; `repoRoot` and `findFuncPosition` already exist in that package under the same build tag, so a literal reading yields a duplicate-declaration build failure. **Fix:** Say "reuse the existing package-level `repoRoot` (and `findFuncPosition`) helper" rather than "the same way … does".

### [NIT:consistency] capabilities struct doc comment left stale by card 1
**Location:** batch 1 card 1 **Issue:** The card correctly repairs the package doc's "No callHierarchy, no implementation" clause but says nothing about `capabilities`' own doc comment in `lspclient.go`, which reads "reports the server's workspace/symbol and documentSymbol support" and becomes incomplete the moment `ImplementationProvider` is added. **Fix:** Add the struct doc comment to the same requirement bullet that adds the field.

## Verdict

REQUEST_CHANGES
Spike cannot produce card 25's asserted figure; card 16 ignores directional mode's documentSymbol phase.
MILL_REVIEW_END
