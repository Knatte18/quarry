MILL_REVIEW_BEGIN
# Review: Batch-answer contract: per-target coverage + fail-closed Status helpers

```yaml
duration_s: 193.0
verdict: APPROVE
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic)
reviewed_file: /home/knatte/Code/quarry/wts/batch-answer-contract/_mill/discussion.md
date: 2026-09-09
```

## Findings

### [NIT:consistency] Scope's test bullet lists fewer tests than Testing
**Section:** §Scope "In" (Tests bullet) vs §Testing **Issue:** Testing also prescribes a bogus-value row in `TestCodeForResolveResult` (verified: `internal/cli/cli_test.go:1140` has five rows, no `default` row), a `quarry.NameReasons` twin assertion, and a `quarry/text_test.go` rejection-case confirmation (present at lines 307/312) — none named in Scope. **Fix:** Fold the three extra test items into the Scope test bullet so the inventories agree.

### [NIT:design] `Name` echo-mismatch message shape has no exemplar
**Section:** §Decisions "Enforcement is a panic from a producer-side verifier" **Issue:** Only `resolve`'s message is spelled verbatim; for `Name` it is unstated whether the message names the diverging field (`unit` vs `target`), so a `Target`-only divergence yields a got/want pair with no field label, and the prescribed exact-string test has nothing to target. **Fix:** Give the `Name` message one verbatim exemplar, as `resolve` gets.

### [NIT:design] §5 paragraph vs `resolve`'s whole-call failure path
**Section:** §Decisions "`docs/glyph.md` §5 carries both statements" **Issue:** The prescribed normative wording ("one answer per input, in argument order") is unqualified, while `resolve` returns `nil, err` on engine failure (verified `internal/engine/resolve.go:417`); the doc decision does not say whether §5 states that exception. **Fix:** Say explicitly whether §5 qualifies the contract to successful calls or stays silent on the error path.

## Verdict

APPROVE
All material claims verified against the tree; three wording-level NITs only.
MILL_REVIEW_END
