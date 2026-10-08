MILL_REVIEW_BEGIN
# Review: Enclose: path:line ranges -> enclosing member glyphs (batch, at revision, CLI verb) — holistic

```yaml
verdict: APPROVE
reviewer_model: opushigh
reviewed_file: plan/
date: 2026-10-08
```

## Findings

### [NIT:consistency] Verb enumerations in parseArgs and Run docs left stale
**Location:** batch 3 / cards 9 and 10
**Issue:** `parseArgs`'s doc comment in `internal/cli/flags.go` says "The verb gate accepts exactly ... delta and name", "--text is valid for every verb", "--root is valid for the five repository verbs (toc, glyphs, resolve, expand and delta)" and "Every verb requires exactly one target". Card 9 asks only to "state the enclose verb's flags and its many-targets rule", which leaves those existing sentences false. `Run`'s doc in `cli.go` ("calls one of runTOC, runResolve, runExpand, or runDelta") is also outside card 10's "four repository verbs" rewrite.
**Fix:** Name these sentences in cards 9 and 10 for a subsystem-level rewrite, as card 10 already does for the "four repository verbs" phrase.

### [NIT:consistency] Prescribed replacement texts still enumerate callers
**Location:** batch 2 / cards 6 and 7
**Issue:** Card 6 rewrites `internal/repopath/doc.go` to "its callers (the command-line and MCP surfaces and the facade)". That is a caller list, which the discussion's "name the subsystem rather than list every caller" rule forbids. Card 7 removes the renderer tally from the `quarry/render.go` header but keeps that header's alias list ("DirAnswer, ResolveResult, ExpandAnswer and NameResult are aliases"), which `EncloseResult` makes incomplete.
**Fix:** Have card 6's text name its callers as a group with no parenthetical list, and have card 7 rewrite the render.go alias sentence the same way.

### [NIT:scope] golangci-lint constraint is not wired anywhere
**Location:** overview / batch verifies
**Issue:** The discussion's Constraints require `golangci-lint run` to pass, but no batch `verify:` and no card mentions it.
**Fix:** Either state where lint runs (outside the plan), or add a one-line lint step to the final card's requirements.

## Verdict

APPROVE
No blocking issues; the cards match the discussion's rules, sequencing and file references verified against source.
MILL_REVIEW_END
