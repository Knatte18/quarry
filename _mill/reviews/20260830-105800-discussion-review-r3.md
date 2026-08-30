MILL_REVIEW_BEGIN
# Review: Port the capability-ladder bench harness to Go

```yaml
duration_s: 207.4
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude, Opus family; environment reports claude-opus-5, which I cannot independently confirm
reviewed_file: /home/knatte/Code/quarry/wts/port-ladder-bench-to-go/_mill/discussion.md
date: 2026-08-30
```

## Findings

### [BLOCKING:design] skill_listing leaks the skill into blinded transcripts
**Section:** "Tool exposure is enforced by a generated agent definition" (residual-limit paragraph) + Technical context "subagent transcript format"
**Issue:** Verified against `/home/knatte/.claude/projects/-home-knatte-Code-millhouse-wts-millhouse/92e32103-.../subagents/agent-a165b5e2e9117ed4e.jsonl` line 3: every subagent transcript carries a `skill_listing` record enumerating user-scope skill **names and descriptions** verbatim (`"content":"- mill:cli: Shell command guidelines. ..."`) — so installing `~/.claude/skills/ladder-run/` puts its quarry-naming description into every blinded `none` run agent's transcript with no `Bash` involved, and `gate_blinding` makes a bare "quarry" outside a `tool_result` fatal. The discussion frames the user-scope install as removing the leak from the agent's cwd and treats `Bash` reachability as the only residual.
**Fix:** Decide how a `none` session's skill discovery is handled given `skill_listing` (e.g. no skill installed for `none` sessions, or a quarry-free frontmatter description), and state that the operator's other `~/.claude/skills/` entries are listed to every run agent too.

### [NIT:consistency] Q&A log still says the skill is copied into the scratch dir
**Demoted-from:** BLOCKING
**Section:** Q&A log ("How does a repo-tracked skill reach a session launched from `/tmp`?") vs "The orchestration loop lives in a repo-tracked skill" and the scratch-dir containment rule
**Issue:** The Q&A entry auto-picks "`prepare-session` copies `SKILL.md` into the scratch dir's `.claude/skills/`" with `~/.claude/skills/` as *fallback*; the decisions state the exact opposite (never copied into a scratch dir; user-scope install for every session type), and the Subcommand surface bullet for `prepare-session` still lists "skill copy" among what is written into the scratch dir.
**Fix:** Replace the superseded Q&A answer and drop "skill copy" from the `prepare-session` scratch-dir list.

### [NIT:consistency] "no dispatch of any kind" vs a mandated `claude` launch
**Demoted-from:** BLOCKING
**Section:** Scope "Out" / "denied_tool_attempts ships provisional" vs Technical context "Known implementation risks"
**Issue:** Scope says "No paid run of any kind — not even a preflight probe" and the provisional-marker rationale rests on "this task runs no dispatch of any kind", while the implementation risks say both must "be settled by an actual `claude` launch during the batch that builds `prepare-session`, never assumed" — establishing agent-definition loading and MCP `env` inherit-vs-replace both require a live launch, and risk 1 arguably a dispatch.
**Fix:** State explicitly whether a non-matrix `claude` launch (and any subagent dispatch inside it) is in scope for the `prepare-session` batch, or move both risks to the follow-up task with a named default.

### [NIT:scope] Testing plan omits three in-scope units the Python suite covers
**Demoted-from:** BLOCKING
**Section:** "Testing" (TDD candidates)
**Issue:** `task_text_for` / `schema_for`, the cold-cell disposition logic, and `probe-record` are all in scope, and all are tested today (`tests/test_run_ladder.py:160,167,174` for the task-text boundary; `:419,:458` for `not-run` / `not_run_causes`; `:248,:263` for the probe), yet none appears in the TDD list — including the boundary the Gotchas section calls "load-bearing, not tidiness" because over-reading pastes the answer key into every prompt.
**Fix:** Add TDD cases for the task-text section boundary (both tasks plus missing-heading), the four cold dispositions with `not_run_causes`, and `probe-record`'s `probe.json` write including `denial_shape_observed`.

### [BLOCKING:design] `max_turns: 60` reused against a redefined turn count
**Section:** "`max_turns` becomes a post-hoc gate" + "Metrics" (Changed)
**Issue:** `num_turns` is redefined as the count of `assistant` records, and the ceiling is evaluated against that count, but `ladder.yaml`'s committed `max_turns: 60` was calibrated against the `claude -p` client's `--max-turns`/`result.num_turns`; the discussion never says whether the threshold is still calibrated, so the gate may be effectively dead or fire spuriously.
**Fix:** State whether 60 is retained as-is under assistant-record counting, or is to be re-baselined, and record which basis the value now means.

### [NIT:consistency] `none` scratch-dir "only" list omits the scorer definition
**Demoted-from:** BLOCKING
**Section:** "What a `none` session's scratch directory may contain" vs Technical context "Session launch and agent-definition discovery"
**Issue:** The containment rule is stated as exhaustive ("contains only: the agent definition, a `settings.json` ..., and no `.mcp.json`"), but `prepare-session` is separately specified to also write `.claude/agents/<config-id>-scorer.md` into the same dir; since the whole blinding argument rests on that enumeration being complete, an omission undoes it.
**Fix:** Either add the scorer definition to the enumeration with its content constraint, or decide it is not written into a `none` session's scratch dir.

### [NIT:consistency] Stale `session_dir_template` examples
**Section:** "The session's working directory is a neutral scratch dir" and Technical context "subagent transcript format"
**Issue:** Both give `/tmp/ladder-session-a5-bundle` (and the slug `-tmp-ladder-session-a5-bundle`), predating the `{n}` suffix decided in "`ladder.yaml` additions", where the path is `/tmp/ladder-session-a5-bundle-1`.
**Fix:** Update both examples to include `-{n}`.

## Verdict

REQUEST_CHANGES
Skill-listing leak, a superseded Q&A auto-pick, a dispatch-scope contradiction, and testing gaps.
_Note: 4 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 2._
MILL_REVIEW_END
