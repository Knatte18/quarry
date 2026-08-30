# Batch: worktree-server-warm

```yaml
task: Port the capability-ladder bench harness to Go
batch: worktree-server-warm
number: 8
cards: 4
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [5]
```

## Batch Scope

Ports the parts of `run_ladder.py` that touch the outside world but not a session: the pinned-worktree
lifecycle, building `quarry-mcp`, the generated MCP server declaration with its environment block, and
the daemon warm-up. The warm-up is rewritten on the Go MCP SDK rather than ported line for line,
because the Python hand-rolled JSON-RPC framing only under a standard-library constraint that does not
exist here.

The external interface the session batch consumes is `EnsureTaskWorktrees`, `BuildWorktree`,
`RestoreWorktree`, `RemoveWorktree`, `BuildServer`, `MCPConfigDocument`, and `Warm`.

Batch-local decision: `Warm` spawns `quarry-mcp` over stdio and issues a real `initialize` plus
`tools/call`, rather than warming through the `quarry` CLI. Warming through the CLI would resolve the
same state directory but exercise a different code path than the one under measurement, which weakens
the warm-versus-cold comparison the matrix exists to draw.

## Cards

### Card 35: Pinned-worktree lifecycle

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/tests/test_run_ladder.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/worktree.go`
  - `bench/loomyard-eval/ladder/internal/ladder/worktree_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `neutralise_worktree` as `NeutraliseWorktree`, `build_worktree` as
  `BuildWorktree`, `restore_worktree` as `RestoreWorktree`, `remove_worktree` as `RemoveWorktree`, and
  `ensure_task_worktrees` as `EnsureTaskWorktrees`, each taking a git runner function so the tests can
  substitute one, mirroring the Python's injected `git` parameter. Port `HarnessError` as an exported
  `HarnessError` type. `RestoreWorktree` performs the reset, the clean, and the re-neutralisation as
  one unit; its doc comment must state that it is unconditional after every attempt, so an attempt that
  edited files cannot leave them for the retry and have the next attempt's dirtiness observation
  attribute them wrongly. Test each function against a fake git runner asserting the exact argument
  vectors, plus `EnsureTaskWorktrees` building a missing worktree and leaving a correctly-pinned one
  alone.
- **Commit:** `feat(ladder): port the pinned-worktree lifecycle`

### Card 36: Building the MCP server binary

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `bench/loomyard-eval/ladder/tests/test_run_ladder.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/server.go`
  - `bench/loomyard-eval/ladder/internal/ladder/server_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `build_server` as `BuildServer(repoRoot string) (string, error)`, returning
  the absolute path of the built binary and keeping the Python's error behaviour when the build fails.
  Use `path/filepath` for every path join rather than a slash-joined literal. Test the failure path
  against a builder function the test substitutes; do not run a real build in the test.
- **Commit:** `feat(ladder): port quarry-mcp build invocation`

### Card 37: The generated MCP server declaration and its environment block

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `bench/loomyard-eval/ladder/internal/ladder/daemon.go`
  - `bench/loomyard-eval/ladder/tests/test_run_ladder.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/server.go`
  - `bench/loomyard-eval/ladder/internal/ladder/server_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `mcp_config_document` as
  `MCPConfigDocument(serverPath, targetDir string) map[string]any`, declaring a single server named
  `quarry` whose command is the built binary's absolute path and whose args carry an explicit
  `--target-dir`. Extend it with an `env` block setting `QUARRY_STATE_DIR` and `QUARRY_BUILD_TAGS` to
  the empty string and leaving `QUARRY_CONFIG` untouched — this is the first of the three points the
  environment scrub now applies at, and the doc comment must record both that and the open question of
  whether such a block replaces or augments the inherited environment. Test that the document names one
  server, carries the target-dir argument, empties both scrubbed keys, and mentions no other
  environment key.
- **Commit:** `feat(ladder): generate the MCP server declaration with a scrubbed env block`

### Card 38: Daemon warm-up over the Go MCP SDK

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/run_ladder.py`
  - `bench/loomyard-eval/ladder/internal/ladder/daemon.go`
  - `internal/mcpserver/stdio_lsp_test.go`
  - `bench/loomyard-eval/ladder/tests/test_run_ladder.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/warm.go`
  - `bench/loomyard-eval/ladder/internal/ladder/warm_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `WARM_UP_TOOL` as an exported `WarmUpTool` constant still holding
  `workspace_symbol`, `_WARM_UP_TIMEOUT_S` as an unexported timeout, and `warm_daemon` as
  `Warm(serverPath, targetDir string, env []string, cacheDir string) error`. Replace the hand-rolled
  JSON-RPC framing with a stdio client built on `github.com/modelcontextprotocol/go-sdk/mcp`, already a
  direct module dependency — `internal/mcpserver/stdio_lsp_test.go` is the in-repo example of driving
  that client over a spawned server. `Warm` performs the initialize handshake and one tool call, then
  asserts its post-condition: a daemon state file now exists at the resolved state directory. The
  spawned server receives the environment passed in, which callers supply from the scrub. Test the
  post-condition failure path with a fake that starts no daemon; do not spawn a real server in the
  test.
- **Commit:** `feat(ladder): warm the daemon over the Go MCP SDK`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` covers `worktree_test.go`, `server_test.go`, and
`warm_test.go` plus every other test file in the ladder subtree. Every test in this batch substitutes
the process-spawning seam rather than running git, a build, or a server for real — the same division
the Python suite drew.
