No tracked in-scope modifications remain. Verify command passed cleanly. All 3 cards (14, 15, 16) declared in the batch file have been committed, matching the 3-card count.

All 3 of 3 cards in this batch are committed: card 14 (2b6d70a), card 15 (d8e3983), card 16 (988fe76). Verify (`go test ./internal/engine/ -run 'TestRoundTrip'`) passed, with the two Loomyard-gated tests skipping as expected on this machine (no `LADDER_LOOMYARD_REPO`), matching the batch's documented green signal via `TestRoundTrip_QuarryItself`.

Files touched:
- `/home/knatte/Code/quarry/wts/glyph-maker/internal/engine/roundtrip_test.go` (card 14)
- `/home/knatte/Code/quarry/wts/glyph-maker/internal/engine/golden_test.go` (card 15)
- `/home/knatte/Code/quarry/wts/glyph-maker/internal/engine/naming_roundtrip_test.go` (card 16, new file)

{"status":"success","commit_sha":"988fe76729e4d6c9e8fbd0c50849fbf834244c2b","session_id":"74af55cc-db90-449d-9344-44aa1e03b206","cards_done":[14,15,16]}