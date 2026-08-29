MILL_REVIEW_BEGIN
# Review: Add an MCP wrapper for quarry

```yaml
duration_s: 231.0
verdict: APPROVE
reviewer_model: opus
reviewer_self_id: Claude Opus-class model (self-assessed as claude-opus-4.5 lineage; runtime reports claude-opus-5)
reviewed_file: /home/knatte/Code/quarry/wts/quarry-mcp-wrapper/_mill/discussion.md
date: 2026-08-29
```

## Findings

### [NIT:consistency] impact entry `target` key collision unresolved
**Demoted-from:** BLOCKING
**Section:** result-shape ("The per-entry object, pinned", table row + note)
**Issue:** The table lists `target`/`definition`/`callers` as *entry-level* keys for `impact` (matching the CLI, where `runBatch` at `cli.go:985-988` flattens `classifyImpactError`'s fields into the entry map and `impact.Result.Target` marshals to top-level `"target"`, `impact/types.go:92`), while the note claims impact's own `target` "is nested under the result" — no wrapper key is named anywhere in the discussion, so the flattened implementation silently overwrites the echoed input `target` and destroys the attributability the section pins.
**Fix:** State the exact impact `found` entry shape — either a named wrapper key for the marshalled result, or a renamed envelope/result key — and say which of the two colliding `target`s survives.

### [NIT:consistency] workspace_symbol bad entry: schema vs handler
**Demoted-from:** BLOCKING
**Section:** Testing tier 1 vs input-schema-strictness / entry-shape-lsp-mirrored
**Issue:** Tier 1 asserts a position or file-scoped entry to `workspace_symbol` is "rejected at schema validation, never reaching the handler" (whole-call failure), while input-schema-strictness says the entry union is deliberately *not* schema-encoded and entry-shape-lsp-mirrored says such an entry is "rejected **per entry** by the handler"; tier 2 assumes the per-entry rule. The two produce opposite observable results (whole call `isError` vs 19 surviving entries).
**Fix:** Delete or reword the tier-1 claim so it matches the per-entry-status rule, or state the deliberate exception and why `workspace_symbol` alone gets entry-level schema strictness.

### [NIT:consistency] toc entry arrays: objects or strings
**Section:** input-schema-strictness vs entry-shape-quarry-native / per-entry-vs-call-wide matrix
**Issue:** input-schema-strictness schema-encodes "the entry array being an array of objects" uniformly across all seven tools, but `toc_file`/`toc_dir` are decided as "arrays of plain paths" (strings) in two other places.
**Fix:** Scope the "array of objects" clause to the five object-entry tools, or state the toc entry object form explicitly.

### [NIT:design] `docSentences` JSON type unstated
**Section:** per-entry-vs-call-wide / input-schema-strictness
**Issue:** Call-wide parameter types are schema-enforced as a whole-call failure, but `--doc-sentences` legitimately takes a number or the string `"all"` (`toc.go:156`, `parseDocSentences` at `tocconfig.go:83`), and the discussion never says whether the MCP parameter is a string, a number, or a union — so `{"docSentences": 3}` may fail the whole call.
**Fix:** Pin the JSON type of `docSentences` in the schema and how a numeric value is accepted.

### [NIT:scope] README and gitignore dispositions unstated
**Section:** Scope ("In") / mcp-json-wiring
**Issue:** `README.md` documents verbs and `go build -o quarry ./cmd/quarry` but has no stated disposition for the new `cmd/quarry-mcp` binary; `.gitignore` ignores `/quarry` only, so the `go build -o` warm-start alternative documented in `docs/mcp-setup.md` leaves an untracked binary at the repo root.
**Fix:** State whether README gets an MCP pointer and whether `.gitignore` gains the new binary name.

## Verdict

APPROVE
Two self-contradictions — impact's `target` collision and workspace_symbol's rejection level — must be settled first.
_Note: 2 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 0._
MILL_REVIEW_END
