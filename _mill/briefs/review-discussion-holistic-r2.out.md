MILL_REVIEW_BEGIN
# Review: Rethink quarry-mcp's per-call targetDir ergonomics

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (anthropic), best-effort self-assessment
reviewed_file: _mill/discussion.md
date: 2026-08-30
```

## Findings

### [BLOCKING:scope] Bench prompt edit breaks an unlisted pytest
**Section:** Scope "In" / `bench-ladder-update` / Testing
**Issue:** `bench/loomyard-eval/ladder/tests/test_ladder_config.py:311` asserts the literal `"Never set targetDir or buildTags" in prompt`; dropping the `targetDir` half at `ladder_config.py:383` fails that test, but the work inventory names only `ladder_config.py` and `gates.py`, and the Testing section never mentions the ladder pytest suite at all.
**Fix:** Add `test_ladder_config.py` to Scope "In" with a stated disposition for its literal assertion, and say in Testing whether the ladder pytest suite is run as part of verification.

### [NIT:consistency] "five input structs" enumerates six
**Section:** Scope "In", first bullet
**Issue:** The bullet says "all five input structs" then lists `lspInput`, `workspace_symbol`'s, `impact`'s, `assert_no_callers`'s, and *both* toc inputs — six structs; `grep` confirms six `TargetDir string \`json:"targetDir..."\`` fields (tools_lsp.go:34, tools_symbol.go:59, tools_impact.go:28, tools_assert.go:34, tools_toc.go:37, tools_toc.go:52).
**Fix:** Say "six input structs"; the enumeration itself is correct and complete.

### [NIT:design] Gate-retention rationale rests on a false premise
**Section:** `bench-ladder-update` rationale
**Issue:** "retaining the `targetDir` arm turns it into a cheap end-to-end assertion that the property really is gone from the live server" is not what the gate does — `gate_no_target_override` (`gates.py:117-136`) inspects transcript `tool_input` maps, so it observes only whether the *model* emitted the key, never the server's published schema; it would pass equally with the property still in place. Compounding this, the same run also loses the prompt line that suppressed the behaviour, while the gate stays `fatal=True`, which contradicts `hard-removal-not-graceful-ignore`'s "loud and self-correcting … retries without it".
**Fix:** Restate the rationale as "retained because it costs nothing and still guards the pinned-worktree constraint", and state whether a fatal kill is still wanted now that the schema rejects the key.

### [NIT:scope] Docs bullet omits the toc absolute-path asymmetry
**Section:** Scope "In", `docs/mcp-setup.md` bullet vs `cross-repo-escape-hatch-is-a-second-server`
**Issue:** The decision's Note mandates documenting that `toc_file`/`toc_dir` still escape the launch root via an absolute arg (`cli.ResolveTOCPath`, `internal/cli/toc.go:253-258`, returns `filepath.Clean(arg)` for an absolute arg) while the five LSP-backed tools cannot; the Scope bullet lists only three doc items and not this one.
**Fix:** Add the asymmetry to the `docs/mcp-setup.md` bullet so a plan writer working from Scope alone does not drop it.

## Verdict

REQUEST_CHANGES
One unlisted bench test breaks on the prompt edit; three minor consistency and scope gaps.
MILL_REVIEW_END
