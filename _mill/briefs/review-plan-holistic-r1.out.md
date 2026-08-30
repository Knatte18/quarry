MILL_REVIEW_BEGIN
# Review: Rethink quarry-mcp's per-call targetDir ergonomics — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude (Anthropic); presented to me as Opus 5. Best-effort self-assessment only — I cannot verify my own build from inside the session.
reviewed_file: plan/
date: 2026-08-30
```

## Findings

### [BLOCKING:design] Card 6 mandates a false absolute about state-dir keying
**Location:** batch 2 / card 6, requirement "Second"
**Issue:** The card requires writing "State is keyed by a hash of the cleaned absolute target path, so two worktrees of the same repository never share a daemon, lock, socket, or language-server process. No configuration and no repointing is involved." — but `cli.ResolveStateDir` (internal/cli/paths.go:118-140) only reaches `workspaceKey(targetDir)` in its third tier; when `--state-dir` or `$QUARRY_STATE_DIR` is set, the leaf is that value verbatim and `targetDir` never enters the key, so two worktrees pinned to one explicit state dir do collide.
**Fix:** Qualify the claim to the default (user-cache) tier and name `--state-dir`/`$QUARRY_STATE_DIR` as the exception, since the same file's `## Launch-only flags` section documents `--state-dir` and card 6 leaves that bullet in place.

### [BLOCKING:consistency] Card 8 declares no Edits and no Commit yet authorizes source edits
**Location:** batch 2 / card 8
**Issue:** `Edits: none`, `Creates: none`, `Commit: none`, but Requirements say "If any check fails, fix the offending comment or string in the file it lives in and re-run all three" — any such fix lands in `internal/mcpserver/*` or `docs/mcp-setup.md` with no declared Edits entry, no commit, and no Go compile, since batch 2's `verify:` is the ladder pytest suite only.
**Fix:** Either make card 8 purely diagnostic (a failed check fails the batch and is fixed by re-opening the owning card), or give it an explicit `Edits:` list, a commit message, and a Go verify step.

### [BLOCKING:scope] Card 8's check-one whitelist omits real test-file survivors
**Location:** batch 2 / card 8, check one
**Issue:** The grep is `grep -rni 'targetdir' internal/mcpserver/ docs/mcp-setup.md`, which includes `*_test.go`, but the enumerated "intentional survivors" list covers only production identifiers; legitimate hits it does not sanction include `TestExceptSet_ResolvesAgainstTargetDirNotProcessCwd` (nativeentry_test.go:132), `TestCallTool_TargetDirIsAbsoluteEvenFromRelativeProcessCwd` plus `launchTargetDir`/`gotTargetDir` (transport_errors_test.go:341,359,366), `TestAssertHandler_RelativeExceptResolvesAgainstTargetDir` (tools_assert_test.go:123), and the two new tests cards 1 and 2 themselves add.
**Fix:** Extend the whitelist to cover test-function names and test-local identifiers, or scope check one to non-`_test.go` files plus a separate, explicitly enumerated test-file pass.

### [NIT:consistency] mcp-setup.md's opening summary is not dispositioned
**Location:** batch 2 / card 6
**Issue:** `docs/mcp-setup.md` lines 5-8 enumerate what the document covers; card 6 inserts a new top-level section between `## What the committed .mcp.json does` and `## Cold-start behaviour` and never says whether that enumeration should gain the new section.
**Fix:** State explicitly that the intro list is either extended with the scoping-contract section or deliberately left as-is (it already omits `## Launch-only flags`).

### [NIT:consistency] Completeness gate's check three omits exceptSet
**Location:** batch 2 / card 8, check three; overview `greps-are-necessary-but-not-sufficient`
**Issue:** The decision's rationale names four grep-invisible items — three input-struct doc comments and `exceptSet`'s "the effective absolute target directory" phrasing (nativeentry.go:119-120) — but check three re-reads only the six input structs, so `exceptSet` is verified by no gate, only prescribed by card 4.
**Fix:** Add `exceptSet`'s doc comment to check three's mandatory re-read list so the gate matches the rationale it cites.

## Verdict

REQUEST_CHANGES
One false doc claim, one under-specified gate card, one unreliable grep whitelist.
MILL_REVIEW_END
