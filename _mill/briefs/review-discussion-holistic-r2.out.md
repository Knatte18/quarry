MILL_REVIEW_BEGIN
# Review: Add an MCP wrapper for quarry

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic)
reviewed_file: _mill/discussion.md
date: 2026-08-29
```

## Findings

### [BLOCKING:design] Serialization rests on a misapplied single-flight claim
**Section:** batching-execution-model, part 2 + Constraints "The LSP client is single-flight"
**Issue:** The cited hazard is per-`lsp.Client` (`query/callers.go:52-55`), but every facade call builds its own client — `runOnConnection` (`query/refs.go:230`) calls `acquireConnection` (`refs.go:153`) → `daemon.EnsureServer` → `lsp.NewClientDial` (`daemon/ensureserver.go:367`) — and no in-process client cache exists, so two concurrent `tools/call` requests cannot consume each other's responses; the stated justification for the process-wide mutex is false as written.
**Fix:** State the real reason for serializing (or drop it), since a mutex held for a whole 64-entry call with a per-entry `--timeout` and no whole-call deadline gives unbounded head-of-line blocking that the "64 is a guard" rationale does not actually bound.

### [BLOCKING:scope] Helper export list is short of what two handlers need
**Section:** shared-resolution-helpers / within-filter-helpers / Constraints (last bullet)
**Issue:** The Constraints bullet permits only six exports, but `toc_file` parity needs `resolveDocSentences`/`resolveTOCConfigPath`/`loadTOCConfig`/`parseDocSentences` (`toc.go:306`, `tocconfig.go:105,162,47,83`), `assert_no_callers` needs `filterUnexpectedCallers` (`cli.go:737`) for declaration exclusion, and error-mapping claims it "mirrors `classifyLookupError`/`classifySymbolError` exactly" (`cli.go:942,963`) — all unexported, none listed, no disposition given.
**Fix:** Re-derive the export inventory from the actual per-handler call chain and state, per helper, whether it is exported or deliberately reimplemented.

### [BLOCKING:design] Launch-time `--target-dir` absolutisation unstated
**Section:** target-dir-resolution + mcp-json-wiring
**Issue:** The committed `.mcp.json` passes `--target-dir .`, which can only resolve against the server process cwd — the exact behaviour target-dir-resolution rejected as "silently wrong if the client launches the server from elsewhere" — and nothing says when the default is absolutised; `filterWithin` silently falls back to `filepath.Abs` against process cwd for a relative base (`cli.go:762-771`), and `WithCwd` panics on a non-absolute dir.
**Fix:** State that both the launch default and the per-call override are absolutised at a named point, and against what, before any `AbsOrJoin`/`FilterWithin`/state-dir use.

### [BLOCKING:design] Which tools carry `resolution: "complete"` is unspecified
**Section:** Technical context "Existing output envelope" + Testing tier 2
**Issue:** The CLI emits the marker only from `classifyLookupError` (`cli.go:947`) and impact (`impact.go:215,252`); `classifySymbolError` (`cli.go:963`) and both toc verbs never do. "Keep this marker in the MCP result envelope" plus an untargeted tier-2 assertion reads as all seven tools, which would contradict the entry-shape-quarry-native promise that quarry-native output stays byte-comparable with the CLI's.
**Fix:** Name the exact tools whose `found` entries carry the marker.

### [NIT:consistency] `structToFields` cited to the wrong file
**Section:** Technical context "Existing output envelope"
**Issue:** It is defined at `internal/cli/toc.go:401`, not `impact.go`; the `"toc: "` prefix claim and `rewordImpactMarshalFailure` (`impact.go:261`) are otherwise accurate.
**Fix:** Correct the citation.

### [NIT:scope] Facade-only layering for `internal/mcpserver` has no enforcement
**Section:** Constraints "imports the facade only" + Testing
**Issue:** `internal/quarryengine/layering_test.go` policies only `internal/quarryengine/...` (its table has no `internal/cli` or `internal/mcpserver` row), so the new constraint is convention-only while the analogous engine rule is mechanical.
**Fix:** Say whether the constraint is test-enforced or reviewer-enforced.

## Verdict

REQUEST_CHANGES
Four blocking gaps: false concurrency premise, incomplete export inventory, unstated targetDir absolutisation, ambiguous marker scope.
MILL_REVIEW_END
