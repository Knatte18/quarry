All 5 cards committed with matching commit-message subjects (cards 18-22), and the batch's declared 5 cards are all accounted for. Verify (`go test ./internal/cli/`) passes.

5 of 5 cards committed. All cards (18, 19, 20, 21, 22) are complete: `--build-tags`/`$QUARRY_BUILD_TAGS` on all four verbs, the `tags-<hex>` state-directory segment, `assert-no-callers` rewired onto `quarry.Callers` with `--no-verify`, the help-text rewrite, and the new `internal/cli` behaviour tests. `go test ./internal/cli/` and `go vet -tags lsp ./...` both pass, and `git status --porcelain --untracked-files=no` shows no dirty tracked files.

Relevant files:
- `internal/cli/paths.go`
- `internal/cli/cli.go`
- `internal/cli/paths_test.go`
- `internal/cli/resolve_test.go`
- `internal/cli/cli_test.go`

{"status":"success","commit_sha":"6633ba052100ae7c289c9203258cd7e4062b2179","session_id":"f787200a-ca06-4f82-8ced-def0966015b0","cards_done":[18,19,20,21,22]}
