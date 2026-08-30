MILL_REVIEW_BEGIN
# Review: Rethink quarry-mcp's per-call targetDir ergonomics — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 4.x-class model (Anthropic); exact build not self-verifiable
reviewed_file: plan/
date: 2026-08-30
```

## Findings

### [BLOCKING:scope] Test-helper source files missing from Context

**Location:** batch 1, cards 1 and 5
**Issue:** Card 1's `Requirements:` names `withStubbedFacade` (defined `internal/mcpserver/tools_lsp_test.go:47`) and `definitionFn` (`internal/mcpserver/facade.go:14`), and card 5 names `tocFileFn` (`facade.go:33`) and `statusFound` (`internal/mcpserver/result.go:18`) — none of those three files appears in either card's `Context:` or `Edits:`, so the implementer cannot read the seam-variable declarations or the status constant it must assert against.
**Fix:** Add `internal/mcpserver/tools_lsp_test.go` and `internal/mcpserver/facade.go` to card 1's `Context:`, and `internal/mcpserver/facade.go` and `internal/mcpserver/result.go` to card 5's — card 5 already lists `tools_lsp_test.go` for the same reason, so the omission is inconsistent within the batch.

### [NIT:consistency] Card 8's provenance sentence omits card 2

**Location:** batch 2, card 8 (final paragraph of `Requirements:`)
**Issue:** "All four checks are expected to pass on arrival, because cards 3, 4, 5, and 6 already did the work they verify" omits card 2, which is the only card that deletes the six `TargetDir` fields, their `jsonschema` tags, and `effectiveTargetDir` — i.e. what makes checks one, one-b, and two pass at all.
**Fix:** Name card 2 alongside 3, 4, 5, and 6 in that sentence.

### [NIT:consistency] New resolveCall test duplicates an existing assertion

**Location:** batch 1, card 2 (`TestResolveCall_TargetDirIsAlwaysConfigTargetDir`)
**Issue:** The card's prescribed `StateDir` assertion is byte-for-byte the assertion `TestResolveCall_StateDirMatchesCLIResolution` already makes (`callcontext_test.go:52-58`), which the same card retains; only the `TargetDir == cfg.TargetDir` half is new.
**Fix:** State in the card that the `StateDir` half is deliberately restated as a paired invariant, or drop it and keep the new test to the `TargetDir` claim alone.

## Verdict

REQUEST_CHANGES
Plan is sound and well-grounded; two cards' Context lists omit files their Requirements name.
MILL_REVIEW_END
