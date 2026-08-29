MILL_REVIEW_BEGIN
# Review: Per-capability quarry-mcp benchmark suite

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5), Anthropic
reviewed_file: _mill/discussion.md
date: 2026-08-29
```

## Findings

### [BLOCKING:design] Scorer blinding is defeated by `evidence` fields
**Section:** "Correctness scoring against the existing fasit" **Issue:** The decision asserts the scorer "never sees the config id, the rung's tool set" — but task 04's schema *requires* an `evidence` string per caller saying how it was confirmed, and the committed A-arm answer (`results/2026-08-29/04-shedadapters-shuttle-impact-mcp/a.json`) states "quarry textDocument_definition on…", "quarry textDocument_references…", "singlellm.go's toc_file entry", i.e. the answer names the rung's exact tool set. Every Ladder B answer will do this, so the scorer is de-blinded on 24 of 45 runs by the one input it is allowed to see. **Fix:** State a disposition — strip/redact tool names from `answer.json` before scoring, or drop the blinding claim and say what the residual bias is.

### [BLOCKING:design] Run cwd is a worktree carrying its own `CLAUDE.md` and `.claude/`
**Section:** "Hard enforcement…" (cwd choice) / Technical context **Issue:** The discussion says each run's cwd is the Loomyard worktree so "nothing in a run's ambient context points at quarry" — but the live checkout at `/home/knatte/Code/loomyard/wts/loomyard` carries a tracked root `CLAUDE.md` (project instructions incl. "Read CONSTRAINTS.md … every session") plus `.claude/settings.local.json` (its own `permissions.allow`) and six `.claude/agents/*.md` definitions. A `claude -p` rooted there auto-loads these into all 45 runs as an uncontrolled prompt and a second permissions source, which the discussion never mentions. **Fix:** Decide explicitly whether the worktree's `CLAUDE.md`/`.claude/` are removed, ignored, or accepted-and-recorded, and say how project settings rank against the generated `--settings`.

### [BLOCKING:design] No separation rule for the comparisons called primary
**Section:** "Reporting discipline" vs the steering-confound block **Issue:** The only stated bar for a claim is "range disjoint from its own ladder's `none` control". But the confound block then rules that grep counts are *never* compared to `none` and that "the cleanest attribution claims this suite can make are between quarry rungs" — and no criterion is given for a rung-vs-rung claim, nor for the `a5-bundle` vs `a5-bundle-cold` contrast the cold cell exists to produce. `summarize.py`'s test scenarios only mention the control. **Fix:** State the separation criterion (and the tested behaviour) for rung-vs-rung and for warm-vs-cold, not only for rung-vs-control.

### [BLOCKING:consistency] "#006-comparable" contradicts "not for direct comparison"
**Section:** "Reuse existing tasks 01 and 04" vs Technical context **Issue:** The reuse rationale rests on the runs being "directly comparable to #006's own numbers for the same tasks", and `bash_grep_count` is defined as "the field used for any comparison against #006's numbers" — while Technical context labels the prior numbers "for orientation (not for direct comparison — different methodology)". #006's `usage.json` files also record no model id, and #006's own `tool_uses` is inconsistently scoped (task-01 warm `9` = all tools; task-04 warm `5` = quarry only against a 10-call breakdown), so cost comparability is not available at all. **Fix:** Pick one — say which axes (if any) may be compared to #006 and forbid the rest explicitly.

### [BLOCKING:design] `none` blinding gate matches the target codebase's own text
**Section:** "Hard enforcement…" / Protocol validation gates **Issue:** The gate rejects any `none` run whose transcript "contains … the string 'quarry'", and the discussion makes that detection the thing that "actually holds the constraint". The Loomyard checkout itself contains "quarry" ~100 times across 12 tracked files (`docs/code-comment-conventions.md`, `manifest/designs/quarry-plan-symbol-verification.md`, `internal/loomengine/plan_test.go`, …), so a `none` agent reading or grepping outside the task's package scope trips the gate with no harness leak — and after 3 such attempts the rule halts the entire 45-run matrix. **Fix:** Narrow the gate to leak-shaped evidence (`mcp__quarry__*`, `/tmp/quarry-bench`, paths into this repo) and state how a target-origin "quarry" hit is dispositioned.

## Verdict

REQUEST_CHANGES
Blinding, ambient run context, comparison rules and #006 comparability need explicit dispositions.
MILL_REVIEW_END
