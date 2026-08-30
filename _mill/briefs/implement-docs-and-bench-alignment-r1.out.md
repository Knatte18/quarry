All 3 cards in this batch are complete: card 6 (docs) committed as `7294b61`, card 7 (bench prompt) committed as `b276278`, and card 8 (the diagnostic completeness gate, Commit: none) had all four checks re-run this turn and all passed. Verify (the ladder pytest suite) passed with 208 passed. No tracked files are dirty.

Summary of files touched:
- `/home/knatte/Code/quarry/wts/mcp-target-dir-ergonomics/docs/mcp-setup.md` — new "The server's scoping contract" section, updated opening summary and `--target-dir` bullet.
- `/home/knatte/Code/quarry/wts/mcp-target-dir-ergonomics/bench/loomyard-eval/ladder/scripts/ladder_config.py` — replaced the stale `targetDir`/`buildTags` prompt instruction with a `buildTags`-only one; reworded the `DAEMON_BACKED_TOOLS` comment's `effectiveTargetDir` reference.
- `/home/knatte/Code/quarry/wts/mcp-target-dir-ergonomics/bench/loomyard-eval/ladder/tests/test_ladder_config.py` — narrowed the matching assertion to the new literal.

{"status":"success","commit_sha":"b27627810dc38ef57a61b226dabe7f61d07b0f19","session_id":"14b02808-9398-40a9-b283-2905666420de","cards_done":[6,7,8]}

3 of 3 cards committed/completed (cards 6 and 7 via real commits, card 8 verification-only with Commit: none, re-verified this turn).

{"status":"success","commit_sha":"b27627810dc38ef57a61b226dabe7f61d07b0f19","session_id":"14b02808-9398-40a9-b283-2905666420de","cards_done":[6,7,8]}
