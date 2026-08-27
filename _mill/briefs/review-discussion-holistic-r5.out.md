MILL_REVIEW_BEGIN
# Review: Add file/dir toc verbs (Tree-sitter-backed)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus (system-reported model id claude-opus-5); best-effort self-assessment
reviewed_file: /home/knatte/Code/quarry/wts/toc-verbs/_mill/discussion.md
date: 2026-08-27
```

## Findings

### [BLOCKING:consistency] Go type unit: type_spec vs type_declaration
**Section:** "Signature: exact source text…" vs "Technical context — Per-language extraction shapes (Go)"
**Issue:** The Decision says a `type ( … )` block yields "one symbol per `type_spec`" and that a docstring attached to the `type` line "attaches to no individual spec and is dropped", while the Go notes say the walked kind is `type_declaration` with the name "nested one level down in a `type_spec`" and the docstring is the declaration's contiguous `comment` prev-siblings; for the common ungrouped `// Doc` + `type FileLock struct{…}` the two rules disagree on the symbol node, so the signature is either `type FileLock struct` (declaration first byte) or `FileLock struct` (spec first byte), and the docstring is either attached or dropped.
**Fix:** State which node is the symbol for a single-spec `type_declaration`, and state explicitly that its preceding comment block is its docstring and that the emitted signature includes the `type` keyword in both the grouped and ungrouped forms.

### [BLOCKING:design] Staleness invariant covers only counts, not LSP-only prose
**Section:** Scope → "Exact-count and exact-claim invariants"
**Issue:** The stated invariant is limited to prose that "enumerate[s] the engine's packages, count[s] them, count[s] quarry's verbs, or count[s] the facade's re-exports", but the inventory's own `README.md:3` entry is a non-count claim, and two more non-count claims verified in-tree are outside both the invariant and both greps: `README.md:4` ("by speaking the Language Server Protocol to each language's own server rather than reimplementing a parser per language") and `internal/cli/cli.go:23-29` (package cli's batch contract, "one JSON entry per symbol under a top-level \"results\" array", which toc's `path`-keyed driver contradicts).
**Fix:** Restate the invariant to cover any prose asserting quarry is LSP-only or describing the batch entry key, not only counts and package enumerations, so the plan writer's sweep is defined by the property rather than by two grep expressions.

### [NIT:consistency] Single-arg exit code for rank-3 outcomes
**Section:** "Batch mode" table vs "Path-type validation" / "Unparseable input"
**Issue:** "toc's exit codes are 0, 1, and 3 only" sits under a table whose rank-3 rows include wrong-path-type and unsupported-language, yet the unreadable-file case is separately fixed at "exit 1 single-arg", and `internal/cli/cli.go:13-21` documents the single-arg contract as 0/1/2 for every existing verb; the single-arg code for a wrong path type or a `.ts` argument is never stated.
**Fix:** Say that every single-argument toc failure is `output.Err` exit 1 and that rank 3 is a batch-only value.

### [NIT:scope] README testing-section update missing from the Scope inventory
**Section:** Scope "In" vs Testing
**Issue:** Testing says the two verification commands "belong in README's testing section", but the Scope inventory lists only the "Building and running" and verb-list README updates; `README.md:66-71` documents exactly two tiers today and would go stale unlisted.
**Fix:** Add the README "Testing" section update to the Scope "In" inventory.

### [NIT:design] `toc dir` aggregate parse cost never measured
**Section:** Scope "Out" — "Caching or a daemon for the parser"
**Issue:** "Parses are cheap enough to do per call (see 'Technical context' for the measurement)" is backed only by a single-file number (113 ms for 38.8 KB *including process startup*); no per-directory figure is recorded, though a `dirbench` spike artefact exists under `.scratch/tsspike/`.
**Fix:** Record the measured cost of one `toc dir` call over a realistic directory, or state the expected N-file cost explicitly.

## Verdict

REQUEST_CHANGES
Go type-symbol unit is self-contradictory and the doc-staleness invariant under-covers non-count prose.
MILL_REVIEW_END
