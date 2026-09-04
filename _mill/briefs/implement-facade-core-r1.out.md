All 4 cards from the batch file (Cards 1–4) match these 4 commits exactly. 4 of 4 cards committed — all complete.

{"status":"success","commit_sha":"7ab2727a3877abd088d0a70b67fd3f95c96e0c5f","session_id":"48778a4b-4ab9-4c53-a7a3-16943a7de8da","cards_done":[1,2,3,4]}

Summary of work: implemented batch 1 (`facade-core`) of the Facade + CLI, toc (T5a) plan in full — all 4 cards committed, verify green.

Files created:
- `/home/knatte/Code/quarry/wts/facade-cli-toc/quarry/doc.go` — package doc for the public facade
- `/home/knatte/Code/quarry/wts/facade-cli-toc/quarry/quarry.go` — type aliases (`DirAnswer`, `FileEntry`, `Symbol`, `Kind`, `TOCOptions`), `Kind` constants, `DepthAll`, and error sentinels aliasing `internal/engine`
- `/home/knatte/Code/quarry/wts/facade-cli-toc/quarry/repo.go` — `Repo`, `Open`, and `(*Repo).TOC` delegating to the engine
- `/home/knatte/Code/quarry/wts/facade-cli-toc/quarry/scratchtree_test.go` — package-local `writeScratchTree` helper (two `filepath.Dir` steps, per the plan's decision)
- `/home/knatte/Code/quarry/wts/facade-cli-toc/quarry/repo_test.go` — covers `Open` (relative/missing/file/valid root), `TOC` on directory and file targets, sentinel transitivity via `errors.Is` against the facade's own sentinels, and compile-time alias-identity assertions

`go test ./quarry/...` and `go build ./...` both pass. No files outside the batch's declared scope were touched. Working tree is clean (no uncommitted tracked changes).

{"status":"success","commit_sha":"7ab2727a3877abd088d0a70b67fd3f95c96e0c5f","session_id":"48778a4b-4ab9-4c53-a7a3-16943a7de8da","cards_done":[1,2,3,4]}
