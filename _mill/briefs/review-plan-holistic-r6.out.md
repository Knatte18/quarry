MILL_REVIEW_BEGIN
# Review: Improve gopls query precision (build tags + scoping) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic), accessed as an agent with Read/Grep/Glob/Write
reviewed_file: plan/
date: 2026-08-27
```

## Findings

### [BLOCKING:scope] internal/quarryengine/doc.go is never edited by any card
**Location:** whole plan (`## All Files Touched`; cards 1, 6, 15, 16)
**Issue:** That file's "What this engine deliberately does not do" bullet says "No call hierarchy, no implementation. Only textDocument/references, textDocument/definition, and the workspace/symbol resolver are wired… implementation was never in this engine's rubric" (card 1 wires it); its package-layout bullet says `query` is "References, Definition, and Symbol, the entry points internal/cli calls" (card 15 adds `Callers`, cards 14/15 add `verify.go`/`callers.go`); its typed-error-vocabulary paragraph enumerates every engine error and omits card 6's `ErrBuildTagsUnsupported`; its `daemon/daemontest` bullet enumerates the four exported seams and omits card 16's three `ConnKind` constants. The path appears only as read-only `Context:` on card 6 and is absent from `## All Files Touched`.
**Fix:** Give the affected cards (or one card in batch 7) an `Edits: internal/quarryengine/doc.go` requirement naming those four statements, and add the path to `## All Files Touched`.

### [BLOCKING:consistency] Card 10 under-scopes its daemon/doc.go edit
**Location:** batch 3 / card 10
**Issue:** The requirement is limited to "where it spells out `EnsureServer`'s signature", but the same file describes `ensureNative` as "spawn gopls -remote=auto, a disposable local proxy subprocess; gopls itself dedups and owns the real shared daemon behind it", and describes `ConnKindNative` as "quarry's own disposable -remote=auto proxy subprocess for this one call, not the shared daemon behind it — closing it ends only this session, never gopls's real shared daemon". Card 10's private-spawn branch falsifies all three for a non-empty tag set.
**Fix:** Extend card 10's doc.go requirement to name the native-strategy paragraph and the `ConnKindNative` bullet alongside the signature line.

### [BLOCKING:consistency] query's package doc still claims three entry points and one LSP call
**Location:** batch 4 / cards 13 and 15
**Issue:** `internal/quarryengine/query/refs.go` opens with "Package query implements References, Definition, and Symbol, the engine's public orchestration entry points and the sole packages internal/cli calls into" and "refs.go … defines the shared lookup pipeline (acquireConnection, teardownConnection, lookup) that References wraps and that Definition wraps too — both differ only in which single LSP call they make". Both cards edit that exact file and add a fourth public entry point plus a deliberately multi-call scope, yet neither requires touching the package doc — the same defect card 1 explicitly forbids for `lsp/lspclient.go`.
**Fix:** Add a requirement to card 13 (or 15) to widen that package doc to name `Callers`/`runOnConnection` and retract the single-call claim.

### [BLOCKING:scope] Live tier stays silently skippable for anyone following the README
**Location:** batch 7 / card 26 (and the `gopls-lives-in-the-toolchain-cache-not-on-path` Shared Decision)
**Issue:** README's `## Testing` section says the live tier "Requires a real language-server binary (e.g. `gopls`) on `$PATH`". The Shared Decision establishes gopls is *not* on `$PATH` here and that the pinned v0.23.0 lives in the toolchain cache, and names "silence looks like a pass" as the failure this task exists to remove. Batch 6 adds the two tests that matter most, but the `PATH=…` prefix that makes them run is recorded only in per-batch `verify:` lines, which do not survive the plan; card 26 has no requirement to record it.
**Fix:** Add a requirement to card 26 to update the `## Testing` section with the toolchain-cache `PATH` prefix and the silent-skip consequence.

### [NIT:design] Card 15 does not say whether the pre-verification phases run under SkipVerification
**Location:** batch 4 / card 15 (and card 16's `SkipVerification` test)
**Issue:** The call-order paragraph lists definition → implementation → references unconditionally, while the skip list makes `opts.SkipVerification` a "verification is skipped entirely" trigger; whether `textDocument/implementation` (and, in directional mode, `documentSymbol`) is still issued is left open, which decides what card 16's `--no-verify` fake server must answer.
**Fix:** State explicitly in card 15 which phases are elided when `SkipVerification` is set.

### [NIT:consistency] filterWithin's own doc comment keeps the retracted rationale
**Location:** batch 5 / cards 20 and 21
**Issue:** `filterWithin`'s doc comment in `internal/cli/cli.go` reads "mitigating gopls' interface-method reference conflation across packages" — the framing card 21 retracts from the help text; card 20 names two other stale comments to fix but explicitly says "Do not modify `filterWithin`".
**Fix:** Add that comment to card 20's stale-comment list, scoped to the doc comment only, leaving the function body untouched.

### [NIT:consistency] daemontest's package doc still names only two callers
**Location:** batch 4 / card 16 (and card 24)
**Issue:** `daemontest.go`'s package comment says "Today the only caller outside package daemon is query's refs_test.go and refs_integration_test.go"; card 16 adds `callers_test.go` and card 24 adds `buildtags_lsp_test.go` as callers, and card 16 edits that file without updating the sentence.
**Fix:** Add the sentence to card 16's requirements alongside the `ConnKind` constants.

## Verdict

REQUEST_CHANGES
Design and sequencing are sound; four package-doc/README statements the plan itself falsifies are unowned.
MILL_REVIEW_END
