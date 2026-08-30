MILL_REVIEW_BEGIN
# Review: Rethink quarry-mcp's per-call targetDir ergonomics

```yaml
duration_s: 221.0
verdict: APPROVE
reviewer_model: opushigh
reviewer_self_id: Anthropic Claude, Opus-class (session reports claude-opus-5)
reviewed_file: /home/knatte/Code/quarry/wts/mcp-target-dir-ergonomics/_mill/discussion.md
date: 2026-08-30
```

## Findings

### [NIT:consistency] Comments naming deleted `effectiveTargetDir` undispositioned
**Demoted-from:** BLOCKING
**Section:** Scope (doc-comment bullet, lines 57-59) vs `delete-effectivetargetdir` **Issue:** Scope declares `translate.go` unchanged because `translate.go:29` names a live parameter, but `translate.go:23` reads "callers only ever pass ResolveLaunchTargetDir's or **effectiveTargetDir's** result" — a reference to a function this task deletes; the same dangling-reference class exists at `tools_toc.go:5`, `tools_toc.go:162`, `tools_toc.go:185`, `callcontext.go:8`, `nativeentry.go:119` ("the effective absolute target directory"), and `tools_toc_test.go:370`, and none appear in Scope or in "Gotcha — stale doc comments". **Fix:** State one disposition for every comment naming `effectiveTargetDir`, and exclude that identifier from the verification grep's "intentional survivors" whitelist, which currently reads as if any Go identifier spelling is acceptable and would wave all of them through.

### [NIT:consistency] Toc input-struct comments mischaracterised
**Section:** Technical context, second enumeration criterion **Issue:** The text says "the two toc comments enumerate their own fields"; `tools_toc.go:22-24` and `tools_toc.go:40-43` actually read "plus the per-call overrides toc_file/toc_dir accepts" — a back-reference of the same shape as `symbolInput`'s, not an enumeration. **Fix:** Correct the characterisation so the mandatory re-read is aimed at the right phrasing.

### [NIT:consistency] `workspaceKey` line citation disagrees with itself
**Section:** Problem vs Technical context **Issue:** Problem cites `internal/cli/paths.go:76-97` for `workspaceKey`; Technical context cites `paths.go:76-86`. The function body is 76-86; 88-99 is `buildTagsSegment`, a separate helper the discussion elsewhere is careful to keep distinct. **Fix:** Use `paths.go:76-86` in both places.

## Verdict

APPROVE
One class of stale comments naming the deleted helper has no stated disposition.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 0._
MILL_REVIEW_END
