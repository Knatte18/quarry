MILL_REVIEW_BEGIN
# Review: Engine core (T3)

```yaml
duration_s: 274.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic Claude Opus 5, via Claude Agent SDK)
reviewed_file: _mill/discussion.md
date: 2026-09-03
```

## Findings

### [BLOCKING:design] Root-directory package has no valid unit
**Section:** D7 / D18 / D20 **Issue:** D7 gives a file the unit `<dirRelPath>`, and D20 makes `""` and `"."` valid targets, but `glyph/golang.go:checkGoUnit` rejects both — `""` → `ReasonUnitEmpty`, `"."` → `ReasonUnitDotSegment` — so a `.go` file at the repository root produces an `id` that `glyph.Parse` cannot read, failing Testing 15's parse assertion and leaving `SpansOf` no unit to invert. **Fix:** state the disposition for a root-level Go package (excluded from `symbols`, given a defined unit spelling, or declared out of scope with the reason), and say what `dir` is emitted for the root answer.

### [NIT:consistency] `SpansOf` and gitignore contradict each other
**Demoted-from:** BLOCKING
**Section:** D16 vs D22 **Issue:** D22 says the ignore pattern set is collected fresh "on each `TOC` and `SpansOf` call", but D16's algorithm parses "that directory's `.go` files" filtered only by D7's unit rule and never mentions the ignore set — so a gitignored `.go` file beside listed ones contributes spans `toc` never listed, which is a round-trip *extra* under Testing 14/15. **Fix:** decide and state whether `SpansOf` filters by gitignore, and make D22's per-call collection claim match.

### [BLOCKING:design] `const`/`var` symbols are widened without a derivation rule
**Section:** D5 **Issue:** D5 fixes "one symbol per declared name" but leaves `signature`, `start`, `sigend` and `end` undefined for the shapes it names — `var x, y int` (two symbols over one spec span), a grouped `const (…)` spec, and an `iota` block whose later specs are a bare name with no type and no value — where the analogous type case is pinned in detail (grouped vs ungrouped, `"type "` prepended, `goTypeBody`'s no-body → `sigend` 0). **Fix:** state the span and signature rule per shape, as D5's own note already does for docstring association.

### [NIT:design] Symlink as an explicit `TOC` target is undefined
**Section:** D19 / D20 **Issue:** D19 fixes symlink handling for entries the *walk* encounters; D20's validation ladder (outside-repo → not-found → answered) never says what an explicit target that is itself a symlink yields, and the `os.Stat`-vs-`os.Lstat` choice silently decides it. **Fix:** add the case to D20 the way the gitignored-target case is already handled.

### [NIT:consistency] `Makefile` is not an extension
**Section:** D10 **Issue:** `extensionHeaderRules` is defined as mapping "an extension to a pure-text header extractor", but the `#`-block row lists `Makefile`, for which `filepath.Ext` returns `""` — the key type cannot express it. **Fix:** say whether the table is keyed by extension plus a base-name fallback, or drop `Makefile`.

### [NIT:design] The Loomyard pin check has no stated mechanism
**Section:** D17 **Issue:** Tests must *fail* when the checkout's `HEAD` is not `72c23d9`, but the discussion never says how `HEAD` is read, while D9 rejects a `git` subprocess on cost/coupling grounds — leaving the reader to guess between `exec.Command("git")` in a test and parsing `.git/HEAD` (which differs for a worktree). **Fix:** name the mechanism.

## Verdict

REQUEST_CHANGES
Three unresolved rules: root-package unit, `SpansOf` gitignore contradiction, `const`/`var` derivation.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 2._
MILL_REVIEW_END
