Summary: All 4 cards of the cli-pipelines batch are committed (4 of 4 — verified count, not an unqualified claim). Files touched:

- `/home/knatte/Code/quarry/wts/facade-cli-resolve-expand/internal/cli/cli.go` — added `codeForResolveResult`, `codeForExpandAnswer`, `codeForExpandError`; restructured `Run` into a shared-steps dispatch over `runTOC`, `runResolve`, `runExpand`.
- `/home/knatte/Code/quarry/wts/facade-cli-resolve-expand/internal/cli/cli_test.go` — table tests for the three new mapping functions, extended the shared pipeline fixture (`pkg/other` with `Make`, `Widget.Value`, `Empty`), and end-to-end `TestRun_Resolve` / `TestRun_Expand` suites.
- `/home/knatte/Code/quarry/wts/facade-cli-resolve-expand/internal/cli/doc.go` — package doc now describes all three verbs, the `#`-classification rule, the `ok`-key semantics, and the recorded reclassification contract gap.

`go build ./...` and `go test ./internal/cli/` (the batch's declared verify command) both pass cleanly. No uncommitted tracked changes remain.

{"status":"success","commit_sha":"387425903869650bd2a09416e03a59f7782b239c","session_id":"3aa42928-76bb-415d-9702-335d6d5347b0","cards_done":[10,11,12,13]}
