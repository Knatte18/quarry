All 4 declared cards' commit messages are present in the log (plus 2 corrective fixup commits I made to fix my own staging/text mistakes, which are honestly documented). 4 of 4 cards committed and verified. Batch verify (`go test ./internal/repopath/... ./internal/cli/...`) passes, and module-wide `go vet ./...` is clean.

{"status":"success","commit_sha":"0fb496ad8fc5697e8da5afe0151e66db0e74541e","session_id":"e4741755-1d56-4be4-a80d-1d36bd6a386f","cards_done":[1,2,3,4]}

Summary: All 4 cards in batch 01-repopath-extraction are committed (4 of 4 complete). Two additional corrective commits (`c53e631`, `b26e4f9`) were needed to fix staging/content mistakes I made while implementing card 1 (I had forgotten to re-stage edited file content before the first commit, and had to restore the "pass --root" substring in the sentinel text that card 3's tests depend on) — both are honestly recorded in the git history rather than squashed away. Batch verify (`go test ./internal/repopath/... ./internal/cli/...`) and the module-wide `go vet ./...` both pass.

Key files touched:
- `/home/knatte/Code/quarry/wts/mcp-thin/internal/repopath/root.go` (new, moved from internal/cli/root.go)
- `/home/knatte/Code/quarry/wts/mcp-thin/internal/repopath/target.go` (new, moved from internal/cli/target.go)
- `/home/knatte/Code/quarry/wts/mcp-thin/internal/repopath/doc.go` (new)
- `/home/knatte/Code/quarry/wts/mcp-thin/internal/repopath/root_test.go`, `target_test.go`, `scratchtree_test.go` (new/moved)
- `/home/knatte/Code/quarry/wts/mcp-thin/internal/cli/cli.go` (calls repopath, adds `rootUsageMessage`)
- `/home/knatte/Code/quarry/wts/mcp-thin/internal/cli/doc.go` (corrected package doc)
- `/home/knatte/Code/quarry/wts/mcp-thin/internal/cli/scratchtree_test.go` (header sentence fix)
- `/home/knatte/Code/quarry/wts/mcp-thin/internal/cli/message_test.go` (new, exact-string test)

{"status":"success","commit_sha":"0fb496ad8fc5697e8da5afe0151e66db0e74541e","session_id":"e4741755-1d56-4be4-a80d-1d36bd6a386f","cards_done":[1,2,3,4]}
