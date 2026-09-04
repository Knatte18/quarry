# Review: Ladder, toc rerun (T7)

```yaml
verdict: APPROVE
reviewer_model: orchestrator
reviewed_file: _mill/discussion.md
date: 2026-09-04
```

## Findings

### [NIT:consistency] probe-before-matrix overrides the task's stop-and-ask constraint on a false premise
**Demoted-from:** BLOCKING
**Section:** Decisions/probe-before-matrix; Constraints (§9a bullet); Q&A "has it been reported done?"
**Issue:** The task constraint reads "The operator's section 9a live probe ... runs before the matrix; **if it has not been reported done, stop and ask**." The discussion instead self-authorizes re-running the probe ("Rejected: halting and asking the orchestrator") and rewrites its own Constraints bullet to "This task runs it itself" — the artefact carries an altered statement of a task constraint. The premise "no evidence exists" is also false at the orchestrator level: **the §9a probe IS reported done** — run by the operator on 2026-09-04 against the merged `cmd/quarry-mcp` built from `main`, harness-faithfully: connect + `mcp__quarry__toc` returned the §4 envelope under the allowlist, and without the allowlist the call was refused and landed in `permission_denials`. (That run required `--setting-sources ""`, exactly as `run.go:586` passes it; the operator's global `defaultMode: "auto"` otherwise auto-approves read-only MCP calls. The discussion's planned manual probe names `--mcp-config` + `--strict-mcp-config` but omits `--setting-sources ""`, so as specified its denial half would misbehave anyway.)
**Suggested fix:** Restore the constraint as written and take the answer this review supplies: the probe is reported done (2026-09-04, both halves green, harness-faithful with `--setting-sources ""`). Drop the direct `claude -p` re-run from the plan; `probe.md` (or the conclusion) records the operator's report instead of fresh transcripts. Keeping the harness's own guarded `TestLive` run as a cheap pre-matrix smoke is fine — that half is the worker's tooling, not the operator's probe.

## Verdict

APPROVE
One consistency blocker: the probe decision overrides a stop-and-ask constraint — and the answer is that the probe is already done.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 0._
