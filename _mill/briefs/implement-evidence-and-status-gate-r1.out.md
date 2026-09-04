All 4 cards (14, 15, 16, 17) committed since the batch-start commit `de799f7`, matching the batch's declared 4 cards exactly. Final report:

{"status":"success","commit_sha":"2d26bc42aa3931089af81796654e4e25e60cfa32","session_id":"00cf4797-e12d-4b0d-ace3-d9847a5853c8","cards_done":[14,15,16,17]}

Summary: 4 of 4 cards committed.

- Card 14: `internal/cli/glyph5_test.go` (new) — self-built fixture tree proving all four resolution statuses (found, not_found x2, ambiguous, multipart) plus expand's not-a-type and expand-ambiguous, end to end through `Run`. Commit `1010b9d`.
- Card 15: `internal/cli/after_test.go` — `afterGoldenCase` gained `verb` and `exitCode` fields; assertions became per-row. Commit `d168417`.
- Card 16: eight new files under `docs/research/output-formats/after/` (`resolve-glyph.txt`, `resolve-glyph-text.txt`, `resolve-method.txt`, `resolve-not-found.txt`, `resolve-path.txt`, `expand-type.txt`, `expand-type-text.txt`, `expand-not-a-type.txt`) — generated against the pinned Loomyard checkout (`72c23d9`) following the ordered three-step procedure; the four pre-existing `toc` goldens were confirmed unchanged via `git diff --exit-code`. Commit `a494763`.
- Card 17: `docs/research/output-formats/after/INDEX.md` — rewritten with a total before-to-after mapping table (every before-side file gets a row, every after-side file gets an exit-code cell) and a new "what changed" section for resolve/expand. Commit `2d26bc4`.

Batch verify (`go test ./internal/cli/`) passes both without `LADDER_LOOMYARD_REPO` set (skips the golden comparison, all other tests pass) and with it set (all twelve goldens compare byte-for-byte). `go build ./...` succeeds. Working tree is clean of tracked in-scope modifications.
