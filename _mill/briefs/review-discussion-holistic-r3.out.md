MILL_REVIEW_BEGIN
# Review: MCP, thin (T6)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic)
reviewed_file: /home/knatte/Code/quarry/wts/mcp-thin/_mill/discussion.md
date: 2026-09-04
```

## Findings

### [BLOCKING:consistency] Q&A log repeats the rationale D7 forbids
**Section:** Q&A log, "Success payload — text only, or text plus `structuredContent`?"
**Issue:** Its **Why** reads "`structuredContent` would duplicate the payload into every transcript and corrupt the token metric T7 exists to produce" — the exact argument D7's "Not the rationale" paragraph refutes against `docs/research/mcp-surface.md` ("It does not cost context", verified line 103) and says "must not be repeated"; the later `[r2 gap]` entry confirms the reversal but the superseded entry stands unedited three lines above it.
**Fix:** Restate that entry's **Why** as payload-shape simplicity plus the no-`outputSchema` implementation requirement, so the artefact carries one rationale.

### [BLOCKING:decision] Tracked `.mcp.json` has no stated disposition
**Section:** Scope / D15
**Issue:** `docs/rewrite-plan.md:51` lists `.mcp.json` among the deleted V1 files and says "T5b and T6 write the new documents and the new `.mcp.json` when the surface they describe exists"; no `.mcp.json` exists in the tree, and the discussion mentions only a README *snippet*, never whether the tracked repo-root file is recreated, deferred, or dropped. The no-machine-paths constraint makes this a real choice, not a formality.
**Fix:** State in Scope Out (or D15) that T6 does not recreate a tracked `.mcp.json`, and why (a real command path cannot be committed), or put it in scope with its contents fixed.

### [BLOCKING:design] Tool and property description text is undecided
**Section:** D5 / D6 / Testing
**Issue:** D6 says "the property description spells that out" but no section fixes the `toc` tool's `description` or its property descriptions, and the `tools/list` assertions in Testing cover count, name, required/optional, `minimum: -1` and absent `outputSchema` — never wording. That prose is exactly what the granted cell's prompt cost consists of and what decides whether the agent calls the tool (plan §9a: a cell that never calls it "measured the tool's prompt cost only"), and D6 itself cites V1 evidence that schema wording changed agent behaviour in 5 of 6 sessions.
**Fix:** Fix the tool description and the three property descriptions verbatim in the discussion (or state the length/shape rule they must satisfy) and say whether a test pins them.

### [NIT:consistency] Two spellings of the invalid-`depth` message
**Section:** D6 / D8 / Testing
**Issue:** D6 mandates `--depth must be -1 (whole tree) or a non-negative integer, got <n>`, D8 says CLI wording is used "verbatim where the condition is the same", and Testing says the handler returns "the CLI-worded message" — but the CLI's actual string is `--depth must be a non-negative integer or "all", got %q` (`internal/cli/flags.go:101-102`), which cannot be reused because `-1` is valid here.
**Fix:** Say once that this message deliberately diverges from the CLI's and that D6's literal is the asserted string.

### [NIT:scope] D13's success probe fixes no tool flags
**Section:** D13 item 2
**Issue:** The flag list omits `--allowedTools mcp__quarry__toc` and `--tools ""` (plan §9a's own allowlist row carries both), so nothing stops the probe agent answering from built-in reads; the rooting assertion — the load-bearing one for D3 — then has no `toc` answer to inspect.
**Fix:** Name the full flag set for the success session and require a `toc` `tool_use` record in the stream, as D14 already does for the denial session.

### [NIT:consistency] `repopath`'s scratch-tree copy keeps the `cli-tests` path
**Section:** D11, "Test fixtures for the moved tests"
**Issue:** The copy is said to need adjustment "for depth" only, but `internal/cli/scratchtree_test.go:39` hard-codes `.scratch/cli-tests/<name>`, and a `repopath` copy writing there is misleading and shares a parent directory with a package that tests in parallel. (No collision today: cli's surviving caller is `cli_test.go:33`'s `pipeline-*` names.)
**Fix:** Say the copy also renames its scratch subdirectory, e.g. `.scratch/repopath-tests/`.

## Verdict

REQUEST_CHANGES
Superseded rationale, undisposed `.mcp.json`, and unspecified tool description text block plan writing.
MILL_REVIEW_END
