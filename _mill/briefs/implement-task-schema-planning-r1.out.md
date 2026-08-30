All 4 cards committed, matching the batch's 4 declared cards exactly: 4 of 4 cards committed. Verify (`go test ./bench/loomyard-eval/ladder/...`) passed.

{"status":"success","commit_sha":"6b3f2edd8f4e829193c63e2f12488d4cd30e58eb","session_id":"8efe05e2-5f60-4885-9645-fe64bc92fa5f","cards_done":[39,40,41,42]}

Summary: Implemented all 4 cards in batch 09-task-schema-planning:
- `bench/loomyard-eval/ladder/internal/ladder/task.go` (new) - `TaskTextHeading`, `TaskTextFor`, `firstFencedJSONBlock`, `section`, schema constants, `SchemaFor`.
- `bench/loomyard-eval/ladder/internal/ladder/task_test.go` (new) - tests reading the real committed task 01/04 files and README.
- `bench/loomyard-eval/ladder/internal/ladder/plan.go` (new) - `PlanRuns`, `MainRuns`, `ColdRuns`, `SessionDir`.
- `bench/loomyard-eval/ladder/internal/ladder/plan_test.go` (new) - enumeration and session-directory tests.

`go test ./bench/loomyard-eval/ladder/...` passes. 4 of 4 cards committed (39, 40, 41, 42), all pushed to `port-ladder-bench-to-go`. Working tree is clean.

{"status":"success","commit_sha":"6b3f2edd8f4e829193c63e2f12488d4cd30e58eb","session_id":"8efe05e2-5f60-4885-9645-fe64bc92fa5f","cards_done":[39,40,41,42]}
