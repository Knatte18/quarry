# Review: Facade + CLI, resolve + expand (T5b)

```yaml
verdict: APPROVE
reviewer_model: orchestrator
reviewed_file: _mill/discussion.md
date: 2026-09-04
```

## Findings

### [NIT:consistency] D10's "no separator check" purity is already conceded by D8
**Section:** D10, D8
**Issue:** D10 keeps the CLI free of a `#` check and detects the no-separator case post-engine via `errors.As` on the wrapped `*glyph.ParseError` — but D8 already gives the CLI its own `strings.Contains(target, "#")` copy of `isGlyphTarget` for `resolve` routing, so the purity argument is spent, and the `errors.As` route runs root discovery, `quarry.Open` and the engine's parse before reporting what is then called a usage error (exit 2's T5a comment reads "TOC is never called on this path").
**Suggested fix:** Keep the recommended outcome (exit 2, the CLI-spelled message) but let the plan route it on the CLI's existing `#` check at argument-handling time, before any engine call; a target *with* `#` that the grammar still rejects (e.g. `#x`) keeps D4's exit-1 envelope path unchanged. If the plan prefers the `errors.As` route anyway, state the "usage error discovered after the engine ran" deviation explicitly.

### [NIT:consistency] Plan §5's variadic `resolve` bullet stays unamended by this task
**Section:** D1 (Tension, flagged for review)
**Issue:** `docs/rewrite-plan.md` §5 still spells `resolve <glyph|path>...` while the task builds a one-target CLI; the discussion flags the tension but leaves the plan text's disposition open.
**Suggested fix:** Proceed on single-target exactly as decided — the plan §5 amendment is an operator follow-up on `main` (the same move as `toc`'s one-target amendment, commit `55161a6`), not a change this worktree makes; the plan may note that in one line so the inventory is closed.

## Verdict

APPROVE
All four flagged auto-picks (D1 single-target CLI, D2 payload-not-envelope on negative answers, D3's ambiguous→1 and grammar-rejection→1 rows, D10 exit 2 for a `#`-less expand target) are confirmed against the merged engine/CLI code and plan §4/§5; the two NITs are routing and bookkeeping, not blockers.
