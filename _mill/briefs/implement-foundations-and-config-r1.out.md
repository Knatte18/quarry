All 7 cards match all 7 declared `Commit:` messages in order. 7 of 7 cards committed, all complete.

{"status":"success","commit_sha":"d611915ebfb8adae87fe0ec6b6d5f87af72ec0b6","session_id":"a6ceedc2-7fca-4777-a777-5f38fdcd9817","cards_done":[1,2,3,4,5,6,7]}

Summary: all 7 cards of batch 01-foundations-and-config were committed and verified — 7 of 7 cards complete, `go test ./bench/loomyard-eval/ladder/...` passes, `go build ./...` passes, and the working tree is clean.

Key files touched:
- `/home/knatte/Code/quarry/wts/ladder-harness/go.mod`, `go.sum` — added `gopkg.in/yaml.v3` as a direct dependency
- `/home/knatte/Code/quarry/wts/ladder-harness/bench/loomyard-eval/ladder/.gitignore` — new, `results/*/raw/`
- `/home/knatte/Code/quarry/wts/ladder-harness/bench/loomyard-eval/ladder/internal/ladder/match.go` — new, shared token matcher
- `/home/knatte/Code/quarry/wts/ladder-harness/bench/loomyard-eval/ladder/internal/ladder/config.go` — new, `Ladder`/`Task`/`Config` types, `LoadLadder`, accessors, `BuiltinTools`, and full `validate()` including retired-key rejection
- `/home/knatte/Code/quarry/wts/ladder-harness/bench/loomyard-eval/ladder/ladder-toc.yaml` — migrated to the new shape (4 cells, `toc` tool, `server` block)
- `/home/knatte/Code/quarry/wts/ladder-harness/bench/loomyard-eval/ladder/ladder.yaml`, `ladder-compact.yaml`, `ladder-annex.yaml`, `ladder-task05.yaml`, `ladder-followup.yaml` — deleted
- `/home/knatte/Code/quarry/wts/ladder-harness/bench/loomyard-eval/ladder/internal/ladder/config_test.go`, `match_test.go` — new test coverage

Last commit SHA: `d611915ebfb8adae87fe0ec6b6d5f87af72ec0b6` (all commits pushed to `ladder-harness` branch).

{"status":"success","commit_sha":"d611915ebfb8adae87fe0ec6b6d5f87af72ec0b6","session_id":"a6ceedc2-7fca-4777-a777-5f38fdcd9817","cards_done":[1,2,3,4,5,6,7]}
