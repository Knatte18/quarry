MILL_REVIEW_BEGIN
# Review: MCP, thin (T6)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5), high reasoning effort
reviewed_file: _mill/discussion.md
date: 2026-09-04
```

## Findings

### [BLOCKING:consistency] `%w: …` wrapping cannot preserve CLI message bytes
**Section:** D11 "Error shape", vs Constraints "same messages"
**Issue:** D11 prescribes `fmt.Errorf("%w: …")` around two sentinels while requiring `internal/cli` to preserve its message bytes exactly; the current strings are `no repository root found above %s; pass --root` (`internal/cli/root.go:28`) and `--root is not a directory: %s` (`root.go:55`), and no sentinel text plus `": "` reproduces the first one — mapping back via `usageError(err.Error())` yields `no repository root found: above …; pass --root`. No test guards this: `root_test.go:94,166` only assert substrings, and `docs/research/output-formats/` carries no error golden, so the drift is silent.
**Fix:** State which side owns the literal — either drop the `": "` wrap form so the sentinel is a prefix of the real sentence, or say `internal/cli` re-formats its own two strings from the sentinel — and name the assertion that pins the exact bytes.

### [NIT:decision] Server `Implementation.Version` has no disposition
**Section:** D4 / Technical context (`mcp.Implementation{Name: "quarry", Version: ...}`)
**Issue:** The server name and tool name are fixed as hard external contracts, but the version string is left as a literal ellipsis; the implementer must invent one, and it is sent on every `initialize`.
**Fix:** State the value (e.g. a fixed placeholder, or explicitly "any, not part of any contract").

### [NIT:scope] `--root` ships with no test and no probe
**Section:** Testing (`cmd/quarry-mcp`) + D13 item 2
**Issue:** Flag handling is deliberately untested ("needs no test of its own beyond the package compiling"), and D13's live probe is run *without* `--root` (from another repo, exercising cwd discovery). So the one path that rescues a falsified D3 — wiring `--root {target_dir}` — is verified by nothing.
**Fix:** Either name a table test for root resolution at the `internal/mcpserver` boundary, or state that `--root` is knowingly unverified until the placeholder contract has a consumer.

### [NIT:consistency] Moved tests both "move" and "stay unchanged"
**Section:** Testing, `internal/repopath` paragraph
**Issue:** The same paragraph says `internal/cli/root_test.go` and `target_test.go` move with the functions, then that "`internal/cli`'s own tests stay and must still pass unchanged" — readable only if the second clause excludes the two files it just moved. The moved tests cannot pass unchanged either: they assert `err.(usageError)` (`root_test.go:89,161`), a type that does not exist in `repopath`.
**Fix:** Say explicitly which files are deleted from `internal/cli`, and that the moved cases switch from the `usageError` type assertion to `errors.Is` on the new sentinels.

### [NIT:design] No disposition on `depth: -1` payload size on the measured surface
**Section:** D5a (`depth` description) / D7
**Issue:** The tool prose advertises whole-tree recursion to the agent and D7 returns the full envelope with no cap, on the one surface whose token cost T7 measures; the discussion states no position (no cap, truncation, or warning) even to say "none, deliberately — same as the CLI".
**Fix:** Record one sentence taking the no-cap position, noting V1's `toc_dir` had the same reach so the baseline comparison is unaffected.

## Verdict

REQUEST_CHANGES
Error-text ownership in the `repopath` lift is unresolved and unguarded; four nits.
MILL_REVIEW_END
