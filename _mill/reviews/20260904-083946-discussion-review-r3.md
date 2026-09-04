MILL_REVIEW_BEGIN
# Review: Facade + CLI, resolve + expand (T5b)

```yaml
duration_s: 357.0
verdict: APPROVE
reviewer_model: opus
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: _mill/discussion.md
date: 2026-09-04
```

## Findings

### [NIT:consistency] Three new goldens get no INDEX.md row or exit code
**Demoted-from:** BLOCKING
**Section:** D14 + D15
**Issue:** D14 says the expected exit code "moves into the test table and into `INDEX.md`'s own column", but D15 keys the mapping table on before-side files only; `resolve-not-found.txt`, `resolve-path.txt` and `expand-not-a-type.txt` have no before-side counterpart, so three of the eight new goldens — including both exit-1 cases — end up with no row and no recorded exit code, defeating D14's own rationale ("the exit code is still recorded in a tracked file").
**Fix:** State that the table also gains a `*(none)* | <after file> | new: …` row per new after-side golden, the way T5a already did for `toc-dir-text.txt` / `toc-file-text.txt`, so every after file carries an exit-code cell.

### [NIT:consistency] D15 miscounts the rows it says stay unchanged
**Section:** D15
**Issue:** "The existing four `toc` rows and the two compact-view rows stay as T5a wrote them" does not describe `after/INDEX.md`, whose table is two `toc` before→after rows, two compact "no successor" rows, and two `*(none)*` text-view rows.
**Fix:** Restate the preserved rows as they actually are (2 + 2 + 2), since D15's whole claim is that the table is checkable row by row.

### [NIT:consistency] D2's envelope-only list predates D10a
**Section:** D2
**Issue:** D2 says `fail()`/`RenderErrorJSON` are used "only … : usage errors, internal errors, and `expand`'s `*NotATypeError` (D4)", but D10a adds a fourth payload-free envelope path — `expand`'s exit-1 grammar rejection, message `expand <target>: <reason>`. D3's table has it; D2's enumeration is superseded.
**Fix:** Add the D10a path to D2's list so the two sections enumerate the same set.

### [NIT:consistency] D13's exit-1 usage line omits the grammar rejection
**Section:** D13
**Issue:** The proposed line `1  negative answer: not found, outside the repository, ambiguous, or not a type` enumerates four conditions, but D3 also maps an unspellable glyph (`resolve` payload `reason`, `expand` envelope) to exit 1; D13 claims the wording is "true of all three" verbs.
**Fix:** Include the grammar-rejection case in the exit-1 wording, or make the line non-enumerative.

### [NIT:design] Payload `target` for a path result is unstated
**Section:** D8, D7, D10b
**Issue:** For a glyph the payload's `target` is argv verbatim (D8); for a path it is whatever the CLI hands the engine, i.e. `repoRelPath`'s rebased form — so `quarry resolve logger.go` run inside `internal/logger/` echoes `internal/logger/logger.go`, and D7's text line 1 `<Target> <status>` inherits that. It is derivable from D8 + D5 + `cli/doc.go`'s "output is always repository-root relative", but never said; D10b's `resolve ../x` byte-pinned example silently assumes cwd == root.
**Fix:** State that a path result's `target` is the repository-relative form and pin the goldens/examples with that assumption made explicit.

### [NIT:scope] D3's exit-2 row and `repoRelPath`'s own error are unenumerated
**Section:** D3, D8
**Issue:** D3 says a mapping function written from the table "must test the named conditions first and fall through to 3", yet the exit-2 row omits `discoverRoot`'s existing `no repository root found above …; pass --root` usage error; and `repoRelPath`'s own failure (D8: "erroring only on `filepath.Rel`'s own failure", today returning `ErrTargetOutsideRepo`) has no row, so `resolve` would send it to 3 where `toc` sends it to 1.
**Fix:** Add the root-discovery row to exit 2 and state `resolve`'s disposition for a `repoRelPath` error.

## Verdict

APPROVE
Evidence-set bookkeeping leaves three new goldens with no INDEX row or exit code.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 0._
MILL_REVIEW_END
