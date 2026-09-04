All 9 of 9 cards committed this turn (cards 5-9 in `0bca441`, cards 10-13 in `15efb0c`). Verify (`CGO_ENABLED=1 go test ./internal/engine/`) passes, plus module-wide `go vet ./...` is clean, and the tree has no dirty tracked files.

Relevant files:
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/resolve.go` - added `unitDirsResult`, `unitMemo`/`newUnitMemo`/`symbolsOf`/`dirsOf`, `isGlyphTarget`, `statusForMatches`, `matchesFor`, `resolveGlyphTarget`, `resolvePathTarget`, `Resolve`/`resolve`, plus the five comment retensings.
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/resolve_test.go` - added `TestStatusForMatches`, `TestIsGlyphTarget`, `TestResolve_Found`, `TestResolve_Multipart`, `TestResolve_AmbiguousBuildTags`, `TestResolve_NotFoundBothWays`, `TestResolve_AmbiguousCollision`, `TestResolve_CandidatesOrdered`, `TestResolve_ParsesEachUnitOnce`, `TestResolve_UnitDirectoryMissingIsNotAnError`, `TestResolve_ReadFailureFailsTheCall`, `TestResolve_ArgumentOrderAndArity`, `TestResolve_PathTargets`, `TestResolve_MalformedGlyphEntries`, plus header-comment update; reused `marshalToMap` from `answer_test.go` rather than redeclaring it.

{"status":"success","commit_sha":"15efb0c9d0543607887f9af380e3f1f3da89850c","session_id":"bbc75a40-2495-4c1d-ad39-0f582af7d770","cards_done":[5,6,7,8,9,10,11,12,13]}
