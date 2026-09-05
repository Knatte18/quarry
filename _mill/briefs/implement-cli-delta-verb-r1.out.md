All 6 of 6 cards (42-47) in this batch are committed. Verify command `go test ./internal/cli/ -run 'TestParseArgs|TestRun|TestCodeFor'` passes, and the full repository test suite (`go build ./... && go vet ./... && go test ./...`) is green.

Key files touched:
- `/home/knatte/Code/quarry/wts/diff-to-symbols/internal/cli/flags.go` - parses the `delta` verb and its `--from`/`--to` flags
- `/home/knatte/Code/quarry/wts/diff-to-symbols/internal/cli/usage.go` - usage text for the fourth verb
- `/home/knatte/Code/quarry/wts/diff-to-symbols/internal/cli/cli.go` - `runDelta` pipeline and `codeForDeltaError` mapping
- `/home/knatte/Code/quarry/wts/diff-to-symbols/internal/repopath/target.go` - doc-comment corrections naming the two path-taking verbs
- `/home/knatte/Code/quarry/wts/diff-to-symbols/internal/cli/doc.go` - package doc updated for the fourth verb
- `/home/knatte/Code/quarry/wts/diff-to-symbols/internal/cli/flags_test.go` - new/updated parser tests
- `/home/knatte/Code/quarry/wts/diff-to-symbols/internal/cli/cli_test.go` - new end-to-end delta tests, including the fine-grained "unreadable loose blob" fixture technique

{"status":"success","commit_sha":"111f63c85c2dcaa6c1452dbba3e1ed102a0e303b","session_id":"b33b4ab3-0279-4e48-8ebe-45aa1b809d53","cards_done":[42,43,44,45,46,47]}
