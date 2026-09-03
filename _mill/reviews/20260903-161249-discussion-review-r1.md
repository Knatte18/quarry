# Review: Ladder harness around headless claude -p (T2)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: orchestrator
reviewer_self_id: Claude Fable 5 (claude-fable-5)
reviewed_file: _mill/discussion.md
date: 2026-09-03
```

## Findings

### [BLOCKING:design] Gate 2b fatally fires on every control rep
**Section:** gates (check b) vs no-tmp-paths
**Issue:** no-tmp-paths puts every task worktree at `<quarry-repo>/.scratch/ladder-worktrees/<task-id>`, so every cell's cwd — and thus the `system.init` record, `Read`/`Bash` absolute paths in tool blocks — contains the quarry repo root as a prefix; check (b) ("transcript contains the quarry repo root path → fatal", no `tool_result` carve-out) then kills every control rep. V1's check was sound only because V1 worktrees lived in `/tmp` (see `worktree: /tmp/loomyard-eval-01` in the current `ladder-toc.yaml`).
**Suggested fix:** scope check (b) to quarry-root paths *outside* the task-worktree subtree (strip/whitelist the worktree prefix before matching), or place ladder worktrees outside the quarry repo root.

### [NIT:consistency] Line budget misread as "including tests"
**Demoted-from:** BLOCKING
**Section:** Constraints ("Roughly 1 000–1 500 lines including tests")
**Issue:** plan §9a says "one Go program of roughly 1 000–1 500 lines with tests", and §2's table sets ~1 000–1 500 against V1's "9 000 (+8 300 test)" — i.e. the budget parallels the non-test figure; the discussion's own Testing section (six table-test files, two test-built binaries, e2e with resume/failure paths) cannot fit inside 1 500 total, so the stated constraint contradicts both the governing doc and the discussion's own testing plan.
**Suggested fix:** state the budget as ~1 000–1 500 non-test lines, tests besides.

### [NIT:consistency] HANDOFF citations use phantom rule numbers and wrong section
**Section:** one-preamble-for-every-cell ("HANDOFF rule 6"), Constraints ("HANDOFF.md §2 rule 1", "Rules 2, 3, 4, 5 and 7")
**Issue:** HANDOFF.md contains no numbered rules; the two surviving rules are prose in §3 ("Two harness rules carry over into T2"), and §2 is the decisions list. The content cited is right, the addresses are fabricated.
**Suggested fix:** cite HANDOFF §3 and drop the rule numbering (it belongs to the V1 README on `v1-final`, if anywhere).

### [NIT:design] Worktree trust under `claude -p` is asserted, not probed
**Section:** no-tmp-paths ("`.scratch/` … sits under an already-trusted ancestor")
**Issue:** V1's `/tmp` lesson was that Claude Code silently degrades permissions in an untrusted fresh directory; a fresh git worktree under `.scratch/` is its own project root, and whether ancestor trust (or headless `-p` with explicit `--tools`) exempts it was not among the probes, unlike every other invocation claim.
**Suggested fix:** have the live smoke test assert the expected `tools` list in `system.init` from inside a freshly created ladder worktree.

## Verdict

REQUEST_CHANGES
Two decisions collide (blinding gate vs worktree relocation) and the line budget contradicts the plan; both are one-paragraph fixes.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 1._
