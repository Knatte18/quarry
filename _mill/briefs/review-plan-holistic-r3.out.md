MILL_REVIEW_BEGIN
# Review: Ladder breadth (M1) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic), high reasoning effort
reviewed_file: plan/
date: 2026-09-04
```

## Findings

### [BLOCKING:design] Copied schema block plants a reed path in both new prompts
**Location:** batch 2 card 6, batch 3 card 8, batch 4 card 12
**Issue:** Cards 6 and 8 copy task 01's exploration schema block verbatim and card 12 requires the two new `wantSchemaBlock` constants be byte-identical to `task01SchemaBlock`; that block's example is `"relevant_files": ["internal/reedengine/geometry.go", "..."]`, and `RenderPrompt` puts `SchemaBlock` into every rendered prompt — so task 06's "names no package and no file" prompt names `internal/reedengine/geometry.go`, and card 8's constraint (b) is silently unsatisfiable if card 9's read lands the subject in `internal/reedengine` or `internal/reedcli`.
**Fix:** State the disposition explicitly — either exclude those two packages from card 8's pick, or make the example path placeholder-only and drop card 12's byte-identity requirement against `task01SchemaBlock`.

### [BLOCKING:consistency] Results-root date substitution stops at batch 5
**Location:** batch 5 batch-local decision; batch 6 cards 17 and 18; overview `## All Files Touched`
**Issue:** Batch 5 permits renaming the root when the invocation starts on a different calendar date but scopes the substitution to "everywhere in this batch"; cards 17 and 18 hardcode `bench/loomyard-eval/ladder/results/2026-09-04-breadth/conclusion.md` and `` `2026-09-04-breadth`, reps 5 ``, so a date change leaves the roadmap citing a path that does not exist in a tracked doc.
**Fix:** Widen the decision to name batches 5 and 6 and the overview's `## All Files Touched` as the substitution scope.

### [NIT:consistency] Card 14 has no commit despite the shared prefix decision
**Location:** overview `### Decision: conventional-commit-prefixes`; batch 5 card 14
**Issue:** The decision reads "Every card's `Commit:` uses a conventional-commit prefix", but card 14 is a preflight that changes no file and declares `**Commit:** none`.
**Fix:** Qualify the decision to cards that produce a diff, so the preflight card is not read as a violation.

### [NIT:scope] A few identifiers named outside their card's Context
**Location:** batch 1 cards 4 and 5; batch 2 card 7
**Issue:** Card 5 names `ExtractFencedJSON` (`fenced.go`) and `cfg.ID` (`config.go`); card 4 names `summary.Incomplete` (`summarize.go`) and `Run` (`run.go`); card 7 names `validate` matching `configs[].task` (`config.go`) — none of those source files is in the respective `Context:` or `Edits:`.
**Fix:** Each identifier is reachable from an existing line in the card's own `Edits:` file, so either add the file or drop the identifier from the prose; nothing here forces cold-start exploration.

### [NIT:consistency] Card 15 prescribes a report step the run subcommand already performs
**Location:** batch 5 card 15
**Issue:** `cmd/ladder`'s `runCommand` already calls `summarizeAndReport(*resultsRoot)` after `ladder.Run`, so `summary.json` and `table.txt` exist when the invocation returns; the card's "then run the report path" reads as a required second step.
**Fix:** State it as a re-derivation only needed after a resume or a killed invocation, not as an unconditional follow-up.

## Verdict

REQUEST_CHANGES
Two blocking items: the copied reed schema example, and the date-substitution scope.
MILL_REVIEW_END
