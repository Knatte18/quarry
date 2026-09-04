MILL_REVIEW_BEGIN
# Review: MCP, thin (T6) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Claude Opus-family; harness-reported ID)
reviewed_file: plan/
date: 2026-09-04
```

## Findings

### [BLOCKING:scope] stdout decision has no enforcement mechanism
**Location:** 00-overview.md, Shared Decision "stdout belongs entirely to the MCP transport"; batch 2 card 10.
**Issue:** The sibling facade-only decision gets a mechanical test on the explicit ground that "review alone is not a mechanism", yet the stdout rule — which the same plan calls "the one way this binary fails catastrophically and silently" — is enforced by prose in card 9's header comment only, even though card 10's walk already parses every `.go` file in both `internal/mcpserver/` and `cmd/quarry-mcp/`.
**Fix:** Extend card 10's Requirements to also fail on an `os.Stdout` / `fmt.Print*` reference in those two directories, or state why the two equally-weighted decisions get unequal mechanisms.

### [BLOCKING:design] card 8 leaves two identifiers unnamed, one used cross-batch
**Location:** 02-mcp-server.md card 8; 03-mcp-server-tests.md card 15.
**Issue:** Card 8 names `tocInput`, `tocInputSchema`, `registerTOC`, `serverVersion`, but describes the decision function and the failure-result helper only by signature. Card 15, in a different batch, requires "calling the handler's decision function directly" — an identifier no card fixes, and card 12/13/14 headers reference the same layer.
**Fix:** Name both functions in card 8's Requirements and have card 15 cite that name.

### [NIT:consistency] broken-symlink expectation departs from the discussion, unrecorded
**Location:** 03-mcp-server-tests.md card 16.
**Issue:** Discussion Testing says the broken-symlink case "must return `isError` true … and text equal to `quarry.RenderErrorJSON(<the CLI's own wording>)`"; card 16 requires the opposite (success, name-only entry). Card 16 is factually right — `internal/engine/toc.go:176-178` returns `FileEntry{Name: targetBase}` with no error — but the departure sits in a card body while the only other CLI-wording divergence was lifted into Shared Decisions.
**Fix:** Record the symlink-succeeds disposition in `## Shared Decisions` alongside the depth-wording divergence.

### [NIT:scope] layering test is direct-import only; card 10's mandated header would overclaim
**Location:** 02-mcp-server.md card 10.
**Issue:** Discussion Testing asks for a walk of the *import graph*; card 10 specifies a direct-import scan of two directories. Direct-only is the only workable form (the `quarry` facade necessarily imports the engine), but the narrowing is unstated, so an engine dependency reached through a future intermediate `internal/` package passes the check.
**Fix:** State in card 10 that the check is direct-import only and why, so the required file header says that rather than "enforces the facade-only rule".

### [NIT:consistency] card 18's rooting rationale is refuted by D13
**Location:** 04-docs-and-config.md card 18.
**Issue:** "So the committed configuration exercises the rooting the measurement depends on" — D13 states the opposite: running from the quarry repo "would prove nothing about D3, because cwd discovery there succeeds whether or not the inheritance assumption holds". The `.mcp.json` spawns with quarry's own root as cwd, not a foreign `Dir: dest`.
**Fix:** Drop that clause; the card's other three reasons (plan §2 instruction, no build step, no machine paths) carry the decision.

### [NIT:consistency] card 2 leaves a stale in-body comment in cli.go
**Location:** 01-repopath-extraction.md card 2.
**Issue:** Card 2 scopes doc-comment edits to "`Run`'s doc-comment step list", but `internal/cli/cli.go:127-129` carries an inline comment asserting "`resolveRoot` and `discoverRoot` are contracted (card 11) to return only a `usageError`", which becomes false and names a previous task's card number.
**Fix:** Name that comment explicitly in card 2's Requirements, as card 1 does for the moved files' stale cross-references.

### [NIT:scope] no card pins mcpserver's no-repository-root wording
**Location:** 02-mcp-server.md cards 6 and 7.
**Issue:** Card 6 mandates `quarry-mcp: no repository root found above <cwd>; pass --root`; card 7's four cases cover discovery-success, relative, absolute and not-a-directory only. Batch 1 card 4 exists precisely to pin the CLI's structurally identical unreachable sentence, so the asymmetry reads as an oversight rather than a choice.
**Fix:** Either add the case (or a named formatter card 7 can call, as card 2 did with `rootUsageMessage`), or state in card 7 why this sentence is left unpinned.

### [NIT:consistency] card 5 states "direct requirements" then "expect indirect"
**Location:** 02-mcp-server.md card 5.
**Issue:** The first paragraph says to add both modules "as direct requirements"; the fifth says to expect them "marked indirect until card 8 runs tidy". The second is correct for `go get` with no importer, so the first is a superseded statement inside the same card.
**Fix:** Drop "as direct" from the opening sentence; the deferred-tidy paragraphs already carry the real contract.

## Verdict

REQUEST_CHANGES
Two enforcement/naming gaps; the rest of the plan verifies cleanly against source.
MILL_REVIEW_END
