MILL_REVIEW_BEGIN
# Review: Rethink quarry-mcp's per-call targetDir ergonomics — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: plan/
date: 2026-08-30
```

## Findings

### [BLOCKING:design] Check two's repo-wide grep can never return zero hits
**Location:** batch 2, card 8 (check two) and `## Batch Tests` item 3
**Issue:** `grep -rn 'effectiveTargetDir' .` from the repo root also matches the planning artefacts in the same worktree — `_mill/plan/01-mcpserver-targetdir-removal.md` (10 hits), `_mill/plan/02-...md` (5), `_mill/plan/00-overview.md` (2), `_mill/discussion.md` (13), plus `_mill/reviews/` and `_mill/briefs/` — so "zero hits, no exceptions" is false on arrival even after every card lands.
**Fix:** Scope check two to the code/doc/bench surface (e.g. exclude `_mill/`) and state the excluded path explicitly, so the check's stated expectation is achievable.

### [BLOCKING:scope] Card 2 names cli.ResolveStateDir/ResolveBuildTags without listing paths.go
**Location:** batch 1, card 2 (`Context:` / `Requirements:`)
**Issue:** The new test `TestResolveCall_TargetDirIsAlwaysConfigTargetDir` is specified as asserting `cli.ResolveStateDir(cfg.StateDir, cfg.TargetDir, cli.ResolveBuildTags(""))`, and the retained ordering names `cli.ResolveConfigPath` too, but `internal/cli/paths.go` (where `ResolveStateDir`/`workspaceKey` live) is in neither `Context:` nor `Edits:` — card 6 does list it.
**Fix:** Add `internal/cli/paths.go` to card 2's `Context:`.

### [NIT:consistency] Card 6's doc content can trip card 8's check one
**Location:** batch 2, card 6 vs card 8 (check one, `docs/mcp-setup.md` arm)
**Issue:** Check one greps `docs/mcp-setup.md` case-insensitively for `targetdir` and forbids "a schema property name", yet card 6 requires stating that no tool accepts a target directory as an input property — the natural phrasing ("no `targetDir` property") would fail the very check the same batch runs, and no card forbids the literal token in the doc.
**Fix:** Have card 6 state that the doc names the concept in prose only and never spells the removed property token, or have check one whitelist a single explanatory mention.

## Verdict

REQUEST_CHANGES
Check two is unsatisfiable as written; card 2's Context omits internal/cli/paths.go.
MILL_REVIEW_END
