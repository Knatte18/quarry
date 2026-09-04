2 of 2 cards committed (both card commit subjects match exactly). Both cards done, verify passed, no uncommitted tracked changes.

{"status":"success","commit_sha":"3f63fe3a25de1b28abf6e8a077e441fbcad64038","session_id":"2f044bfa-2c59-4b0a-8c45-76fa93a026f3","cards_done":[1,2]}

Summary: Implemented batch 01-answer-types in full — 2 of 2 cards committed.

- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/answer.go`: added `Status` type + 4 constants (commit `c84bf3e`), then added `ResolveResult` and `ExpandAnswer` structs with exact fields/tags per spec, re-tensed the file header comment, `Symbol.HeadStart`'s comment, and rewrote `Symbol.File`'s comment to name `symbolsOfUnit`/`Resolve`/`Expand` instead of "batch 5 adds" (commit `3f63fe3`).

`CGO_ENABLED=1 go build ./...` and `CGO_ENABLED=1 go vet ./internal/engine/` both pass. `goimports`/`gofmt` show no drift. Working tree is clean aside from the untracked brief file at `_mill/briefs/implement-answer-types-r1.md`, which is outside this batch's scope.

{"status":"success","commit_sha":"3f63fe3a25de1b28abf6e8a077e441fbcad64038","session_id":"2f044bfa-2c59-4b0a-8c45-76fa93a026f3","cards_done":[1,2]}
