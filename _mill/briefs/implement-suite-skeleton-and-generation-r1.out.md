All 5 cards are committed and match the batch's 5 declared cards exactly (5 of 5 cards committed). Verify passed with 70 tests.

{"status":"success","commit_sha":"242a0de8d080328cb4c86d159f73ff5c172171e9","session_id":"8c7a710d-f135-4924-a3cb-419641388b62","cards_done":[1,2,3,4,5]}

Summary: All 5 cards of batch `01-suite-skeleton-and-generation` were completed and committed (5 of 5 — full completion, verified by matching commit subjects against the batch's `## Cards` `Commit:` messages).

Files touched:
- `/home/knatte/Code/quarry/wts/mcp-capability-bench/.gitignore` — added the `bench/loomyard-eval/ladder/results/**/raw/` ignore block
- `/home/knatte/Code/quarry/wts/mcp-capability-bench/bench/loomyard-eval/ladder/tests/conftest.py` — new, puts `scripts/` on `sys.path`
- `/home/knatte/Code/quarry/wts/mcp-capability-bench/bench/loomyard-eval/ladder/ladder.yaml` — new, the 15-config declarative source of truth
- `/home/knatte/Code/quarry/wts/mcp-capability-bench/bench/loomyard-eval/ladder/scripts/ladder_config.py` — new, `load_ladder`/`LadderConfig`/`deny_list_for`/`settings_document_for`/`write_settings`/`preamble_for` and supporting constants
- `/home/knatte/Code/quarry/wts/mcp-capability-bench/bench/loomyard-eval/ladder/tests/test_ladder_config.py` — new, 70 tests covering all deterministic units

Verify command `uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests/test_ladder_config.py -q` passes: 70 passed. No project formatter was available/configured in this repo, so no formatting drift step was needed. Working tree is clean (no uncommitted tracked changes).
