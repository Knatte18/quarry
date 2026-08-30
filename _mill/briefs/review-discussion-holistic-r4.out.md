MILL_REVIEW_BEGIN
# Review: Rethink quarry-mcp's per-call targetDir ergonomics

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude, Opus-class (environment reports model ID claude-opus-5); self-assessment only
reviewed_file: /home/knatte/Code/quarry/wts/mcp-target-dir-ergonomics/_mill/discussion.md
date: 2026-08-30
```

## Findings

### [BLOCKING:design] Stale-comment sweep relies on a grep that misses its target
**Section:** Technical context → "Gotcha — stale doc comments" / Testing → Verification
**Issue:** The completeness check is stated as "a grep for `targetDir` and `TargetDir` across `internal/mcpserver/` after the change should surface only intentional survivors", but three input-struct doc comments state the fact by *count* and contain neither string: `tools_lsp.go:23-24`, `tools_impact.go:18-19`, and `tools_assert.go:18-19` all read "the three call-wide resolution overrides every language-server-backed tool in this package accepts", which becomes two — none of the three is named in Scope or in the Gotcha list either. The Verification command is also narrower than the rule it implements: `grep -rn 'targetDir'` is case-sensitive and does not match `TargetDir`.
**Fix:** State a second enumeration criterion that does not depend on the literal token — e.g. every input struct's own doc comment in `internal/mcpserver/` must be re-read — and make the two statements of the grep agree on case.

### [NIT:decision] `serverVersion` has no stated disposition
**Section:** Decisions → `drop-per-call-targetdir`
**Issue:** `serverVersion = "0.1.0"` (`mcpserver.go:20`) is invoked as load-bearing rationale for "no deprecation obligation", but neither Scope's In nor Out says whether a breaking removal from the published tool schema bumps it; `mcpserver.go` is otherwise edited by this task, so a plan writer is left to guess.
**Fix:** Add one line to Out (or In) recording that `serverVersion` is deliberately left at `0.1.0`.

### [NIT:consistency] "Keep a case" describes a test that does not exist
**Section:** Testing → "`tools_toc_test.go` specifically"
**Issue:** It says to *keep* a case proving an absolute `target` resolves outside the launch root, but `tools_toc_test.go` has no such case today — every toc test roots its fixture at `cfg.TargetDir`. A plan writer looking for the existing case may conclude the escape hatch is already covered and skip it.
**Fix:** Reword to "add", so the pinning test for the documented partial hatch is unambiguously new work.

## Verdict

REQUEST_CHANGES
Sound decisions and verified claims; the stale-comment enumeration method provably misses three sites.
MILL_REVIEW_END
