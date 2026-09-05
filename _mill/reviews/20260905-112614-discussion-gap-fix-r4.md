# Discussion fix — round 4

Source review: `_mill/reviews/20260905-112357-discussion-review-r4.md` (verdict REQUEST_CHANGES, reviewer `opushigh`, 1 BLOCKING / 5 NIT, 3 demoted from BLOCKING).

All six findings verified against the worktree before fixing. Under `--orch` the decision tree's PUSH BACK path is unavailable; every finding was treated as FIX.

## Fixed

- "[BLOCKING:design] `<unit>_test#` has no stated engine disposition": verified `unitDirs` (`internal/engine/resolve.go:55–63`) resolves a `_test` unit by stripping the suffix and finding the base directory, so `internal/logger_test#Foo` resolves while `internal/logger_test#` is a path existing nowhere. Added **D10a** deciding both halves: (a) the self form resolves through the plain path lookup, never `unitDirs`; (b) `Unit` **is** set on a self-glyph `not_found` by the same `unitDirs` rule every other glyph uses — reversing D10's suppression, which is deleted rather than softened. Recorded why the original rationale held for `internal/logger#` and failed for `internal/logger_test#`, and required the asymmetry to be documented in `docs/glyph.md` §2 (D15) rather than hidden. Testing gains the `_test` engine row asserting `unit: found` beside the still-resolving `<unit>_test#Foo`, noting the existing `testdata/foo_test/` fixture, plus a CLI exit-code row.

- "[NIT:scope] `docs/glyph.md` §6's Go API list is not in the edit inventory": verified glyph.md:204–210 enumerates `Language`, `Glyph`, `Parse`, `String()` only, and glyph.md:9 makes that enumeration the contract boundary. D15's §6 entry split into two edits, (b) adding `Self(lang Language, path string) (Glyph, error)` and `Glyph.IsSelf() bool` with signatures — without which task item 5's own function sits outside the contract.

- "[NIT:scope] D7's \"four paragraphs become false\" undercounts `internal/cli`": verified `flags.go:49–52` states the expand-gate property "its fixture-free table test rests on", and `cli.go:229–231` documents the expand `*ParseError` message as "the value's reason word". D7's inventory raised from four to six, both named with what falsifies them, plus an explicit note that the enumeration is asserted complete.

- "[NIT:consistency] Two `glyph/` doc comments misplaced or unlisted": verified the "exports no constructor and no Validate method" sentence is at `glyph/glyph.go:17–19` on `Glyph`'s doc comment, and that `glyph/doc.go` carries no such sentence — an edit aimed there would have changed nothing and left the false claim standing. Corrected in both Scope-In and D20. Also added `ParseError.Detail`'s doc comment (`glyph/errors.go:99–100`) to the inventory, falsified by D4 putting a suggested spelling rather than an offending fragment in `Detail`.

- "[NIT:decision] `TestIsGlyphTarget` has no stated disposition": verified at `internal/engine/resolve_test.go:329`. Named as deleted with the function, in the same disposition style the `repopath` tests use.

- "[NIT:design] D14's listing branch drops two guards the arm it replaces has": verified the default arm guards both a nil `Dir` and `Status != StatusFound`. D14 now keeps the `StatusFound` guard explicitly and falls back to `r.Target` when `ID` is empty, so line 1 can never begin with a space; two guard rows added to `quarry/text_test.go`.

## Pushed Back

None.
