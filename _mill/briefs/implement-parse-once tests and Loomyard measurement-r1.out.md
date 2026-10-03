Batch 2 is done: 1 of 1 committed card (card 6) plus card 7, which has no commit. Verify passes on all four packages with the pinned Loomyard clone. Nothing tracked is left uncommitted.

- **Card 6:** I created `/home/knatte/Code/quarry/wts/resolve-self-dirpackage-memo/internal/engine/memo_test.go` with the four tests and the `assertBuilds` helper as specified. All four pass. I committed and pushed it.
- **Card 7:** I read-only built both `lyx` binaries against the Loomyard prime (the prime was not modified) and timed them on the quarry34-plan repro.
  - **Baseline (quarry v0.2.0):** 68.9 s, exit 0.
  - **Branch:** 1.9 s, exit 0.
  - **Comparison:** `cmp` reports the findings files identical, 5186 bytes each.
  - **Acceptance:** met (branch at most 5 s, findings identical).
  - **Where it's written:** `/home/knatte/Code/quarry/wts/resolve-self-dirpackage-memo/.scratch/loomyard-measurement.md`. The findings JSON files and both binaries are under `.scratch/` in the worktree.

{"status":"success","commit_sha":"4fd3fbb3dfce4c3a1859be0ad9d80f397b3f8bdc","session_id":"306432e0-9daf-447f-9719-7b1188db7ced","cards_done":[6,7]}
