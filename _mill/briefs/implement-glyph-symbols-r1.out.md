{"status":"success","commit_sha":"d6b52365a36f99dd514c2baa76cfa13584a26c1b","session_id":"2a901edd-2234-4336-ab6c-36792ebbb965","cards_done":[23,24,25,26,27,28,29,30]}

All 8 cards of the `glyph-symbols` batch (04-glyph-symbols.md) are committed and the batch's verify command (`CGO_ENABLED=1 go build ./...` then `CGO_ENABLED=1 go test ./internal/...`) passes clean.

Summary of work:
- Card 23: rewrote `Symbol` in `/home/knatte/Code/quarry/wts/engine-core/internal/engine/answer.go` to carry `glyph.Glyph`/`ID`/`File`/`HeadStart`/`HeadEnd`, dropped `Name`/`Owner`, renamed `Docstring`→`Doc`, widened `Kind` to five values.
- Card 24: `Strategy.Symbols` in `strategy.go` now takes `unit string`.
- Card 25: `unitFor`/`unitSpellable` added in `walk.go`; `fileEntry` threads the unit through and gates `Symbols` on spellability; `toc.go`'s `fileTargetAnswer` updated too (added to the plan's Edits list for this card, committed first).
- Cards 26-28 (combined into one commit, named after card 28 per the plan's combined-commit allowance): `golang.go` now builds glyphs for functions/methods/types, strips a generic receiver's type parameters, skips blank identifiers everywhere, lists package-level `const`/`var`, and lists a named interface's own `method_elem`s with the type's head span.
- A follow-up fix commit corrected `goSpecNames` to drop the `","` separator token the grammar tags with the same `"name"` field (found while building card 30's fixtures).
- Card 29: ported `golang_test.go`, `toc_test.go`, `answer_test.go`, `toc_integration_test.go`, `classify_test.go` onto ID/Doc; also touched `walk_test.go` (added to the plan's scope for this card) since its one `Symbol.Name` assertion and root-placed fixture no longer compiled/worked under the new unit rule.
- Card 30: added `internal/engine/glyph_test.go` and the `testdata/glyphs/` and `testdata/units/` fixtures (including the space-containing directory), covering glyph assignment, blank-identifier exclusion, `init`'s shared id, const/var shapes, interface scope/head-span, and both unspellable-unit entry points.

Two plan amendments were made (each committed and pushed before the corresponding code change), per the brief's "surprise file" protocol: `toc.go` added to card 25, `walk_test.go` added to card 29.

All 8 declared cards are committed; nothing is outstanding.

{"status":"success","commit_sha":"d6b52365a36f99dd514c2baa76cfa13584a26c1b","session_id":"2a901edd-2234-4336-ab6c-36792ebbb965","cards_done":[23,24,25,26,27,28,29,30]}
