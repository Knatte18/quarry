MILL_REVIEW_BEGIN
# Review: Port the capability-ladder bench harness to Go — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 4.x-class model (Anthropic); exact build not self-verifiable
reviewed_file: plan/
date: 2026-08-30
```

## Findings

### [BLOCKING:scope] test_run_ladder.py survives the Python deletion
**Location:** batch 14 / card 73 **Issue:** `Deletes:` lists 12 files but omits `bench/loomyard-eval/ladder/tests/test_run_ladder.py`, which exists on disk and is cited as `Context:` by cards 25, 35–37, 41, 63; card 73's own prose says "leaving both directories empty", and card 72 drops the grandfather clause in the same batch, so a live Python file importing deleted modules is left in a Go-only tree. **Fix:** add it to card 73's `Deletes:` list.

### [BLOCKING:decision] ingest never enforces the pin set
**Location:** batch 12 / card 58 **Issue:** the discussion names `ingest` among the seven subcommands that call the full `require_pins`; cards 54, 55, 60, 61, 64, 65 each state "enforces the full pin set" and card 58 states nothing — yet `ingest` runs `GateMaxTurns` against `max_turns`, the value card 4 deliberately blanks to `null`. **Fix:** state in card 58's `Requirements:` that `ingest` enforces the full pin set before running the gates.

### [BLOCKING:design] transcript gates ported as single-finding returns
**Location:** batch 3 / cards 14, 15; batch 4 / card 21; batch 5 / card 22 **Issue:** cards declare `GateDeniedToolsNotUsed(...) GateFinding`, `GateNoTargetOverride(...) GateFinding`, `GateModelPinned(...) GateFinding`, `GateColdBefore(...) GateFinding`, `GateWorktreeNeutralised(...) GateFinding`, but every Python counterpart returns a *list* — `gate_denied_tools_not_used` emits one finding per offending call and `gate_no_target_override` one per offending key, so a singular return silently drops violations, contradicting "keeping their Python fatality and message shape"; for the 0-or-1 gates no zero-value/no-finding convention is defined. **Fix:** declare these as `[]GateFinding` (or define the empty-finding sentinel explicitly in card 13).

### [BLOCKING:scope] session-input paths and skill roots never named
**Location:** batch 10 / cards 46, 48, 49, 50 **Issue:** the cards say "the settings document", "the run agent definition", "the MCP server declaration", "the user-scope skills root and the plugin-cache root" without ever fixing the load-bearing literals the discussion pins (`.mcp.json`, `.claude/settings.json`, `.claude/agents/<config-id>.md`, `.claude/agents/scorer.md`, `~/.claude/skills/*/SKILL.md`, `~/.claude/plugins/cache/*/*/*/skills/*/SKILL.md`), and `_mill/discussion.md` is in `Context:` only for cards 68 and 70 — so the implementer must invent paths that Claude Code's own discovery depends on, and card 48's "contains exactly the definition and the settings document and nothing else" test is unwritable. **Fix:** name each path literal in the cards' `Requirements:`, or add `_mill/discussion.md` to those cards' `Context:`.

### [BLOCKING:decision] the launch flag combination has no stated disposition
**Location:** batch 10 / card 50; batch 11 / card 52 **Issue:** `LaunchCommand(inputs SessionInputs) string` must return "the exact command line the operator runs", but the plan's own no-smoke-launch decision defers the `--setting-sources` combination and the agent-definition-location question to the follow-up task while no card says which fallback ships — yet card 70 is told to "name the documented fallback for each", naming something no implementation card decided. **Fix:** have card 50/52 state the shipped flag set and whether definitions go to the scratch dir's `.claude/agents/` or the `~/.claude/agents/ladder-<config-id>` fallback.

### [BLOCKING:scope] result-tree artifact names left to inference
**Location:** batch 5 / card 26; batch 12 / card 62 **Issue:** card 26 says `GateRunCompleteArtifacts` is "updated to the artifact set the new results layout defines" without enumerating it — the Python set is `answer.json`/`answer.redacted.json`/`usage.json`/`score.json` and the new layout adds transcript, metadata, launch inputs and `ingest.json`, none of it fixed anywhere in the plan; card 62 likewise describes the probe record without naming `probe.json` or its `allowlist_blocks`/`denylist_blocks`/`denial_shape_observed` keys. **Fix:** enumerate the required artifact filenames in card 26 and the probe record's filename and keys in card 62.

### [NIT:consistency] two types named Usage in one package
**Location:** batch 2 / cards 8, 10 **Issue:** card 8 gives `Record.Message` a `Usage` member with the four token classes and card 10 defines a package-level `Usage` struct for the `usage.json` shape; if card 8's member is a named type the two collide. **Fix:** name the transcript-side type distinctly (e.g. `MessageUsage`) in card 8.

### [NIT:scope] the usage `transcript` field falls out of the partition
**Location:** batch 2 / cards 10, 11 **Issue:** the discussion's partition is explicitly exhaustive and lists `transcript` under "survives unchanged", but it appears in neither card 10's survives list, card 11's changed/added/renamed lists, nor card 11's dropped list — only as the unused-looking `transcriptPath` parameter. **Fix:** name `Transcript` in card 10's or card 11's field enumeration.

## Verdict

REQUEST_CHANGES
Six blocking gaps: a stranded Python file, an unenforced pin check, gate return shapes, and three unnamed artifact/path sets.
MILL_REVIEW_END
