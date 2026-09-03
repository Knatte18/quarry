4 of 4 content-commit cards (4, 5, 6, 7) are committed, matching their `Commit:` messages exactly. Card 8 is `Commit: none` and its verification steps (go list -deps, test-file import reads, go.mod diff) were performed and confirmed passing. All 5 cards declared in this batch are complete: 4 of 4 real commits made, plus 1 verification-only card done.

{"status":"success","commit_sha":"63b07a6d805a24cdc93185873163e55b2c79e029","session_id":"8201b83d-5011-4967-bdf4-66ce7722006d","cards_done":[4,5,6,7,8]}

Files touched (all under `/home/knatte/Code/quarry/wts/glyph-package`):
- `glyph/parse.go` (new) — `Parse` and `splitGlyph`
- `glyph/golang.go` (new) — `parseGo`, unit/member validation, `goKeywords`, `isASCIIControl`, `isGoIdentifier`
- `glyph/parse_test.go` (new) — split table, first-`#` test, `rejectCase` type, `parseReject` table
- `glyph/golang_test.go` (new) — `acceptCase`/`goAccept`, `goReject`, completeness/field/Error()/case-sensitivity tests
- `glyph/string_test.go` (edited) — appended `TestRoundTrip_ParseThenString` and `TestRoundTrip_StringThenParse`

{"status":"success","commit_sha":"63b07a6d805a24cdc93185873163e55b2c79e029","session_id":"8201b83d-5011-4967-bdf4-66ce7722006d","cards_done":[4,5,6,7,8]}
