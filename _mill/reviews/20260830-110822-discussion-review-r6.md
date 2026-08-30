MILL_REVIEW_BEGIN
# Review: Rethink quarry-mcp's per-call targetDir ergonomics

```yaml
duration_s: 199.0
verdict: APPROVE
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: /home/knatte/Code/quarry/wts/mcp-target-dir-ergonomics/_mill/discussion.md
date: 2026-08-30
```

## Findings

### [NIT:scope] Per-tool "override cases" to remove do not exist
**Section:** Testing → "Per-tool tests" **Issue:** Every `targetDir`/`TargetDir` hit in `tools_lsp_test.go`, `tools_impact_test.go`, `tools_assert_test.go`, `tools_toc_test.go` is `cfg.TargetDir` or prose; no test sets `in.TargetDir`, so "remove override-specific cases" and "convert an override to `Config.TargetDir`" describe zero work, and the only override-exercising tests are `callcontext_test.go:12-39`. **Fix:** State that `callcontext_test.go` is the sole file with override cases and that the four per-tool files are expected to need no deletions, only the new toc absolute-path case.

### [NIT:consistency] Undispositioned per-call phrasing in a test comment
**Section:** Testing → Verification grep / Scope reword list **Issue:** The grep rule says survivors must "never" carry per-call phrasing, but `tools_toc_test.go:212` ("never the call's targetDir") is per-call phrasing in `internal/mcpserver/` that the Scope reword list (production files only) and the `effectiveTargetDir` disposition (`tools_toc_test.go:370` only) both miss. **Fix:** Add it to the reword list or name it an intentional survivor so the grep rule is not self-contradicting.

## Verdict

APPROVE
Decisions complete and source-accurate; two cosmetic gaps in test-side scope wording.
MILL_REVIEW_END
