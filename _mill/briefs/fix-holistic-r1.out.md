Both findings from the review were fixed and committed:

1. **BLOCKING:design** — Provenance write deferred past two abort paths. Fixed by moving `WriteProvenance` to run immediately after `MergeProvenance` succeeds (before the memory-path scan and `BuildServer`), and re-writing provenance once the server hash is known. File: `/home/knatte/Code/quarry/wts/ladder-harness/bench/loomyard-eval/ladder/internal/ladder/run.go`. Commit `d88ce3f77bfed38ef9fd55e08ef43542b7ab95c3`.

2. **NIT:consistency** — Unused `cfg` parameter in `redactedAnswerText` and `writeAnswerFiles`. Dropped the dead parameter from both functions and their call site. Same file. Commit `039e72b04583c2859559f7afeebb51ec2bbf4bab`.

Both changes verified with `go build ./...` and `go test ./bench/loomyard-eval/ladder/...` (all pass, exit 0). No new test was mandated by either finding. `git status --porcelain --untracked-files=no` shows no outstanding tracked modifications. Baseline HEAD was `05598b2d2f6b3052d3298f5d986756fe96fc7991`; current HEAD `039e72b04583c2859559f7afeebb51ec2bbf4bab` differs.

{"status":"success","commit_sha":"039e72b04583c2859559f7afeebb51ec2bbf4bab","session_id":"8d9fb2fb-4532-4a87-874c-d36e3f5ccbbe"}

{"status":"success","commit_sha":"039e72b04583c2859559f7afeebb51ec2bbf4bab","session_id":"8d9fb2fb-4532-4a87-874c-d36e3f5ccbbe"}
