# Review: Rethink quarry-mcp's per-call targetDir ergonomics

```yaml
verdict: APPROVE
reviewer_model: orchestrator
reviewed_file: _mill/discussion.md
date: 2026-08-30
```

## Findings

### [NIT:consistency] workspaceKey/buildTagsSegment mischaracterized as one hash
**Section:** Technical context, "Daemon isolation (the investigation that de-risked this task)"
**Issue:** The doc claims `workspaceKey` is "a SHA-256 over the absolute target path and the normalized build tags." Actual code: `workspaceKey(targetDir string)` (`internal/cli/paths.go:76-86`) hashes only the target path; `ResolveStateDir` (`paths.go:118-140`) appends a separate `buildTagsSegment(buildTags)` path component only when tags are non-empty — two independent hashes composed as nested directories, not one combined digest.
**Suggested fix:** Reword to "a SHA-256 over the absolute target path; build tags contribute a separate path segment appended by `ResolveStateDir` when present." The isolation conclusion itself is unaffected — verified `resolveCall`/`effectiveTargetDir` (`callcontext.go:49-101`), `ResolveTOCPath` (`toc.go:253-258`), and the call-wide `additionalProperties` test (`schema_test.go:106-111`) all match the doc's claims exactly.

### [NIT:decision] No stated ordering against #009's planned deletion of the same file
**Section:** Decisions/bench-ladder-update
**Issue:** This task edits `bench/loomyard-eval/ladder/scripts/ladder_config.py:383`. The active sibling task `#009 port-ladder-bench-to-go` (also spawned, also running now) has already committed, in its own wiki brief, to deleting `bench/loomyard-eval/ladder/scripts/` entirely once its Go port reaches parity. Neither task's discussion says which should merge first, and a modify/delete race between two independently-progressing worktrees is a real (if low-probability, since #007 is far smaller and likely merges first) `mill-merge-in` conflict scenario, not a hypothetical one.
**Suggested fix:** A one-line note ("expected to merge well before #009's final deletion batch; if not, the conflict resolves as delete-wins") would remove the ambiguity for whoever merges second — not required to proceed to planning.

## Verdict

APPROVE
Decisions are complete with rationale and rejected alternatives, scope is well-bounded and verified against source, and both findings are precision notes that don't affect correctness.
