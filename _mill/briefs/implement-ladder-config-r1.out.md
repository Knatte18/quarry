7 of 7 cards committed, matching all `Commit:` messages from cards 1–7 in the batch file. All cards complete.

{"status":"success","commit_sha":"f814d5b4177586f9f1b4301715f27dc2f7b82c28","session_id":"2bcb57e9-8308-4bc3-9110-7547dd58272e","cards_done":[1,2,3,4,5,6,7]}

Summary: implemented all 7 cards of batch `01-ladder-config`. Created the `ladder` Go package under `bench/loomyard-eval/ladder/internal/ladder/` porting `ladder_config.py` in full:

- `bench/loomyard-eval/ladder/internal/ladder/ladder.go` — tool constants (`QuarryTools`, `MCPPrefix`, `DaemonBackedTools`, `MCPName`), the four config types (`LadderConfig`, `TaskEntry`, `ScorerConfig`, `Ladder`), `ConfigError`, `LoadLadder` with every validation rule, `ConfigByID`/`ControlFor`/`WarmCounterpartFor`, and `RequirePins`/`RequireSessionPins`.
- `bench/loomyard-eval/ladder/internal/ladder/settings.go` — `DenyListFor`, `SettingsDocumentFor`, `WriteSettings`.
- `bench/loomyard-eval/ladder/internal/ladder/preamble.go` — `PARALLEL_OPENING`, `PARALLEL_BLOCK`, `B_PREAMBLE_BODY`, closing sentence, tool descriptions, `PreambleFor`.
- `bench/loomyard-eval/ladder/internal/ladder/fenced.go` — `ExtractFencedJSON` with `ErrNoFencedJSONBlock` sentinel.
- Corresponding `_test.go` files for each, plus `bench/loomyard-eval/ladder/ladder.yaml` updated with `run_effort`, `session_dir_template`, and blanked `max_turns`.

`go test ./bench/loomyard-eval/ladder/...` passes (all subtests green). Working tree is clean; all 7 commits pushed to `port-ladder-bench-to-go`.
