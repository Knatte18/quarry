MILL_REVIEW_BEGIN
# Review: Rethink quarry-mcp's per-call targetDir ergonomics — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic Claude Opus 5); self-assessed, not independently verifiable from inside the session
reviewed_file: plan/
date: 2026-08-30
```

## Findings

### [BLOCKING:scope] Completeness whitelists omit two real survivors
**Location:** batch 2 card 8 (checks 1 and 1-b), and batch 1 card 4's leave-unchanged list
**Issue:** `resolveTOCFileEntry(arg, targetDir, ...)` and `resolveTOCDirEntry(arg, targetDir, ...)` in `internal/mcpserver/tools_toc.go` (params at :107/:141, doc comments at :99-103/:133-135, uses at :108/:142) are legitimate `targetDir` identifiers that survive the change, but card 8's production whitelist enumerates only `nativeEntry.query`, `lspEntry.query`, `resolveEntryFile`, `exceptSet`, and the two handler locals; card 4's "leave unchanged" list has the same omission. On the test side, `targetDir := "/some/project"` at `internal/mcpserver/nativeentry_test.go:133` is a test-local identifier outside check 1-b's enumeration (`launchTargetDir`/`gotTargetDir` in `transport_errors_test.go`, the table field in `translate_test.go`).
**Fix:** Add the two `resolveTOC*Entry` `targetDir` parameters to card 4's leave-unchanged list and to check 1's whitelist, and `nativeentry_test.go`'s `targetDir` local to check 1-b's.

### [BLOCKING:scope] Card 6 Context omits the file defining ResolveStateDir/workspaceKey
**Location:** batch 2 card 6
**Issue:** The card's Requirements name `cli.ResolveStateDir` and `workspaceKey` and dictate their three-tier precedence in detail, but `internal/cli/paths.go` (where both live, :76 and :118) is in neither `Context:` nor `Edits:`, and no file that is in Context calls either — `mcpserver.go` does not, `callcontext.go` is not listed. The doc paragraph's correctness rests entirely on the card's prose being right, with no way for the implementer to check.
**Fix:** Add `internal/cli/paths.go` to card 6's `Context:`.

### [BLOCKING:design] Cwd-inheritance platform claim is asserted unsourced
**Location:** batch 2 card 6, requirement "Second"
**Issue:** The card requires documenting "A project-scoped `.mcp.json` is launched by the client with the project root as its working directory ... No configuration and no repointing is involved" as an unconditional guarantee, and this claim is the entire justification for removing the per-call override; no artifact in the card's Context establishes it — `.mcp.json` carries only `command`/`args`, `main.go` only calls `os.Getwd`, and the existing `docs/mcp-setup.md:20-21` deliberately hedges ("even though `.mcp.json` never states it explicitly"). The card qualifies the adjacent state-dir keying claim explicitly but leaves this stronger, client-side one unqualified.
**Fix:** Require the paragraph to scope the claim to clients that launch a project-scoped server from the project root (and name the stderr `resolved target directory` line as how an operator confirms it per session), rather than stating it as a universal client guarantee.

### [NIT:consistency] Replacement prompt rationale diverges from the gate it mirrors
**Location:** batch 2 card 7
**Issue:** The new second line, "the default build-tag set is the one this run is scoped to", is not the reason `gate_no_target_override` marks a `buildTags` key fatal — `gates.py:119-121` gives the pinned-worktree constraint and the cold cell's daemon key. The card correctly insists the rationale must change with the key list, then substitutes one that does not match the gate.
**Fix:** Have the card specify a rationale naming the run's pinned build-tag scoping / daemon key, matching `gate_no_target_override`'s own docstring.

## Verdict

REQUEST_CHANGES
Two Context/whitelist gaps and one unsourced platform claim; the Go removal itself is sound.
MILL_REVIEW_END
