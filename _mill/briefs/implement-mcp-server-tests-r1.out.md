All 6 cards for this batch are committed, `git status --porcelain --untracked-files=no` is clean (only the untracked brief file exists, which is not tracked/in-scope), and `go test ./internal/mcpserver/...` passes with the full commit range containing exactly 6 card commits matching the 6 declared cards (11-16).

Card-count self-check: 6 of 6 cards committed (cards 11, 12, 13, 14, 15, 16 - commits 71b388f, fa67437, 401f757, 5f62cbc, b3b9ec7, 2534c3e respectively). This is a genuine "all complete" batch.

Key files touched (all under `/home/knatte/Code/quarry/wts/mcp-thin`):
- `internal/mcpserver/testdata/repo/alpha/{doc.go,alpha.go,sub/leaf.go}` - committed fixture repository
- `internal/mcpserver/fixture_test.go` - `fixtureRepoRoot` and `connectedClient` helpers
- `internal/mcpserver/tools_test.go` - `tools/list` shape/schema/prose pin
- `internal/mcpserver/toc_golden_test.go` + `internal/mcpserver/testdata/golden/*.json` - six golden payloads plus CLI-mirror assertion
- `internal/mcpserver/toc_defaults_test.go` - absent-symbols and absent-depth default pins, plus shared `wireFileEntry`/`wireDirAnswer`/`callTOC` helpers
- `internal/mcpserver/toc_depth_test.go` - depth -1 acceptance, depth -2/-7 protocol rejection, and `tocResult` wording pin
- `internal/mcpserver/toc_errors_test.go` - not-found/outside-repo failure envelopes and the never-follow broken-symlink success case

{"status":"success","commit_sha":"2534c3e1ef7661ea78650b82f6fdc62a347a3ff4","session_id":"5bb7ce02-a688-4a1c-af33-e7946e8ac098","cards_done":[11,12,13,14,15,16]}
