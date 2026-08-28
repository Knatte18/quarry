All 7 cards committed since the batch start commit, matching all 7 declared cards in the batch file (`### Card 1` through `### Card 7`). 7 of 7 cards committed — all complete.

{"status":"success","commit_sha":"a7a3d9289201f3f4776fc433062e88927349cc49","session_id":"3a5a1371-3269-4d83-9202-2feb12c1e3ac","cards_done":[1,2,3,4,5,6,7]}

Summary: All 7 cards of the treesitter-backend batch (batch 1) were implemented and committed, 7 of 7 cards done. Files touched:

- `/home/knatte/Code/quarry/wts/toc-verbs/go.mod`, `/home/knatte/Code/quarry/wts/toc-verbs/go.sum` — pinned the 6 tree-sitter modules at exact versions
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/cgoguard.go`, `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/cgoguard_nocgo.go` — new CGO_ENABLED=0 compile-time guard
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/treesitter/treesitter.go` — new parsing backend (`Supported`, `Languages`, `WithTree`)
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/treesitter/treesitter_test.go` — new tests
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/registry/extension.go` — new file-extension language map
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/registry/extension_test.go` — new tests
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/layering_test.go` — added treesitter rows

Batch verify command (`go test ./internal/quarryengine ./internal/quarryengine/treesitter ./internal/quarryengine/registry`) passes. Working tree is clean; all commits pushed.

{"status":"success","commit_sha":"a7a3d9289201f3f4776fc433062e88927349cc49","session_id":"3a5a1371-3269-4d83-9202-2feb12c1e3ac","cards_done":[1,2,3,4,5,6,7]}
