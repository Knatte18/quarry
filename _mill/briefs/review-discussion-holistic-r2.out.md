MILL_REVIEW_BEGIN
# Review: Engine core (T3)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic), accessed via the Claude Agent SDK
reviewed_file: _mill/discussion.md
date: 2026-09-03
```

## Findings

### [BLOCKING:design] Emitted order of `files` and `dirs` is unstated
**Section:** D12 / D13 / Testing 12–13
**Issue:** No decision fixes the order of `DirAnswer.Files` or `DirAnswer.Dirs`, yet `internal/quarryengine/toc/types.go` states outright that "Ordering is the caller's (internal/cli's) responsibility, not this package's" — and that caller is T5a, out of scope here; §4's examples are alphabetical and the golden tests compare marshalled JSON, so an unpinned order makes the byte-for-byte criterion untestable (and a `-update` run would silently freeze whatever order the walk happened to produce).
**Fix:** State the ordering rule for `files` and `dirs` (e.g. lexicographic by base name / `dir`) as a decision, and that T3 owns it rather than the CLI.

### [BLOCKING:design] Recursive walk has no symlink or cycle rule
**Section:** D13 (`DepthAll`) / D9
**Issue:** T3 introduces directory recursion for the first time — V1's `TOCDir` never descends (`internal/quarryengine/toc/toc.go:146`) — and nothing says whether the walk follows directory symlinks, so `--depth all` over a real tree can loop forever or list the same package twice; `os.ReadDir`'s `IsDir()` is also false for a symlink-to-directory, which would silently emit one as a *file* entry.
**Fix:** Decide symlink policy (do not follow / follow with a visited-set) and how a symlink appears in the answer.

### [BLOCKING:decision] `TOC`'s target-error contract is undefined
**Section:** D15 / D13
**Issue:** `TOC(target string, opts TOCOptions) (DirAnswer, error)` never says what happens for a nonexistent target, an absolute target, or one escaping the root via `..`; T5a's exit codes and `ok`/`status` map exactly this error vocabulary, so leaving it unstated hands the envelope's failure half to a later task that the discussion says T3 pins.
**Fix:** State the validation rule for `target` and the error values `TOC` returns for each rejection.

### [BLOCKING:decision] `ErrLanguageUnsupported` has no stated disposition
**Section:** Technical context (today→after table)
**Issue:** The table moves `errors.go` unchanged, but its only content is `ErrLanguageUnsupported` ("a file's extension maps to no language, or a requested language has no toc strategy") and both triggers are removed by this task — D10 lists every file regardless of language and D15 deletes `langOverride`, so the sentinel is either dead or silently repurposed.
**Fix:** Say whether the sentinel is deleted, kept for a new caller, or replaced by the D13 target errors.

### [BLOCKING:decision] `toc_integration_test.go` is absent from the ported-test list
**Section:** Testing — "Ported tests"
**Issue:** The paragraph enumerates five test files as if exhaustive, but the package also holds `internal/quarryengine/toc/toc_integration_test.go`, which calls `TOCFile(path, "", Options{DocSentences: 1})` — an entry point, a parameter and a field all deleted by D14/D15 — and hard-codes `internal/quarryengine/treesitter/treesitter.go`, a path D1 moves.
**Fix:** State its disposition (port onto `Repo.TOC` with the new path, or delete).

### [BLOCKING:consistency] D15 caches the ignore set; Constraints forbid caching
**Section:** D15 vs Constraints
**Issue:** D15 says the `Repo` is "where the gitignore pattern set (D9) is cached for the process", while Constraints say "No cache, index, daemon or concurrency in the engine … Every answer reads source as it is at that moment" — under T6's long-lived MCP surface a process-lifetime pattern set goes stale against an edited `.gitignore`.
**Fix:** Either scope the constraint explicitly to parsed-source caching, or state the ignore set's invalidation rule.

### [NIT:scope] `toc/doc.go` missing from the today→after table
**Section:** Technical context (table)
**Issue:** The table maps the root `doc.go` but not `internal/quarryengine/toc/doc.go`; two package-comment files collapse into one `internal/engine` package, and the `toc` one documents the sentence-boundary rule D14 deletes.
**Fix:** Add a row saying the `toc` package comment is folded into the new `doc.go` minus the deleted rule.

### [NIT:decision] `Strategy`'s `known` return values are left orphaned
**Section:** Technical context (`strategy.go` row) / D12
**Issue:** The row says `Strategy` only *gains* `PackageDoc` and a unit on `Symbols`, but D12 drops the `*bool` discipline that the existing `Generated(...) (generated, known bool)` and `TestFile(base) (isTest, known bool)` returns exist to feed, leaving `known` with no consumer.
**Fix:** State whether the `known` returns are dropped from the interface.

## Verdict

REQUEST_CHANGES
Six blocking gaps: answer ordering, symlinks, target errors, two undisposed artifacts, cache contradiction.
MILL_REVIEW_END
