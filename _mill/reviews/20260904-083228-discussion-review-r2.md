MILL_REVIEW_BEGIN
# Review: MCP, thin (T6)

```yaml
duration_s: 306.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic Claude Opus family, Agent SDK; knowledge cutoff May 2026)
reviewed_file: /home/knatte/Code/quarry/wts/mcp-thin/_mill/discussion.md
date: 2026-09-04
```

## Findings

### [BLOCKING:design] D7 rests on a premise the repo already refuted
**Section:** D7 (and the matching Q&A line) **Issue:** D7 rejects `structuredContent` because it "would corrupt the metric the task exists to feed", but `docs/research/mcp-surface.md:103-106` measured the opposite against a ladder transcript — "It does not cost context… Claude Code renders `content[].text` and discards `structuredContent`. The duplication is transport-only." **Fix:** restate D7's rationale on grounds that survive that measurement (one payload shape, CLI-identical bytes, no output schema to maintain) or cite the measurement and re-decide; the same paragraph should also say how the tool is registered so the SDK infers no `outputSchema`, since the research note records that all seven V1 tools declared one.

### [BLOCKING:design] cwd-inherited rooting is load-bearing and ungated
**Section:** D3 / D13 **Issue:** the whole "harness untouched" scope rests on the stdio server inheriting `claude`'s `Dir: dest` (`run.go:590`), yet nothing verifies it: §9a's probe table has no cwd row, and D13's live probe is run against the quarry repo, where cwd discovery would succeed even if the assumption is false — a wrong root then presents as a granted cell whose answers are all not-found, exactly D1's silent-failure mode. **Fix:** make the live probe run `claude -p` with cwd in a *different* repository (or run the ladder once at `--cells a2-toc-dir --reps 1`) and assert the returned envelope names that repository's files.

### [BLOCKING:design] Moved tests lose their fixture helper
**Section:** Testing, `internal/repopath` / D11 **Issue:** `internal/cli/root_test.go` and `target_test.go` build every fixture with `writeScratchTree` — an unexported `package cli` helper (`scratchtree_test.go:29`) whose header records that `t.TempDir()` is banned for this repo — so "the existing table tests move with the functions" has no stated fixture mechanism in the new package, and the discussion elsewhere calls duplicating that helper the cost it wants to avoid. **Fix:** state where `repopath`'s tests get their trees (helper duplicated, helper promoted to a shared internal test package, or committed `testdata/`), and reconcile it with "internal/cli's own tests stay unchanged".

### [BLOCKING:design] `depth` has no stated validation or absent-value semantics
**Section:** D5 / D6 **Issue:** the CLI rejects any negative `--depth` other than `all` (`flags.go:100`), while a bare JSON integer accepts `depth: -7`, which `walk.go:386-390` then treats as unbounded recursion that never applies the identity-only cut — a surface the discussion calls "a mirror of the CLI" diverging on invalid input; absent `depth` is likewise never given a meaning. **Fix:** say whether the schema carries `minimum: -1` or the handler returns a D8 error for `depth < -1`, and state that an absent `depth` means 0, the CLI's own default.

### [NIT:consistency] "Both currently return usageError" is false
**Section:** Technical context, "What is being moved" **Issue:** `repoRelTarget` returns `quarry.ErrTargetOutsideRepo`, never `usageError` (`internal/cli/target.go:38,43`); its only reuse blocker is that it is unexported, and D11 itself already assigns it the facade sentinel. **Fix:** correct the sentence to name only `discoverRoot`/`resolveRoot` as the `usageError` returners.

### [NIT:consistency] D12's stderr line is discarded, not merely uncaptured
**Section:** D12 **Issue:** `invokeMeasuredProcess` sets no `Stderr` on the measured `Cmd` (`run.go:589-595`, `worktree.go:47-50`), so in a ladder run the diagnostic line goes nowhere — the rationale "a misrooted server is otherwise invisible" does not hold for the case it names. **Fix:** keep the decision but scope its rationale to interactive/operator use, and say where a misrooting is observed during a ladder run.

### [NIT:design] D11 leaves the error shape as two options
**Section:** D11 **Issue:** "a `repopath`-owned sentinel **or** error type" for the not-a-directory / no-root conditions is two alternatives with no choice, and the CLI's wrapping back into `usageError` differs between them. **Fix:** pick one and name it.

### [NIT:design] The denial probe has no elicitation or miss criterion
**Section:** D14 **Issue:** §9a's denial arose from a second, non-granted tool the agent happened to try; with one tool the probe depends on the model attempting a call it is not allowed, and no prompt or fallback is stated for the case where no call is attempted. **Fix:** state the probe prompt that forces the call and what the operator records if none occurs.

## Verdict

REQUEST_CHANGES
Four blocking items: a refuted rationale, an ungated rooting assumption, fixture-less moved tests, unspecified depth validation.
MILL_REVIEW_END
