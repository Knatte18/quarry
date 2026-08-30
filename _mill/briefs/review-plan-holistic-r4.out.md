MILL_REVIEW_BEGIN
# Review: Rethink quarry-mcp's per-call targetDir ergonomics — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Anthropic Claude, Opus-tier reasoning model; harness reports the exact id as claude-opus-5
reviewed_file: plan/
date: 2026-08-30
```

## Findings

### [BLOCKING:scope] Stale `effectiveTargetDir` comment left in a file card 7 edits
**Location:** batch 2 / card 7 (and card 8's check two)
**Issue:** `bench/loomyard-eval/ladder/scripts/ladder_config.py:37-42` states `tocFileHandler`/`tocDirHandler` "call effectiveTargetDir/tocPreflight directly and never resolveCall" — the identical claim card 4 rewords in `tools_toc.go` and card 5 rewords in `tools_toc_test.go` — but card 7 says "Change nothing else in the file" and card 8's `effectiveTargetDir` zero-hit grep is scoped to `internal/mcpserver/`, so nothing catches it.
**Fix:** Disposition that comment in card 7 (reword to "read `cfg.TargetDir`/`tocPreflight` directly"), or widen card 8's check two beyond `internal/mcpserver/` and name the file as the card that owns it.

### [NIT:consistency] Test-side grep whitelist blanket-allows the prose the production pass forbids
**Location:** batch 2 / card 8, check one-b
**Issue:** Check one requires production hits to "never return ... per-call override phrasing", but check one-b lists "prose in test doc comments" as an unconditional intentional survivor, so a stale per-call phrasing in a `_test.go` comment passes the gate silently.
**Fix:** Qualify one-b's prose survivor the same way check one does — prose naming the server's target directory only, never per-call override phrasing.

### [NIT:scope] Card 1 rests on a test function in a file absent from `Context:`
**Location:** batch 1 / card 1
**Issue:** `Requirements:` asserts `TestInputSchemaFor_CallWidePropertySurvives` "pins the call-wide `additionalProperties` behaviour the whole-call rejection depends on", but `internal/mcpserver/schema_test.go` appears in neither `Context:` nor `Edits:`, so the implementer cannot check the premise the new transport test stands on. (Same shape, lower stakes: card 5 cites `cli.ResolveTOCPath` with `internal/cli/toc.go` absent from its `Context:`.)
**Fix:** Add `internal/mcpserver/schema_test.go` to card 1's `Context:` as read-only, keeping the existing do-not-edit instruction.

## Verdict

REQUEST_CHANGES
One undispositioned stale reference to the deleted helper; the rest is sound and source-verified.
MILL_REVIEW_END
