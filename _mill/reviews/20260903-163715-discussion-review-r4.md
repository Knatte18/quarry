MILL_REVIEW_BEGIN
# Review: Ladder harness around headless claude -p (T2)

```yaml
duration_s: 186.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic Claude Opus; per environment, not independently verifiable)
reviewed_file: /home/knatte/Code/quarry/wts/ladder-harness/_mill/discussion.md
date: 2026-09-03
```

## Findings

### [BLOCKING:design] Gate 2 (c) still fatal for auto-loaded target context
**Section:** gates / no-tmp-paths ("What actually reaches the transcript, probed")
**Issue:** Check (c) is non-fatal only when `quarry` has an earlier `tool_result` antecedent, but the Environment section states the pinned Loomyard checkout's own `CLAUDE.md`, `CONSTRAINTS.md` and `.claude/agents/` are loaded into every cell — that content arrives without any `tool_result`, so a control that echoes it (or the injected context itself, if it appears in the stream) fails fatally and, per the same decision, is never retried; the zero-occurrence probe was run from an empty `~/.cache/ladder-eval/worktrees/probe`, not from the Loomyard checkout, so it does not cover this.
**Fix:** State whether auto-loaded project context counts as `target_origin`, and back it with a probe from a real pinned Loomyard worktree rather than an empty directory.

### [BLOCKING:design] Scorer redaction omits the bare server name
**Section:** scorer
**Issue:** The alternation is built from `quarry_tools` (bare and prefixed), the quarry repo root and the worktree path — the bare token `quarry` is not in it, so an answer whose prose says "the quarry toc tool" reaches the scorer intact and reveals the arm, contradicting the decision's own stated invariant and the treatment gate 2 check (c) gives that same token.
**Fix:** Say explicitly whether `server.name` (bare, from mcp-tool-prefix) joins the redaction alternation.

### [BLOCKING:design] Failure taxonomy: max_turns and scorer failures undisposed
**Section:** resume-and-failure / answer-extraction / scorer
**Issue:** Two named outcomes have no disposition: (a) a rep ending `terminal_reason: max_turns` — recorded as a metric and given a test fixture — produces no fenced answer, so it enters the `.invalid-<k>` path and burns three paid attempts before landing in `incomplete[]`, though it is a legitimate measurement, not "a nondeterministic formatting miss"; (b) a scorer call that exits non-zero, emits no fence, or fails the derived required-field check is not in the failure list at all, so it is unclear whether only the scorer is retried or the whole measured run is re-executed.
**Fix:** Enumerate the disposition of a `max_turns`/`is_error` rep and of a scorer-side failure, including which of the two `claude -p` calls a retry re-runs.

### [NIT:consistency] `<worktree-root>/.ladder.lock` location ambiguous
**Section:** single-run-lock
**Issue:** "worktree-root" reads either as `${LADDER_WORKTREE_ROOT:-…/ladder-eval}` or as `…/worktrees/<task-id>`; the second placement puts the lock inside a Loomyard checkout, where it trips `worktree_dirtied` every rep and the pinned restore can delete it.
**Fix:** Give the lock path literally, at the level above `worktrees/`.

### [NIT:consistency] `go.mod` description understates current requires
**Section:** Technical context ("Current tree")
**Issue:** The file is stated as "requiring only `go-tree-sitter` and `tree-sitter-go`", but `/home/knatte/Code/quarry/wts/ladder-harness/go.mod` also carries `tree-sitter-python`, `tree-sitter-rust` and `go-pointer` as indirect requires.
**Fix:** Note the indirect grammars, since the widening rationale leans on the current `go.mod` being minimal.

### [NIT:scope] `report` subcommand has no named test
**Section:** Testing / command-surface
**Issue:** The integration list exercises `run` end to end; nothing asserts `report` re-derives `summary.json` and `table.txt` from an existing `raw/` without re-running or re-scoring, which is the subcommand's whole justification.
**Fix:** Name a `report`-over-fixture-`raw/` case in the offline integration set.

## Verdict

REQUEST_CHANGES
Blinding gate, scorer redaction and failure taxonomy each leave a decision unmade.
MILL_REVIEW_END
