MILL_REVIEW_BEGIN
# Review: Ladder harness around headless claude -p (T2)

```yaml
duration_s: 233.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (best-effort self-assessment; no independent way to confirm)
reviewed_file: _mill/discussion.md
date: 2026-09-03
```

## Findings

### [BLOCKING:design] Worktree path contains "quarry"; gate 2 kills controls
**Section:** no-tmp-paths + gates check (c)
**Issue:** the chosen worktree root `${XDG_CACHE_HOME}/quarry-ladder/worktrees/<task-id>` contains the literal `quarry`, and check (c) is a case-insensitive bare `quarry` match that is fatal anywhere outside a `tool_result`; that string lands in `system.init`'s `cwd` and in every absolute path inside a `tool_use` input, so every control rep — including this task's own `a0-none` done-criterion — fails fatally.
**Fix:** decide the interaction explicitly: either a worktree directory name free of the token, or a stated rule for which transcript fields check (c) scans.

### [NIT:consistency] Migrated ladder-toc.yaml cannot load under its own rules
**Demoted-from:** BLOCKING
**Section:** server-block vs ladder-toc-migration
**Issue:** server-block says an absent `server:` ⇒ every cell must be `allowed: []`, while ladder-toc-migration ships `a2-toc-dir` and `b8-toc-dir` with `allowed: [toc]` and `server:` "commented out or absent until T6" — so the committed file is invalid between T2 and T7, and the `a0-none` run fails at load before reaching the cell.
**Fix:** state whether the all-control rule is a load-time validation or a per-cell run-time check, and what the migrated file contains at T2 merge.

### [BLOCKING:design] Bare `Bash` with no allowlist entry is unprobed
**Section:** claude-invocation
**Issue:** every cell relies on `--tools …,Bash` with `--allowedTools` omitted entirely on controls, but the decision's own rationale records that the probe ran Bash under `--allowedTools "Bash(grep:*)"` — the actual control configuration was never exercised, no `--permission-mode` is decided, and `--setting-sources ""` removes the settings file that granted Bash in V1.
**Fix:** decide the permission mode and state how bare-`Bash` execution is verified; the live smoke test currently asserts only the advertised `system.init` `tools` list, which does not show whether a Bash call is permitted at runtime.

### [NIT:scope] `table.txt` and `conclusion.md` have no producer
**Demoted-from:** BLOCKING
**Section:** Scope "In" vs results-raw-ignored / cache-contamination
**Issue:** results-raw-ignored names `results/<root>/table.txt` and `results/<root>/conclusion.md` as committed artifacts and cache-contamination refers to "the `conclusion.md` template", but Scope "In" lists only `summary.json`, `provenance.json` and "the printed per-cell table", and neither `run` nor `report` is said to write either file.
**Fix:** add both to the work inventory with an owner, or state that `conclusion.md` is T7's hand-written artifact and the table is stdout-only.

### [NIT:design] Worktree and scratch paths reintroduce a collision V1 guarded against
**Section:** no-tmp-paths, Deletes-wholesale (`lock.go`)
**Issue:** the worktree is keyed on `<task-id>` alone and harness scratch on a single `.scratch/ladder/` path, while `ladder-toc.yaml:55-56` shows `session_dir_template` existed precisely so one matrix's dispatch never collided with another's scratch; `lock.go` is deleted with no successor.
**Fix:** state whether concurrent `ladder run` invocations are out of scope or guarded.

### [NIT:consistency] No-machine-paths rule mis-attributed to CLAUDE.md
**Section:** results-raw-ignored rationale
**Issue:** "`CLAUDE.md` / `HANDOFF.md` state that no tracked file may carry a machine path" — the tracked `CLAUDE.md` is three lines and says only "Go repo, no Python"; the rule is in `HANDOFF.md` §1.
**Fix:** cite `HANDOFF.md` §1 alone, matching the round-1 correction to the `§2 rules 1 and 6` address.

### [NIT:scope] Answer-key heading differs between task files
**Section:** Technical context "Task file structure"; Testing (`prompt.go`)
**Issue:** the heading is stated as `## Notes for whoever prepares C's fasit`, which is task 01's; task 04 uses `## Notes for whoever scores this (ground truth — do not reveal to A/B/C)` at line 119, and the prompt test names only the former.
**Fix:** state that extraction is inclusion-based (TASK TEXT + schema block only) so heading spelling is not load-bearing, and cover both files in the test.

## Verdict

REQUEST_CHANGES
Blinding gate contradicts the worktree path; migrated yaml fails its own validation; bare Bash unverified.
_Note: 2 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 2._
MILL_REVIEW_END
