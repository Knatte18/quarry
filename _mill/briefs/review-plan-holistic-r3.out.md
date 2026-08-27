MILL_REVIEW_BEGIN
# Review: Improve gopls query precision (build tags + scoping) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: plan/
date: 2026-08-27
```

## Findings

### [BLOCKING:consistency] Card 18 breaks internal/cli's test build for 4 cards
**Location:** batch 5 / card 18 (with card 22)
**Issue:** Card 18 changes `buildOptions`'s signature but omits `internal/cli/cli_test.go` from `Edits:`, leaving the sole call site (`cli_test.go:585`, `buildOptions(registry, "/target", stateDir, "go", query, 5*time.Second)`) to card 22 — so `go test ./internal/cli/` cannot build across cards 18–21, and card 19's explicitly "written first" empty-tag-set back-compat assertion in `paths_test.go` is unrunnable when written.
**Fix:** Add `internal/cli/cli_test.go` to card 18's `Edits:` and move the `buildOptions` call-site repair into card 18, matching card 19, which already repairs its own `resolveStateDir`/`resolveContext` test call sites in `paths_test.go` and `resolve_test.go` in-card.

### [BLOCKING:design] Directional branch fails open on documentSymbol
**Location:** batch 4 / card 15 (with card 14)
**Issue:** If the spike records `mode: directional`, card 15 names only a `documentSymbol` *error* as a skip-verification trigger. It adds no `client.SupportsDocumentSymbol()` gate (unlike `resolvePosition` in `refs.go`, which gates on exactly that and raises `ErrResolverUnsupported`), and says nothing about an empty or non-hierarchical result — either of which makes `isInterfaceDeclaration` false for every implementation location, silently narrowing the match set and *dropping* references, i.e. turning `assert-no-callers` greener on a degraded server.
**Fix:** Extend card 15's skip-verification enumeration so an unadvertised `documentSymbolProvider` and an empty `documentSymbol` result both skip verification entirely and keep every reference, matching the `verification-is-fail-closed-everywhere` Shared Decision's treatment of the `textDocument/implementation` half.

### [NIT:consistency] Card 15's skip list reads as exhaustive but is not
**Location:** batch 4 / card 15
**Issue:** The bullet beginning "Verification is skipped entirely … in each of these cases" enumerates five cases; a sixth (directional-mode `documentSymbol` error) appears two bullets later, and that bullet first says the error "classifies that file's locations as not-interface" before concluding it should "skip verification entirely" — two different behaviours in one sentence.
**Fix:** Fold the `documentSymbol` case into the enumerated skip list and drop the "classifies as not-interface" clause, keeping only the rationale.

### [NIT:consistency] Card 14's empty-match-set test bullet states a false property
**Location:** batch 4 / card 14
**Issue:** The verify_test bullet leads with "an empty match set keeps nothing and drops nothing", which contradicts the `filterVerifiedReferences` predicate defined three bullets earlier (and the bullet's own parenthetical, which correctly says every attempted, non-empty, non-matching reference *would* be dropped).
**Fix:** Delete the leading clause and keep only the parenthetical's assertion, so the test the implementer writes matches the predicate.

### [NIT:scope] `filterUnexpectedCallers` named without internal/cli in Context
**Location:** batch 4 / cards 15 and 16
**Issue:** Both cards name `internal/cli`'s `filterUnexpectedCallers` in `Requirements:` while `internal/cli/cli.go` is in neither card's `Context:` nor `Edits:`; the same is true of `quarryengine.ErrServerTimeoutSentinel` on card 15, whose `errors.go` card 13 lists but card 15 does not.
**Fix:** Either add `internal/cli/cli.go` and `internal/quarryengine/errors.go` to those cards' `Context:`, or keep the citations purely descriptive — both cards already state the depended-on behaviour inline, so no cold-start exploration is strictly implied.

## Verdict

REQUEST_CHANGES
One mid-batch build break and one fail-open hole in the directional verification branch.
MILL_REVIEW_END
