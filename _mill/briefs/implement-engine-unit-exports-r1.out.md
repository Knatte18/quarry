All 5 cards in this batch (6, 7, 8, 9, 10) are complete: cards 6 and 7 were combined into one commit (both touched units.go/walk.go together), card 8 and card 9 each got their own commit, and card 10 (Commit: none, verification-only) was satisfied by re-running the full batch verify suite and confirming the four pinned behaviour-preservation cases plus every walk/toc/resolve/expand/spans/round-trip test pass, with no new import added outside internal/engine's existing dependencies.

Relevant files:
- `/home/knatte/Code/quarry/wts/diff-to-symbols/internal/engine/units.go` (created — `PackageClause`, `UnitsForClauseMap`, `(*Repo).ClauseMapForFiles`)
- `/home/knatte/Code/quarry/wts/diff-to-symbols/internal/engine/walk.go` (edited — `dirPackage` rewired, file header updated)
- `/home/knatte/Code/quarry/wts/diff-to-symbols/internal/engine/units_test.go` (created — `TestPackageClause`, `TestUnitsForClauseMap`, `TestClauseMapForFiles`, plus two supporting tests)

{"status":"success","commit_sha":"965b37e48c8b56c66dd749808274987148c52880","session_id":"dbd92140-50d4-416f-ae31-94db23ae07e3","cards_done":[6,7,8,9,10]}

{"status":"success","commit_sha":"965b37e48c8b56c66dd749808274987148c52880","session_id":"dbd92140-50d4-416f-ae31-94db23ae07e3","cards_done":[6,7,8,9,10]}
