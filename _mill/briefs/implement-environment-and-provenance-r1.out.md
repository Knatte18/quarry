All 5 cards (20-24) of batch 05-environment-and-provenance are committed: 4 content commits (20+21 combined into one commit per the card's Commit message, then 22, 23, 24), verify passed (`go test ./bench/loomyard-eval/ladder/...`), and the working tree is clean.

{"status":"success","commit_sha":"d6e33b9e857219aef8192281a97138e31df3fce5","session_id":"f86e220b-368a-4052-ba30-8c102fe54160","cards_done":[20,21,22,23,24]}

Summary of work: implemented all 5 cards of batch 05-environment-and-provenance in `/home/knatte/Code/quarry/wts/ladder-harness/bench/loomyard-eval/ladder/internal/ladder/`:
- `worktree.go` (new): `Cmd`, `Runner`, `ExecRunner`, `ResolveQuarryRepoRoot`, `ResolveLoomyardRepo`, `ResolveWorktreeRoot`, `TaskWorktreePath`, `PrepareWorktree`, `RestoreWorktree`, `WorktreeStatus`, and `AcquireRunLock` (cards 20+21, combined into one commit per the allowance since card 21 edits the file card 20 creates).
- `mcp.go` (new): `MCPConfigDocument`, `WriteMCPConfig`, `BuildServer` (card 22).
- `provenance.go` (new): `Provenance`, `SessionFingerprint`, `Invocation`, `ReadProvenance`, `WriteProvenance`, `CollectInvocation`, `MergeProvenance`, `ScanMemoryPaths`, `WarnOnServerHashDrift`, `CompareFingerprints` (card 23).
- `worktree_test.go`, `mcp_test.go`, `provenance_test.go` (new) (card 24).

`go test ./bench/loomyard-eval/ladder/...` passes; all 4 content commits are pushed to `ladder-harness`. Card count self-check: 4 of 4 expected content commits are present in the range `ace1231..HEAD`, covering all 5 declared cards (20 combined with 21) — genuinely all complete, not a partial report.

{"status":"success","commit_sha":"d6e33b9e857219aef8192281a97138e31df3fce5","session_id":"f86e220b-368a-4052-ba30-8c102fe54160","cards_done":[20,21,22,23,24]}
