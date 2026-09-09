MILL_REVIEW_BEGIN
# Review: Batch-answer contract: per-target coverage + fail-closed Status helpers — holistic

```yaml
verdict: APPROVE
reviewer_model: opushigh
reviewer_self_id: Claude Opus (system-declared "Opus 5"); I cannot independently confirm the version beyond that
reviewed_file: plan/
date: 2026-09-09
```

## Findings

### [NIT:consistency] `./glyph/` does not actually cover card 10
**Location:** batch 3 / Batch Tests bullet 3 **Issue:** `glyph/docs_test.go` is a hand-written mirror — its `docsAccept`/`docsReject` tables are Go literals and nothing in the file reads `docs/glyph.md` — so `go test ./glyph/` passes regardless of card 10's markdown edit and "covers card 10" overstates it. **Fix:** Reword to "must keep passing; the doc edit itself is unverified by test", matching card 10's own accurate framing of the no-new-glyph-string rule.

### [NIT:scope] Conditional edit to an undeclared file
**Location:** batch 3 / Batch Tests bullet 2 **Issue:** "if any is missing, add it before the rewrite" authorizes editing `quarry/text_test.go`, which appears in no card's `Edits:` and not in `## All Files Touched`; the three rows do exist (`quarry/text_test.go:306`, `:311`, `:316`), so the branch is dead but the instruction is live. **Fix:** State the rows as verified-present and drop the conditional, or declare the file on card 7's `Edits:` and in `## All Files Touched`.

### [NIT:consistency] "return from the panic site" invites unreachable code
**Location:** batch 2 / card 5, `Unit`-first echo check **Issue:** `panic` never returns, so a literal reading of "and return from the panic site" can produce a `return` after `panic`, which the overview's own `go vet ./...` gate flags as unreachable — costing a fixer round for a clause the unconditional `panic` already satisfies. **Fix:** Reword to "the panic ends the check for that index; `Target` is compared only in the else branch".

## Verdict

APPROVE
Plan is sound and source-accurate; three cosmetic nits, none blocking implementation.
MILL_REVIEW_END
