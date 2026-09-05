MILL_REVIEW_BEGIN
# Review: Glyph self-form and the resolve contract (C1)

```yaml
duration_s: 305.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (session metadata; best-effort self-assessment)
reviewed_file: _mill/discussion.md
date: 2026-09-05
```

## Findings

### [BLOCKING:design] `<unit>_test#` has no stated engine disposition
**Section:** D10 / Testing (`internal/engine/resolve_test.go`)
**Issue:** D10 routes a self glyph to the path helper with `g.Unit`, but `unitDirs` (resolve.go:55–63) strips `_test` and looks up the base directory, so `internal/logger_test#Foo` resolves while `internal/logger_test#` becomes `not_found` on a path that never exists — and D10 suppresses `Unit`, so the answer cannot even say the unit exists; Testing carries an `_test` row in the parse table and in `Self`'s table but none in the engine table.
**Fix:** state whether the self form resolves through `unitDirs` (the unit lookup) or through the plain path lookup, and pin the `_test` self glyph as an engine row.

### [NIT:scope] `docs/glyph.md` §6's Go API list is not in the edit inventory
**Demoted-from:** BLOCKING
**Section:** D15 (§6) / D20
**Issue:** §6 (glyph.md:204–210) enumerates the package's exported API — `type Language`, `type Glyph struct{…}`, `Parse`, `Glyph.String()` — and glyph.md:9 says "Anything not stated here is not part of the contract"; D20's `Self` and D1's `IsSelf()` are therefore outside the contract, yet D15's only §6 edit is the trailing-`#`-in-formats sentence.
**Fix:** add the §6 API-list edit (both new symbols, with `Self`'s signature) to D15's six-site inventory.

### [NIT:scope] D7's "four paragraphs become false" undercounts `internal/cli`
**Demoted-from:** BLOCKING
**Section:** D7 / D19
**Issue:** `internal/cli/flags.go:49–52` (parseArgs' doc comment) states "for `expand` specifically, a target containing no `#` is rejected here … which is the property its fixture-free table test rests on", falsified by D19 and not in any inventory (flags.go appears only for the line-153 code deletion); `internal/cli/cli.go:229–231` (Run step 3) documents the expand `*ParseError` message as "the value's reason word", which D19 replaces, while D7(iv) names only the step-1 and "already guaranteed a `#`" sentences of that comment.
**Fix:** raise the count and name both sites, since D7 asserts the enumeration is complete.

### [NIT:consistency] Two `glyph/` doc comments misplaced or unlisted
**Demoted-from:** BLOCKING
**Section:** Scope (In, `glyph/`) / D20 / D4
**Issue:** the sentence "this package exports no constructor and no Validate method" lives at `glyph/glyph.go:17–19` (Glyph's doc comment), not `glyph/doc.go` as both Scope and D20 state — doc.go holds no such sentence; separately, `ParseError.Detail`'s doc comment (`glyph/errors.go:99–100`, "carries the offending segment, component or rune where one exists") is falsified by D4 putting the suggested `s + "#"` spelling in `Detail`, and is listed nowhere.
**Fix:** correct the file attribution and add the `Detail` doc-comment edit to the inventory.

### [NIT:decision] `TestIsGlyphTarget` has no stated disposition
**Section:** Testing (`internal/engine/resolve_test.go`)
**Issue:** `internal/engine/resolve_test.go:329` tables the `isGlyphTarget` function D4 deletes; the discussion gives explicit dispositions for the `RepoRelPath` tests and the flags gate tests but is silent on this one.
**Fix:** name it as deleted, in the same sentence style the repopath test disposition uses.

### [NIT:design] D14's listing branch drops two guards the arm it replaces has
**Section:** D14
**Issue:** the new branch prints `r.ID + " " + status` then the block whenever `r.Listing != nil`, while today's default arm guards both a nil `Dir` and `Status != StatusFound` for externally built values; a hand-built value with a `Listing` and no `ID` emits a line beginning with a space.
**Fix:** say whether the new branch keeps the `StatusFound` guard and what it prints when `ID` is empty.

## Verdict

REQUEST_CHANGES
Self form's `_test` unit case undecided; three doc/API edit sites missing from the inventory.
_Note: 3 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 1._
MILL_REVIEW_END
