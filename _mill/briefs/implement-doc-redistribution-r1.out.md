Both card commits are present in the range since batch start — 2 of 2 cards committed, matching the batch's declared card count.

{"status":"success","commit_sha":"4101d62bd2a23cc381ee80c1b2290570e088e502","session_id":"4707ae9d-1743-44f6-ab1f-84e603572f28","cards_done":[13,14]}

Summary: 2 of 2 cards committed (both cards had real content commits, both present in git log since batch start).

Relevant files:
- `/home/knatte/Code/quarry/wts/quarry-thin-facade/internal/quarryengine/doc.go` (Card 13 — rewritten five-package overview)
- `/home/knatte/Code/quarry/wts/quarry-thin-facade/internal/quarryengine/lsp/lspclient.go` (Card 14 — package doc comment added)
- `/home/knatte/Code/quarry/wts/quarry-thin-facade/internal/quarryengine/registry/registry.go` (Card 14 — package doc comment added)
- `/home/knatte/Code/quarry/wts/quarry-thin-facade/internal/quarryengine/daemon/doc.go` (Card 14 — new file, package doc comment only)
- `/home/knatte/Code/quarry/wts/quarry-thin-facade/internal/quarryengine/query/refs.go` (Card 14 — package doc comment added, resolvePosition prose relocated)

`verify:` (`go test ./... && go test -tags lsp -run "^$" ./...`) passed clean on both invocations. Working tree has no uncommitted tracked changes.

{"status":"success","commit_sha":"4101d62bd2a23cc381ee80c1b2290570e088e502","session_id":"4707ae9d-1743-44f6-ab1f-84e603572f28","cards_done":[13,14]}
