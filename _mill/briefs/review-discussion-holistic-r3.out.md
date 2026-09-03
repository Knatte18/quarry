MILL_REVIEW_BEGIN
# Review: The glyph package (T1)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5), high reasoning effort
reviewed_file: _mill/discussion.md
date: 2026-09-03
```

## Findings

### [BLOCKING:consistency] member_bad_rune is unreachable under stated precedence
**Section:** Decisions → "One `*ParseError`…" (precedence paragraph) vs Testing → reject table
**Issue:** Precedence puts `member_not_identifier` before `member_bad_rune` "last as the fallback", but every bad rune (`#`, `/`, control, whitespace) also makes the component a non-identifier, so `member_bad_rune` can never fire — contradicting the three table rows that require it (`a#b`, `A .b`, `run ` trailing space) and the same section's own definition of `member_not_identifier` as "a reason no sharper reason covers".
**Fix:** State the intended order explicitly — per-component `member_bad_rune` (the four-rune set) before `member_not_identifier` — or change those three rows to `member_not_identifier` and delete the constant.

### [BLOCKING:design] Reason-completeness test cannot fail the build as claimed
**Section:** Testing → "A completeness test iterates the `Reason` constants…"
**Issue:** Go has no reflection over package-level constants, so the test must iterate a hand-maintained slice; adding a sixteenth constant without adding it to that slice compiles and passes, making the claim "adding a sixteenth constant without a case fails the build" false and the check the tautology the discussion says it is not.
**Fix:** Name the enumeration source (e.g. an exported `Reasons []Reason` in `errors.go` that the closed-vocabulary decision requires updating) and restate the guarantee as what it actually is.

### [BLOCKING:design] Unit-half reject precedence is left undefined
**Section:** Decisions → "The Go unit alphabet" / "The split is at the first `#`"
**Issue:** Order is fixed for the member reasons and for language-vs-split, but not among `unit_empty_segment`, `unit_dot_segment` and `unit_bad_rune`; an input tripping two (e.g. `internal/../lo gger#run`) has no defined reason, and no reject-table row pins it, so two plan writers produce different, equally-conforming parsers.
**Fix:** State the unit-side order (check kind first, or segment-by-segment left to right with a within-segment order) as the member side does, and add one doubly-invalid unit row to the reject table.

## Verdict

REQUEST_CHANGES
Two precedence rules are contradictory or missing; one test-completeness claim rests on a false premise.
MILL_REVIEW_END
