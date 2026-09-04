All 5 commits match all 5 cards (14, 15, 16, 17, 18) declared in the batch file. 5 of 5 cards committed - complete.

Summary of work: Implemented the `expand` verb for the resolve+expand (T4) task, batch 4.

- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/expand.go` (new): `NotATypeError` typed failure and `Repo.Expand`/`Repo.expand` implementing the head/members/disposition table per the batch spec.
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/expand_test.go` (new): `TestExpand_Struct`, `TestExpand_Interface`, `TestExpand_TypeWithoutMembers`, `TestExpand_NotAType`, `TestExpand_AmbiguousBuildTags`, `TestExpand_MalformedTarget`, `TestExpand_NotFound`, `TestExpand_Collision`.

All 5 cards (14-18) committed, one commit each, matching their `Commit:` messages. `CGO_ENABLED=1 go test ./internal/engine/` and the module-wide `CGO_ENABLED=1 go vet ./...` both pass cleanly. Working tree is clean.

{"status":"success","commit_sha":"66a18a8a1e9e88efbc5b01874415c3c8c9cbab32","session_id":"3c2d2fcc-e352-45e7-ac4a-346a3105ec70","cards_done":[14,15,16,17,18]}
