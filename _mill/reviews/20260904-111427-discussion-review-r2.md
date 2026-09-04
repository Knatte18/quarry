MILL_REVIEW_BEGIN
# Review: Ladder, toc rerun (T7)

```yaml
duration_s: 227.0
verdict: REQUEST_CHANGES
reviewer_model: opus
reviewer_self_id: claude-opus-5 (Anthropic Claude, Opus tier; best-effort self-assessment)
reviewed_file: _mill/discussion.md
date: 2026-09-04
```

## Findings

### [BLOCKING:design] Hash-drift detector is defeated by resume
**Section:** `clean-tree-before-the-matrix` / `resume-not-restart`
**Issue:** `run.go` rebuilds the server on every invocation and writes `prov.ServerHashes[repKey(c.ID, rep)] = hash` for **all** reps `1..repsEffective`, overwriting hashes already recorded for completed reps; `CollectInvocation` leaves `Invocation.ServerHashes` nil, so no per-invocation history survives either. After any resume `WarnOnServerHashDrift` sees exactly one distinct hash and cannot fire.
**Fix:** State that the drift warning is not a valid safety net under the resume workflow the task mandates, and name what the run does instead (e.g. record the built binary's hash out-of-band per invocation, or treat `quarry_commit` equality across `invocations[]` as the actual check).

### [BLOCKING:design] Blinding failures carry no attempt ceiling
**Section:** `resume-not-restart`
**Issue:** "MaxAttempts (3) ceiling" governs only the `InvalidateRep` path (infrastructure failure / formatting miss). A fatal gate-2 finding is written by `writeCompleteState(..., blindingFailed=true)`, is never invalidated, and carries no attempt counter — the same command re-attempts it without limit. `run.go`'s own comment says such a rep is re-attempted "once the operator fixes the cause", a step the discussion never names.
**Fix:** Give the blinding path its own stop rule (e.g. one re-attempt, then stop and record the cause in the conclusion) and state what "fixing the cause" means for each fatal check.

### [NIT:scope] Gate inventory omits check (d) and the memory-path scan
**Demoted-from:** BLOCKING
**Section:** `## Technical context` → `gates.go` / `provenance.go`
**Issue:** Two fatal mechanisms are absent. `CheckRenderedControlPrompt` is check **(d)**: pre-dispatch, fatal on bare `quarry`, the server name, any `quarry_tools` entry (`toc`), or the MCP prefix in a control prompt — deterministic, so re-running can never clear it. `ScanMemoryPaths` is fatal on a non-existent auto-memory path or any file under one containing the bare token `quarry`, and `run.go` turns it into `abortRun`, stopping the whole invocation mid-matrix. This host's auto-memory directory path is itself `-home-knatte-Code-quarry-wts-quarry/memory/`.
**Fix:** Add both to the gate inventory and to Testing's "scenarios the conclusion must cover", and state the disposition for each (in particular: what the run does when the memory scan aborts the invocation).

### [NIT:consistency] Smoke test not run under the matrix's environment
**Demoted-from:** BLOCKING
**Section:** `probe-is-reported-done` / `matrix-runs-backgrounded` / Testing
**Issue:** `env -u CLAUDECODE -u CLAUDE_CODE_ENTRYPOINT` is applied only to the matrix invocation. The pre-matrix `LADDER_LIVE_TEST=1 go test ...` is specified bare, so it exercises the `claude -p` seam under markers the matrix deliberately strips — and it is precisely the seam-under-this-condition claim the test exists to make.
**Fix:** Specify the same `env -u` prefix for the smoke-test command, or state why the difference is acceptable.

### [NIT:consistency] Comparison metric key is misnamed
**Section:** `separation-verdict-from-the-harness`
**Issue:** `comparisons[].metric` in `summary.json` is `cache_read_input_tokens` (`costMetricNames`); `cache_read` is only the `table.txt` column header (`tableColumnNames`).
**Fix:** Name both spellings so the plan reads the right key out of the right artifact.

### [NIT:consistency] Wrong resolver function name
**Section:** `## Technical context` → `worktree.go`
**Issue:** The environment reader is `ResolveLoomyardRepo`, not `ResolveTargetRepo`; no such symbol exists.
**Fix:** Correct the name.

### [NIT:decision] probe.md content has no stated source
**Section:** `probe-is-reported-done`
**Issue:** `probe.md` is a committed gate artifact, but no operator report exists in the tree and the discussion does not say what it must contain or which CLI version the 2026-09-04 probe ran under (2.1.236 on this host vs 2.1.259 in plan §9a).
**Fix:** State that `probe.md` transcribes the orchestrator's round-1 answer verbatim, and list the minimum fields it must carry.

## Verdict

REQUEST_CHANGES
Four blocking gaps: drift detector defeated by resume, uncapped blinding retries, two omitted fatal gates.
_Note: 2 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 2._
MILL_REVIEW_END
