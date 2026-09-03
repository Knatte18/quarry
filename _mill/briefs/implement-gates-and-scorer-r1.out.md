All 4 cards (16, 17, 18, 19) have matching commits in the log — 4 of 4 cards committed, all complete.

Summary of work: implemented batch `04-gates-and-scorer` in full.

- `/home/knatte/Code/quarry/wts/ladder-harness/bench/loomyard-eval/ladder/internal/ladder/gates.go` (Card 16) — `Finding` struct plus `CheckGrantedToolUsed` (gate 1), `CheckBlinding` (gate 2, checks a/b/c), `CheckRenderedControlPrompt` (check d), `CheckWorktreeDirtied`.
- `/home/knatte/Code/quarry/wts/ladder-harness/bench/loomyard-eval/ladder/internal/ladder/score.go` (Card 17) — ported `ExplorationRule`/`ImpactRule`/`StripFasitMeta`/`scoreFieldsFromRule`/`ParseScorerReply` from V1; added `RedactAnswer`/`RedactionInput`, `BuildScorerPrompt`, `ScoreRecord`/`UnscoredRecord`.
- `/home/knatte/Code/quarry/wts/ladder-harness/bench/loomyard-eval/ladder/internal/ladder/testdata/transcripts/leaked-prefix.jsonl` and `target-origin-quarry.jsonl` (Card 18) — hand-authored fixtures.
- `/home/knatte/Code/quarry/wts/ladder-harness/bench/loomyard-eval/ladder/internal/ladder/gates_test.go` and `score_test.go` (Card 19) — full coverage per spec, including the never-fatal-check-(c) property table and the real task-01 control-prompt pass/fail cases.

While writing Card 19's tests I found and fixed a real bug in `RedactAnswer` (composed-string replacements must run before the bare-token alternation, otherwise a prefixed tool name's bare suffix like "toc" in "mcp__quarry__toc" is never exposed as a standalone word) and a fixture defect (the target-origin-quarry.jsonl session_id accidentally embedded a third "quarry" occurrence). Both fixes are folded into the Card 19 commit since they were required to make its own tests pass.

`go test ./bench/loomyard-eval/ladder/...` passes, `go vet` is clean, `gofmt -l` reports nothing, and `git status --porcelain --untracked-files=no` shows no dirty tracked files.

{"status":"success","commit_sha":"29ab02b835cb8e8f83b91f2896fdb9c67bcdb8be","session_id":"49b7a646-d252-45d5-be3d-960ad78d7679","cards_done":[16,17,18,19]}
