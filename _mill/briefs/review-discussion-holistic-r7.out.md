MILL_REVIEW_BEGIN
# Review: Ladder harness around headless claude -p (T2)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic), high reasoning effort
reviewed_file: /home/knatte/Code/quarry/wts/ladder-harness/_mill/discussion.md
date: 2026-09-03
```

## Findings

### [BLOCKING:consistency] `report` has no source for the MCP prefix
**Section:** command-surface / mcp-tool-prefix / metrics
**Issue:** `report` takes no `--config`, recomputes metrics from `raw/` (its stated justification: "a summariser or metrics fix does not cost another matrix"), and gate 1 needs `quarry_tool_uses`, which is defined as "uses whose name starts with the MCP tool prefix … never a hardcoded literal" — but the prefix comes from `server.name` in the yaml, and neither `run.json`'s specified payload nor `provenance.json`'s field list persists it (`ladder_file` is explicitly "never read back as configuration").
**Fix:** state where the resolved prefix (or `server.name`) is persisted per root, or that `report` reads it from `run.json`.

### [BLOCKING:design] Void-but-`complete` reps are invisible to exit status
**Section:** gates / resume-and-failure
**Issue:** a gate-2 failure and a memory-taint discard both write a `complete` rep that contributes to no median and is never retried, while `incomplete[]` is derived from `selected_cells × reps_effective` presence — so a control cell whose reps all fail check (d) (deterministic, identical every rep) yields n=0 medians, empty `incomplete[]`, and an unspecified process exit code; the memory-taint case additionally can never be re-run after the operator cleans the directory, since resume skips `complete`.
**Fix:** state the run-level disposition — whether `blinding_failed_count > 0` forces a non-zero exit and whether such reps count as present for `incomplete[]` / are eligible for re-run.

### [BLOCKING:design] `provenance.json` write policy on resume is undefined
**Section:** provenance / resume-and-failure / no-tmp-paths
**Issue:** `selected_cells` and `reps_effective` are defined per invocation ("what `--cells` resolved to"), but `report`'s `incomplete[]` needs the union across every invocation of the root, and the memory-path scan reads the persisted `memory_paths` at startup — nothing says whether a resumed run with different `--cells`/`--reps` rewrites, merges, or must match the existing file.
**Fix:** decide and state the rewrite/merge/refuse rule for `provenance.json` on a resumed root.

### [NIT:decision] `usage.json` and `answer.redacted.json` have no producer or contract
**Section:** results-raw-ignored
**Issue:** both are named in the ignored `raw/` inventory, but no decision says what they contain or who writes them; `metrics` says the figures are "computed" and never says they are persisted.
**Fix:** name the writer and the contents of each, and state that they are diagnostic only since `report` recomputes.

### [NIT:consistency] Schema-block extraction keys on a heading spelling
**Section:** Technical context / exploration-schema-recovery / Testing
**Issue:** extraction takes "the `## Output schema (…)` fenced block", but the parenthetical differs between the two ladder-referenced files (`(exploration tasks)` vs `(impact-analysis tasks)`, verified at `04-…md:104`), while Testing says "heading spelling is not load-bearing and must not be tested as if it were"; the behaviour when a task file has no such heading, and any cross-check against the yaml's `schema:` field, are unstated.
**Fix:** state the match rule (prefix `## Output schema`) and the error for a missing block.

## Verdict

REQUEST_CHANGES
Three gaps: `report`'s prefix source, void-rep exit accounting, provenance-on-resume policy.
MILL_REVIEW_END
