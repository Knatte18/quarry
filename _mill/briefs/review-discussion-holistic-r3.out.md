MILL_REVIEW_BEGIN
# Review: Add file/dir toc verbs (Tree-sitter-backed)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: /home/knatte/Code/quarry/wts/toc-verbs/_mill/discussion.md
date: 2026-08-27
```

## Findings

### [BLOCKING:design] Header rule picks up `//go:build` directive blocks
**Section:** "File header: first block in the file, blank line tolerated"
**Issue:** "First comment block in the file" plus explicit blank-line tolerance selects the build-constraint comment, not the header — verified in this repo: `internal/proc/proc_windows.go:1` is `//go:build windows`, blank line, then the real header at `:3`; same shape at `internal/quarryengine/query/refs_integration_test.go:1`, `daemon/supervised_lsp_test.go:1`, `daemon/ensureserver_integration_test.go:1`. `toc dir internal/proc` would emit `header: "go:build windows"`.
**Fix:** Decide and state the rule for directive/pragma-only leading blocks (Go `//go:build` / `// +build`, `//go:generate`; Python shebang and coding lines) — skip them and take the next block, or state explicitly that they win.

### [BLOCKING:consistency] `toc dir` closed key set omits `partial`
**Section:** "Emitted schema" vs "Unparseable input: partial results"
**Issue:** The `toc dir` per-entry key set is declared closed as `path`/`language`/`header`/`test`/`generated`/`error`, with no `partial`; but the unparseable-input decision says an unreadable dir entry "never sets `partial` — `partial` means 'parsed, lossily'", which only makes sense if a dir entry whose *parse* was lossy does set it. `toc dir` must parse each file to extract the header, so the case is reachable.
**Fix:** State whether a `toc dir` entry carries `partial` when `HasError()` is true, and add or exclude the key in the closed set accordingly.

### [BLOCKING:design] Stale-invariant grep does not produce the stated list
**Section:** Scope → "Exact-count and exact-claim invariants"
**Issue:** The list is presented as "what it currently returns", but the grep does not return `internal/quarryengine/doc.go:66`, `quarry/facade_test.go:117`, `layering_test.go:159`, or `seam_enforcement_test.go:2-3`; it *does* return `seam_enforcement_test.go:10` ("five-package DAG"), which the list omits, plus `docs/scout-vs-grep.md:3` and `:130` ("LSP-backed"), which the rule sweeps into scope with no disposition. It also misses two genuinely stale sites: `layering_test.go:20` ("The six internal/quarryengine/... import paths this guard reasons about" — becomes eight) and `layering_test.go:53` ("query's production files import all four").
**Fix:** Either widen the pattern and re-derive the list from its actual output, or drop the "output of a repeatable enumeration" claim and label the list as hand-curated with an explicit disposition for the two `docs/scout-vs-grep.md` historical hits.

### [NIT:consistency] `minPackageDirs = 8` means different things in the two files
**Section:** Scope → invariants bullet on `minPackageDirs`
**Issue:** `layering_test.go` walks only `internal/quarryengine/` (6 dirs today, 8 after), while `seam_enforcement_test.go` walks that tree plus `quarry/` (7 today, 9 after) — its comment at `:100-103` states the floor is deliberately one below the real count. "Raise both to 8" preserves that slack but reads as an exact-count claim.
**Fix:** Note that 8 is the exact count in the layering guard and a deliberate floor in the seam guard, so the accompanying comment rewrites differ.

### [NIT:decision] `docs/` survey deliverable has no named file
**Section:** Scope → "A per-language docstring-association survey doc under `docs/`"
**Issue:** The deliverable is named only by description ("in the spirit of `docs/scout-multilang.md`"); the plan writer must invent the filename.
**Fix:** Name the file (e.g. `docs/toc-docstring-association.md`) so the deliverable is checkable.

## Verdict

REQUEST_CHANGES
Header rule mis-fires on build directives; `partial` schema conflict; grep enumeration does not match its stated list.
MILL_REVIEW_END
