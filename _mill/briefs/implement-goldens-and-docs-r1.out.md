All 5 cards (14 through 18) were already implemented and committed in this branch's history before this dispatch began — the earlier "blocked" cycle was resolved by merging the missing `naming-counts.json` golden from `main`, and a fresh `mill-go: start batch` marker was then created without any new content commits since. I independently re-verified every card's requirements this turn:

- Card 14 (commit `6221d93`): five golden table rows in `internal/cli/after_test.go`, header comment updated to twenty files / four verbs — confirmed present.
- Card 15 (commit `780bc9b`): the five golden files exist under `internal/cli/testdata/`, `go test ./internal/cli/ -run TestAfterGoldens -v` shows all 20 cases passing including the five new ones, and `go test ./internal/engine/ -run Loomyard -v` shows all Loomyard cases (including `TestRoundTrip_LoomyardNaming`, previously blocked on a missing golden) executing and passing.
- Card 16 (commit `8a8430e`): `internal/cli/testdata/INDEX.md` has the five new rows and updated counts (twenty files, four verbs).
- Card 17 (commit `4ddfc6c`): `docs/rewrite-plan.md` section 5 has `[--view full|glyphs]` plus "The view mechanism." and "The preset rule." paragraphs.
- Card 18 (commit `af44239`): `docs/roadmap.md` no longer contains point 2a.

Full verify (`LADDER_LOOMYARD_REPO="$PWD/.scratch/loomyard-pin" go test ./internal/cli/ ./internal/engine/`) passes. `git status --porcelain --untracked-files=no` shows only the dispatch brief files as dirty (out of batch scope, not touched by me). I made no new commits this turn — 5 of 5 cards confirmed already complete, not "all complete" as a fresh claim but as a verified re-check of prior work.

Relevant paths: /home/knatte/Code/quarry/wts/glyphs-verb/internal/cli/after_test.go, /home/knatte/Code/quarry/wts/glyphs-verb/internal/cli/testdata/INDEX.md, /home/knatte/Code/quarry/wts/glyphs-verb/internal/cli/testdata/glyphs-dir.txt, /home/knatte/Code/quarry/wts/glyphs-verb/internal/cli/testdata/glyphs-dir-text.txt, /home/knatte/Code/quarry/wts/glyphs-verb/internal/cli/testdata/glyphs-file.txt, /home/knatte/Code/quarry/wts/glyphs-verb/internal/cli/testdata/glyphs-file-text.txt, /home/knatte/Code/quarry/wts/glyphs-verb/internal/cli/testdata/toc-view-glyphs-depth.txt, /home/knatte/Code/quarry/wts/glyphs-verb/docs/rewrite-plan.md, /home/knatte/Code/quarry/wts/glyphs-verb/docs/roadmap.md.

{"status":"success","commit_sha":"4ee2c8b9f3f3b7780c5d1ed1ca00d8a97718da73","session_id":"40a970d6-b2ae-458e-89b3-b80abbde3380","cards_done":[14,15,16,17,18],"already_complete":true}
