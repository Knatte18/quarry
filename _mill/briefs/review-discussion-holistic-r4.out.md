MILL_REVIEW_BEGIN
# Review: Add file/dir toc verbs (Tree-sitter-backed)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus-class model (Anthropic); environment reports claude-opus-5
reviewed_file: /home/knatte/Code/quarry/wts/toc-verbs/_mill/discussion.md
date: 2026-08-27
```

## Findings

### [BLOCKING:design] Stale-doc grep matches phrases, not the invariant
**Section:** § Scope → "Exact-count and exact-claim invariants" **Issue:** The enumeration method searches nine fixed phrases, so it misses prose that enumerates the engine package set without using any of them — verified misses: `internal/quarryengine/doc.go:24-25` ("this package and its lsp, registry, daemon, and query subpackages"), `doc.go:36-37` ("walks internal/quarryengine/ recursively — this package, lsp, registry, daemon, daemon/daemontest, and query"), `doc.go:47-71` (the bulleted package-layout list, which needs two new bullets), and `internal/quarryengine/seam_enforcement_test.go:2-3` ("across every subpackage — the root leaf, lsp, registry, daemon, daemon/daemontest, and query"). **Fix:** State the invariant being defended ("any prose that enumerates the engine's packages or counts them"), and give an enumeration method that finds those enumerations rather than a fixed phrase list.

### [BLOCKING:design] Shipped tagged build is exercised by no test
**Section:** § Decisions → "Grammar set"; § Testing → "internal/quarryengine/treesitter" **Issue:** README will document `-tags "grammar_subset,grammar_subset_go,…"` as the build, but `go test ./...` runs untagged, where all 206 grammars are embedded — so the grammar-load test passes regardless, and the claim that it is "the canary for a grammar-registry change in the upstream library" does not hold for the configuration users actually build; a renamed or mistyped subset tag ships a binary where every `toc` call fails while tests stay green. **Fix:** Decide whether the treesitter package's load test is also run under the subset tags (and say so in Testing), or state explicitly that the tagged build is verified by hand only.

### [NIT:design] Result ordering is unspecified in a closed key set
**Section:** § Decisions → "Emitted schema"; § Testing → "Integration" **Issue:** The key sets are fixed exhaustively, but neither `symbols` nor `files` has a stated order, while the integration test asserts "the full JSON envelope" of `internal/output/output.go` — which needs a determinate order to be writable. **Fix:** State source order for `symbols` and directory-read order for `files`, or say ordering is unspecified and tests must not depend on it.

### [NIT:decision] toc's engine error surface left to implementation
**Section:** § Scope (`quarry/facade_test.go:117`, "check rather than assume") **Issue:** Whether `internal/quarryengine/toc` introduces sentinel errors is never decided, yet it determines both the "seven re-exported sentinel error values" count and how `internal/cli` derives batch statuses — the existing verbs classify via `errors.Is`/`errors.As` (`internal/cli/cli.go:875-906`), while toc's table implies most classification happens CLI-side from `os.Stat` and the extension map. **Fix:** State whether toc adds engine sentinels or returns only generic errors with all status classification done in the CLI.

### [NIT:design] `generated` marker vs the header rule
**Section:** § Decisions → "File header"; "Test and generated flags" **Issue:** The Go directive list (`go:build`, `+build`, `go:generate`, `go:embed`, `nolint`) excludes `// Code generated … DO NOT EDIT.`, so a generated file's `header` becomes the generation banner — the same class of non-purpose noise the `go:build windows` rule was added to remove. **Fix:** Say whether a generated-marker block counts as a directive block for header purposes or is emitted as the header.

## Verdict

REQUEST_CHANGES
Enumeration method misses stale package enumerations; documented tagged build has no test coverage.
MILL_REVIEW_END
