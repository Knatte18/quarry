MILL_REVIEW_BEGIN
# Review: Add an MCP wrapper for quarry

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-4-20250514 (best-effort; presented to me as "Opus 5"/opushigh)
reviewed_file: /home/knatte/Code/quarry/wts/quarry-mcp-wrapper/_mill/discussion.md
date: 2026-08-29
```

## Findings

### [BLOCKING:consistency] Q&A log contradicts mcp-json-wiring on --target-dir
**Section:** Q&A log (last-but-two entry) vs `### mcp-json-wiring`
**Issue:** mcp-json-wiring decides `.mcp.json` runs `go run ./cmd/quarry-mcp` with **no** `--target-dir` and explicitly argues against `--target-dir .`; the Q&A log states the committed command is `go run ./cmd/quarry-mcp --target-dir .`.
**Fix:** Update the Q&A entry to the superseding decision, or state which form is authoritative.

### [BLOCKING:design] workspace_symbol cannot honour two of the three entry forms
**Section:** `### entry-shape-lsp-mirrored`
**Issue:** The three-form union (position / `{symbol}` / `{textDocument, symbol}`) is applied uniformly to all three LSP-mirrored tools, but `query.Symbol` (`internal/quarryengine/query/symbol.go:71,90`) reads only `opts.Query.Symbol` — a position entry would query the empty string and a file-scoped entry would silently drop the scoping, both without error. The CLI never offers either form for `symbol` (`cli.go:466-492`, no `--in-file`, no `--within`).
**Fix:** State `workspace_symbol`'s accepted entry form(s) explicitly and what the handler does with the other two.

### [BLOCKING:scope] export-inventory never walked the toc handlers' call chain
**Section:** `### export-inventory`
**Issue:** The list claims completeness "derived from the actual per-handler call chain", but `toc_file`/`toc_dir` also reach `resolveTOCPath` (`toc.go:253`), `validateTOCLang` (`toc.go:239`), `classifyTOCError` (`toc.go:428`) and `tocDirEntries` (`toc.go:376`), none of which is reachable through any of the nine and none of which has a stated disposition. `tocDirEntries` is the subtle one: `toc.DirEntry.Name` is `json:"-"` (`toc/types.go:80`), so `StructToFields(TOCDirResult)` alone yields `files` entries carrying neither `name` nor `path` — the file-reference-form exception names the `filepath.Join(arg, Files[i].Name)` formula but not this trap. `StructToFields`'s "Needed by" column also omits `toc_dir`.
**Fix:** Extend the inventory over the toc call chain with an exported-or-reimplemented disposition per identifier.

### [BLOCKING:design] Per-entry status rule has no toc branch
**Section:** `### error-mapping` / `### export-inventory`
**Issue:** The reused predicates are fixed as `errors.As ErrAmbiguousSymbol` → `ambiguous`, `errors.Is ErrSymbolNotFoundSentinel` → `not_found`, "everything else `error`". toc never uses those sentinels: the CLI returns `not_found` from `os.IsNotExist` on the stat (`toc.go:296`) and `error` for a directory-shaped argument or unsupported language (`toc.go:302,428`). Applied literally, an MCP `toc_file` on a missing file reports `error` where the CLI reports `not_found`, contradicting entry-shape-quarry-native's byte-comparability promise.
**Fix:** State the toc-side status rule alongside the LSP-side predicates.

### [BLOCKING:design] "Daemon spawn failure" has no whole-call detection point
**Section:** `### error-mapping`
**Issue:** `isError` is reserved for whole-call failures including "a daemon spawn failure", but unusable `targetDir` and malformed `servers.yaml` are pre-flight (`resolveContext`) while a spawn failure surfaces only as the error returned from a per-entry facade call, where the stated predicate rule sends it to per-entry `status: "error"`. The two rules disagree and no discriminating predicate is named.
**Fix:** Either drop spawn failure from the `isError` list or name the sentinel/condition that lifts it to whole-call.

### [NIT:design] Empty target array unspecified
**Section:** `### batching-execution-model`
**Issue:** The upper bound (64) is pinned, but a zero-length array has no stated outcome — whole-call `isError`, schema `minItems: 1`, or an empty `results` array.
**Fix:** Name one.

## Verdict

REQUEST_CHANGES
Five decision-level gaps: an internal contradiction, an unworkable entry union, and an inventory short of the toc chain.
MILL_REVIEW_END
