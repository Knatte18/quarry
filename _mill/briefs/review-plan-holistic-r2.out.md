MILL_REVIEW_BEGIN
# Review: Add an MCP wrapper for quarry — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Anthropic Claude, Opus tier; runtime reports claude-opus-5
reviewed_file: plan/
date: 2026-08-29
```

## Findings

### [BLOCKING:decision] `.mcp.json` top-level shape never specified
**Location:** batch 7 / card 28
**Issue:** Two incompatible `.mcp.json` shapes exist — the plugin-root bare map (`{"name": {...}}`, per `/home/knatte/.claude/plugins/marketplaces/claude-plugins-official/plugins/plugin-dev/skills/mcp-integration/SKILL.md:27-37`) and the project-root `{"mcpServers": {...}}` wrapper — and card 28 names neither, while batch 7's own decision says correctness "is verified by dogfooding, not by a test", so a wrong shape has no detector anywhere in the plan.
**Fix:** State the exact top-level key card 28 must emit (this repo has no `.claude-plugin/`, so the project-root `mcpServers` form applies) and have card 29/30 word the auto-connect claim as "after the one-time project-server trust prompt" rather than "no install step".

### [BLOCKING:scope] Card 9 Context omits `internal/mcpserver/translate.go`
**Location:** batch 2 / card 9
**Issue:** Card 9's `Requirements:` names `toZeroBased` for `referenceFieldsWire`/`symbolFieldsWire`, but `toZeroBased` is created by card 8 in `internal/mcpserver/translate.go`, which appears in neither `Context:` nor `Edits:`; card 10 and card 11 both list intra-batch created files (`mcpserver.go`) in `Context:`, so this is a one-off omission against the plan's own convention.
**Fix:** Add `internal/mcpserver/translate.go` to card 9's `Context:`.

### [BLOCKING:design] Tier-3 stdout-purity mechanism left as an either/or
**Location:** batch 7 / card 27
**Issue:** The card prescribes connecting over `mcp.CommandTransport` and then offers "capture stdout independently of the transport, or drive the process directly over pipes"; the first option is not implementable, because `CommandTransport` owns the child's stdout pipe and the framed reader consumes it — leaving the implementer to discover that at write time on the one assertion the whole tier exists for.
**Fix:** Pick one mechanism in `Requirements:` — a second, separately-spawned child driven over raw pipes for the purity assertion, with the `CommandTransport` session used for handshake/list/call only.

### [NIT:consistency] Batch 3 scope claims three `register*` functions
**Location:** batch 3 / Batch Scope
**Issue:** The scope says batches 4–6 consume "the three `register*` functions now called from `NewServer`", but batch 3 declares two — `registerLSPTools` (card 14, two tools) and `registerSymbolTool` (card 15).
**Fix:** Reword to "the two `register*` functions covering the three LSP-mirrored tools".

### [NIT:scope] `callContext.Timeout` sourcing unstated
**Location:** batch 2 / card 11
**Issue:** `resolveCall`'s `Requirements:` enumerates the derivation of `TargetDir`, `BuildTags`, `Registry`, and `StateDir` but never says `Timeout` is carried from `cfg.Timeout`, even though `options()` must populate `quarry.Options.Timeout` exactly as `buildOptions` does.
**Fix:** Name `cfg.Timeout` as the source of `callContext.Timeout` in card 11's `Requirements:`.

## Verdict

REQUEST_CHANGES
Three blocking gaps: `.mcp.json` shape, card 9 Context, tier-3 stdout mechanism.
MILL_REVIEW_END
