Summary of work completed for this holistic fix round:

1. **BLOCKING:consistency** — `bench/loomyard-eval/ladder/scripts/run_ladder.py`: aligned `run_cold_cell`'s persisted disposition strings (`confirmed-cold`, `not-run`, `no-daemon-signal`) with `summarize.py`'s hyphenated comparisons, per the plan's literal spec. Added a cross-module test in `bench/loomyard-eval/ladder/tests/test_run_ladder.py` that feeds `run_cold_cell`'s real output into `summarize.build_summary`.

2. **BLOCKING:design** — Added `iter_tool_use_blocks` to `bench/loomyard-eval/ladder/scripts/extract_usage.py` (id-carrying tool_use traversal, with `iter_tool_uses` now a thin wrapper over it, preserving its documented interface); `bench/loomyard-eval/ladder/scripts/gates.py` now imports and reuses it instead of its private `_tool_use_blocks`.

3. **BLOCKING:design** — Removed `run_ladder.py`'s `_denied_call_succeeded`; `run_probe` now calls `gates.gate_denied_tools_not_used` directly.

4. **NIT:consistency** — Added `ladder_config.extract_fenced_json(text, which="first"|"last")`; `score_run.py`, and `run_ladder.py`'s `_first_fenced_json_block`/`_extract_answer` now all use it instead of three near-duplicate regex/parse idioms. Dropped the now-unused `import re` from `run_ladder.py`.

5. **NIT:design** — `run_cold_cell` now distinguishes "live daemon found before every attempt" from "native-fallback exhausted" as separate not-run causes, recorded in a new `not_run_causes` map in `cold_cell.json`, with accurate reason text for both `not-run` and `partial` dispositions. Added two tests covering both causes.

All 208 tests across the six pytest files pass; all six non-null batch `verify:` commands (`test_ladder_config.py`, `test_extract_usage.py`, `test_gates.py`, `test_score_run.py`, `test_summarize.py`, `test_run_ladder.py`) exit 0. Five commits were made and pushed to `mcp-capability-bench`.

{"status":"success","commit_sha":"648d37c9da8f23b53cbdca17fb9de5431a30c31a","session_id":"6c575eeb-587d-4f3f-a096-aee4d4f2f97b"}

{"status":"success","commit_sha":"648d37c9da8f23b53cbdca17fb9de5431a30c31a","session_id":"6c575eeb-587d-4f3f-a096-aee4d4f2f97b"}
