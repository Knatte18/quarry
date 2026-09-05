I completed Card 14 (committed as `6221d93`) but hit a genuine plan conflict on Card 15 that I cannot resolve without violating a binding Shared Decision.

**What happened:** Card 15 requires `LADDER_LOOMYARD_REPO=... go test ./internal/engine/` to pass "with its Loomyard cases executed rather than skipped." Running it, one test fails:

```
--- FAIL: TestRoundTrip_LoomyardNaming
    naming_roundtrip_test.go:162: counts golden "testdata/loomyard/naming-counts.json" is missing;
    regenerate it on a machine with the pinned checkout: ... -update
```

I confirmed this is genuinely pre-existing and unrelated to this task, not a regression I introduced:
- `git show main:internal/engine/testdata/loomyard/naming-counts.json` — the file does not exist on `main` either.
- I ran the identical checkout-gated test against the sibling `main` worktree (`/home/knatte/Code/quarry/wts/quarry`) using the same pinned checkout, and got the byte-identical failure. This test belongs to a different, already-merged feature (`486d416 "The glyph-maker: declaration to glyph (P1, roadmap 2b)"`), not to glyphs-verb.
- It is the only failure in `./internal/engine/`; every other case passes.

The fix would be to generate the missing golden (`go test ./internal/engine/ -run TestRoundTrip_LoomyardNaming -update`), but that writes a file under `internal/engine/testdata/`, which the overview's `no-file-under-internal-engine-is-modified` Shared Decision (binding on all batches, no testdata exception in its text) explicitly forbids: "no card in any batch edits, creates or deletes a file under `internal/engine/`." Per the brief's stuck-type rules, `stuck_type: verify` requires confirming the failure is **not** pre-existing — I've confirmed the opposite, so that type is unavailable, leaving this a plan contradiction between the batch's own verify requirement and a Shared Decision it cannot violate to satisfy.

**Card-count self-check:** 1 of 5 cards committed (Card 14 only — `test(cli): five glyphs golden table rows and the depth target probe`, commit `6221d93`). Cards 15–18 are not started/committed. The five new golden files for Card 15 were generated via `-update` and verified to be exactly the five expected new untracked files with zero modification to any pre-existing golden under `internal/cli/testdata/` or `internal/engine/testdata/` — but they are intentionally left uncommitted since Card 15's own Requirements (the engine-pass gate) are not satisfiable in-scope.

{"status":"stuck","stuck_type":"logic","reason":"Card 15 requires go test ./internal/engine/ to pass with Loomyard cases executed, but TestRoundTrip_LoomyardNaming fails on a missing testdata/loomyard/naming-counts.json golden that is confirmed pre-existing (identical failure reproduces on main with the same pinned checkout) and unrelated to this task; the only fix is generating that golden under internal/engine/testdata/, forbidden by the overview's no-file-under-internal-engine-is-modified Shared Decision (applies to all batches).","commit_sha":"6221d93cf7e5cbb670ded8141fb56027e15d363d","session_id":"72880d36-0e5f-4386-b794-3e5b9d22c52c"}
