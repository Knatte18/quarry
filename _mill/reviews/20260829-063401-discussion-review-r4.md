MILL_REVIEW_BEGIN
# Review: Add an MCP wrapper for quarry

```yaml
duration_s: 259.0
verdict: REQUEST_CHANGES
reviewer_model: opus
reviewer_self_id: claude-opus-5 (Anthropic Opus-class; environment reports exact ID claude-opus-5)
reviewed_file: /home/knatte/Code/quarry/wts/quarry-mcp-wrapper/_mill/discussion.md
date: 2026-08-29
```

## Findings

### [NIT:consistency] Export count: nine vs twelve
**Demoted-from:** BLOCKING
**Section:** §Scope, §Constraints, §export-inventory
**Issue:** Scope says "Nine newly-exported helpers in `internal/cli`" and Constraints says "exporting the nine helpers named under export-inventory", but the export-inventory table lists twelve rows (`ResolveConfigPath`…`TOCDirEntries`), all of which I verified exist at the cited sites.
**Fix:** State twelve in both places, or name which three of the twelve are not in the permitted-edit set.

### [BLOCKING:design] Whole-call `isError` set is wrong for the toc tools
**Section:** §error-mapping
**Issue:** The "closed" pre-flight set is schema validation, `targetDir` absolutisation, and the three `resolveContext` steps — but `tocFileCommand`/`tocDirCommand` never call `resolveContext`, `LoadRegistry`, or `resolveStateDir` (toc.go:104-149), so applying it verbatim makes `toc_file` fail wholly on a malformed `servers.yaml` where the CLI succeeds; conversely toc's own up-front whole-call validations — `validateTOCLang` (toc.go:115) and the `--doc-sentences` flag parse (toc.go:127) — are absent from the closed set, so their disposition is undefined.
**Fix:** Scope the pre-flight set to the LSP-backed tools and state explicitly where `ValidateTOCLang` / doc-sentences-parse failures land for `toc_file`/`toc_dir`.

### [BLOCKING:design] Per-entry output schema never specified
**Section:** §result-shape
**Issue:** Each tool "declares a JSON output schema" and returns `structuredContent`, but the entry object's field set is stated nowhere and varies by status and tool — `references`/`definitions`/`symbols` plus `resolution`, `candidates` on ambiguous, `error`, `violation`+`callers`, `files`, or nothing on `not_found`. A declared schema must reconcile that union (which keys, which optional), and a strict one turns a valid mixed batch into a protocol-level failure.
**Fix:** Pin the entry schema — the common keys, the per-status optional keys, and whether the schema is per-tool-strict or permissive.

### [NIT:consistency] Superseded "prefer exporting" note in Technical context
**Section:** §Technical context, "`within` filtering is CLI-side"
**Issue:** Still reads "reusing these means exporting them too, or reimplementing … Prefer exporting if the logic is non-trivial" — non-committal language that `within-filter-helpers` already settled by decision, and which the same section elsewhere criticises as the round-1 defect.
**Fix:** Replace with a pointer to the within-filter-helpers decision.

### [NIT:decision] Inventory omits three call-chain identifiers and drifts on two line cites
**Section:** §export-inventory
**Issue:** The list claims completeness "derived by walking every handler's call chain", but `classifyImpactError` (impact.go:234, carries the `resolution` marker and the marshal-failure reword), `referenceFields` (cli.go:835) and `symbolMatchFields` (cli.go:793) get no named disposition — they fall only under the categorical "MCP writes its own". Also `ResolveBuildTags` is paths.go:61 not :63, and `ResolveStateDir` is paths.go:118 not :119.
**Fix:** Name the three under the reimplemented list and correct the two cites.

## Verdict

REQUEST_CHANGES
Export count contradicts itself; toc whole-call rule and output entry schema are unresolved.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 2._
MILL_REVIEW_END
