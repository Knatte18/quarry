MILL_REVIEW_BEGIN
# Review: Port the capability-ladder bench harness to Go

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: /home/knatte/Code/quarry/wts/port-ladder-bench-to-go/_mill/discussion.md
date: 2026-08-30
```

## Findings

### [BLOCKING:consistency] Scratch dir holds both agent definitions, or one
**Section:** §"The session's working directory is a neutral scratch dir" + §Technical context "Session launch and agent-definition discovery" vs §"Tool exposure is enforced by a generated agent definition"
**Issue:** The first says "the generated `.mcp.json`, `settings.json`, and **both agent definitions** live in that scratch dir" and the Technical-context write list names `.claude/agents/<config-id>.md` **and** `.claude/agents/<config-id>-scorer.md`, while the tool-exposure decision says a `none` run session's scratch dir contains "exactly two things and nothing else" and that "prepare-session writes the scorer definition only into the scoring session's scratch dir".
**Fix:** State one disposition, and make the Technical-context write list conditional on session type so the blinding-containment claim and the generator spec agree.

### [BLOCKING:design] Skill-leak precondition scans the wrong directory
**Section:** §"Tool exposure…" — the `skill_listing` channel, mitigation 2
**Issue:** The scan is specified over `~/.claude/skills/*/SKILL.md`; that directory does not exist on this machine, and every installed skill lives under `~/.claude/plugins/cache/millhouse/<plugin>/<ver>/skills/*/SKILL.md` (63 files) — plugin skill names do reach a subagent transcript (`golang-build`, `mill-plan`, `weblens` all appear in `/home/knatte/.claude/projects/-home-knatte-Code-loomyard-wts-trace-logging/04e5b9d5-3796-45e2-9c8f-0fc74760e826/subagents/agent-a278328d065fa9d1b.jsonl`). As written the precondition passes vacuously and closes nothing.
**Fix:** Define the scan over the full set of discoverable skill sources, plugin caches included, and say what it does when a source root is absent.

### [BLOCKING:design] Matched meta with no sibling transcript is unhandled
**Section:** §"Transcript correlation by unique description" / §"Transcript custody"
**Issue:** The decision enumerates only "zero matches or more than one match" as hard errors and otherwise "takes the sibling `.jsonl`"; on disk a `.meta.json` can exist with no sibling — `/home/knatte/.claude/projects/-home-knatte-Code-millhouse-wts-mill-quick/8c571fb2-eb10-49d2-9ee4-38bff0e04c36/subagents/` holds three `.meta.json` and zero `.jsonl`. `ingest`'s whole custody chain rests on that sibling existing.
**Fix:** Give the exactly-one-match-with-missing-`.jsonl` case an explicit disposition (hard error vs. retryable "not yet flushed") and add it to the correlation test list.

### [BLOCKING:consistency] Unmarked superseded Q&A entries contradict the model
**Section:** §Q&A log
**Issue:** Three entries carry stale answers with no supersession marker, while other entries in the same log are explicitly marked superseded: "total sessions rise to 23", "How many sessions? **A:** 17", and "`{n}` is the session index within the config — 1 for a warm session, the repetition index for a cold one". The last directly contradicts the decided uniform repetition index and would collapse a warm config's three reps onto one scratch dir, breaking `ingest`'s one-directory search space.
**Fix:** Mark all three superseded, as the log already does elsewhere.

### [NIT:consistency] Provisional marker absent from the exhaustive partition
**Section:** §"Metrics: what survives, changes, and dies" vs §"`denied_tool_attempts` ships provisional"
**Issue:** The partition's **Added** bullet lists only `effort`, `agent_id`, `transcript_source`, yet `usage.json` also gains `denied_tool_attempts_provisional`; and the marker's propagation onto `summarize`'s stats record changes `summary.json`'s shape while the Consequence line says everything else in `METRICS` is unchanged, with no test named for it.
**Fix:** Add the field to the **Added** list and name a test for the propagation.

### [NIT:decision] `.session-active` has no tracked/untracked disposition
**Section:** §"Single-flight dispatch" / §"Results tree layout"
**Issue:** The lockfile sits at `<results_root>/.session-active`, outside the gitignored `results/**/raw/` pattern, so a live session leaves an untracked file in a tracked directory with no stated disposition.
**Fix:** Say whether it is gitignored or otherwise excluded.

## Verdict

REQUEST_CHANGES
Two verified design gaps, one contradiction, and stale Q&A answers still readable as live.
MILL_REVIEW_END
