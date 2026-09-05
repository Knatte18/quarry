MILL_REVIEW_BEGIN
# Review: Glyph self-form and the resolve contract (C1) — holistic

```yaml
verdict: APPROVE
reviewer_model: sonnethigh
reviewed_file: plan/ + source
date: 2026-09-05
```

## Findings

### [NIT:scope] `codeForExpandError`'s new SelfGlyphError arm has no unit-test row
**Location:** `internal/cli/cli_test.go:1046-1074` (`TestCodeForExpandError`); arm added at `internal/cli/cli.go:136-139`.
**Issue:** Card 27 adds a `*quarry.SelfGlyphError` arm to `codeForExpandError`, and batch 4's own "Batch Tests" section names `TestCodeForExpandError` as the regression gate that "fails if a code moves without its table row" — but the table has no row for the new arm, only for `*NotATypeError` and `*glyph.ParseError`. The behaviour is still exercised end-to-end by `TestRun_Expand`'s `self-glyph` subtest, so this is a gap in the narrow unit gate the plan calls out, not a functional hole.
**Fix:** Add a `{"self-glyph-error", &quarry.SelfGlyphError{ID: "x#"}, exitNegative}` row (and its wrapped-error counterpart, mirroring the `NotATypeError` rows) to close the gate the plan's own text promises.

## Verdict

APPROVE
Every batch's cards are realised, decisions are applied consistently, and cross-batch contracts hold; only a minor test-table gap found.
MILL_REVIEW_END
