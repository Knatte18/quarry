MILL_REVIEW_BEGIN
# Review: Add an MCP wrapper for quarry

```yaml
duration_s: 210.0
verdict: REQUEST_CHANGES
reviewer_model: opus
reviewer_self_id: Claude Opus 5 (Anthropic)
reviewed_file: _mill/discussion.md
date: 2026-08-29
```

## Findings

### [NIT:consistency] Malformed entry: whole-call or per-entry?
**Demoted-from:** BLOCKING
**Section:** error-mapping vs Testing tier 1 vs entry-shape-lsp-mirrored
**Issue:** error-mapping's closed whole-call set lists "Schema validation" as `isError`, and entry-shape-lsp-mirrored relies on schema validation to reject a non-`{query}` `workspace_symbol` entry "before the handler runs" — but Testing tier 1 requires a malformed `textDocument_definition` entry (`position` with no `textDocument`, neither `symbol` nor `position`) to be "rejected with a clear per-entry error", which is only reachable if the entry union is *not* encoded in the input schema.
**Fix:** Decide and state input-schema strictness per tool (is the three-form union a `oneOf` in the schema, or permissive with handler-side parsing?), and add the disposition of an unparseable entry to error-mapping — currently per-entry `status: "error"` is scoped to "everything surfacing from a per-entry facade call", which a parse failure is not.

### [BLOCKING:design] No per-tool input-parameter matrix
**Section:** param-split / per-entry-vs-call-wide
**Issue:** The parameter lists are global ("Call-wide: `lang`, `buildTags`, `docSentences`, `noVerify`, `targetDir`") with no statement of which tool exposes which, while the result-shape and `resolution`-marker tables are pinned per tool. Concretely: the CLI registers `--doc-sentences` on `toc file` only and deliberately not on `toc dir` (`internal/cli/toc.go:153-156`), registers no `--build-tags` on either toc command, and `toc`'s `--lang` is validated against `quarry.TOCLanguages()` via `validateTOCLang` (`toc.go:239`) with different semantics per subcommand (override detection vs restrict the listing) — not the registry key `resolveContext` validates.
**Fix:** Add a seven-row table naming each tool's exact input parameters (per-entry and call-wide), so a plan writer cannot put `docSentences`/`buildTags` on `toc_dir` or give `lang` one validation set across all seven.

### [BLOCKING:design] `except` absolutisation base unstated
**Section:** assert-no-callers-semantics / export-inventory / target-dir-resolution
**Issue:** `FilterUnexpectedCallers` is exported but takes a prebuilt `exceptAbs map[string]bool`; the CLI composes it inline (`cli.go:692-699`) by joining each relative `--except` onto the resolved target dir and `filepath.Clean`ing, to match `filepath.Clean(r.File)`. That composition is neither among export-inventory's thirteen nor in the "deliberately reimplemented, named individually" list, and target-dir-resolution's "every downstream consumer" enumeration omits it — so the base a per-entry `except` resolves against is undecided and could land on process cwd.
**Fix:** State the disposition explicitly: `except` entries resolve through the effective absolute `targetDir` and are cleaned before the set is built, and record it in the export-inventory dispositions.

### [NIT:consistency] `runPathBatch` cited at the wrong line
**Section:** entry-shape-quarry-native
**Issue:** "the CLI uses `symbol` in `runBatch` and `path` in `runPathBatch` (`toc.go:428`ff)" — `toc.go:428` is `classifyTOCError`; `runPathBatch` is at `toc.go:449`.
**Fix:** Correct the citation.

### [NIT:design] `assert_no_callers` status predicates unassigned
**Section:** export-inventory / assert-no-callers-semantics
**Issue:** The error predicates are pinned by reference to `classifyLookupError`/`classifySymbolError`, but the CLI's assert-no-callers path uses `emitAmbiguousOrError` (`cli.go:725-734`), which maps `ErrSymbolNotFoundSentinel` to a plain error envelope, never `not_found` — so MCP's `not_found` status here is new behaviour with no CLI counterpart.
**Fix:** State that the LSP predicates apply to `assert_no_callers` too, and that the resulting `not_found` is an intended batch-envelope addition rather than a divergence.

## Verdict

REQUEST_CHANGES
Entry-validation layer, per-tool parameter set, and `except` base are undecided.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 2._
MILL_REVIEW_END
