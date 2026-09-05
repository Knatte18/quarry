All 9 cards committed, matching the batch's declared 9 cards, and verify (`go test ./glyph/...`) passes.

9 of 9 cards committed — all cards implemented and verified.

Relevant files (all under `/home/knatte/Code/quarry/wts/glyph-self-form`):
- `glyph/parse.go` — count check for multiple `#`, Detail on no-separator arm, `splitGlyph` doc update
- `glyph/errors.go` — `ReasonMultipleSeparators` added, `ReasonMemberEmpty` removed, rewritten reason text/doc comments
- `glyph/golang.go` — empty member accepted as self form in `parseGo`, `checkGoMember`'s empty branch removed, `checkGoUnit` doc updated
- `glyph/glyph.go` — `IsSelf()` predicate, `Glyph` doc comment naming `Self`
- `glyph/doc.go` — widened package summary
- `glyph/self.go` (new) — `Self(lang, path)` compose constructor
- `glyph/parse_test.go`, `glyph/golang_test.go`, `glyph/string_test.go`, `glyph/self_test.go` (new) — updated/added test tables

{"status":"success","commit_sha":"793738cc2d5f4bc537cae80383c16588baab7492","session_id":"1d3d71aa-5ef7-45c3-b55e-c9bfd4fb7cc6","cards_done":[1,2,3,4,5,6,7,8,9]}
