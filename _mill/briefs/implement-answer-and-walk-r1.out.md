Summary: all 10 of 10 cards for batch `answer-and-walk` (cards 13-22) are committed, verify passes (`CGO_ENABLED=1 go build ./...` and `CGO_ENABLED=1 go test ./internal/...` both green), and the working tree has no uncommitted tracked changes.

Key files touched:
- `/home/knatte/Code/quarry/wts/engine-core/internal/engine/answer.go` — new `DirAnswer`/`FileEntry`/`TOCOptions`/`DepthAll`, replacing `FileTOC`/`DirTOC`/`DirEntry`.
- `/home/knatte/Code/quarry/wts/engine-core/internal/engine/strategy.go` — `Strategy` gains `PackageDoc`, `Generated`/`TestFile` drop the `known` return.
- `/home/knatte/Code/quarry/wts/engine-core/internal/engine/golang.go` — `PackageDoc` implementation, signature narrowing.
- `/home/knatte/Code/quarry/wts/engine-core/internal/engine/repo.go` — new: `Repo`, `Open`, `resolveTarget`, `ErrTargetOutsideRepo`, `ErrTargetNotFound`.
- `/home/knatte/Code/quarry/wts/engine-core/internal/engine/walk.go` — new: `dirPackage`, `dirDoc`, `fileEntry`, `walkDir`.
- `/home/knatte/Code/quarry/wts/engine-core/internal/engine/toc.go` — rewritten around `Repo.TOC`, `fileTargetAnswer`, `ancestorChain`, `splitDirBase`.
- `/home/knatte/Code/quarry/wts/engine-core/internal/engine/text.go` — `FirstParagraph` doc comment fix.
- Fixture trees under `/home/knatte/Code/quarry/wts/engine-core/internal/engine/testdata/{tree,broken,tiebreak}/`.
- Test files: `toc_test.go`, `toc_integration_test.go`, `golang_test.go`, `classify_test.go` (ported/adjusted), plus new `repo_test.go`, `walk_test.go`, `answer_test.go`.

Final HEAD: `8594cdef8bb8777330e108cc41db4c2e651a36ab` on branch `engine-core`, pushed to `origin/engine-core`.

{"status":"success","commit_sha":"8594cdef8bb8777330e108cc41db4c2e651a36ab","session_id":"4fa63e51-258a-486d-8df8-86176c6ffdc8","cards_done":[13,14,15,16,17,18,19,20,21,22]}
