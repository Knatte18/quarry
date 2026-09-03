MILL_REVIEW_BEGIN
# Review: The glyph package (T1) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: plan/
date: 2026-09-03
```

## Findings

### [BLOCKING:scope] ParseError.Detail is prescribed but never asserted
**Location:** batch 2, cards 4 and 6 (and discussion's Detail column)
**Issue:** Card 4 fixes an exact `Detail` value for all sixteen reasons — quoted rune for `unit_bad_rune`/`member_bad_rune`/`member_type_params`/`member_parens`/`member_pointer`, the segment for `unit_dot_segment`, the component for `member_keyword`/`member_not_identifier`, the whole member half for `member_too_deep`, `string(lang)` for `unsupported_language`, empty for the rest — and even fixes the `%q` rendering (`' '` for a space) as a batch-local decision, yet no test in cards 5, 6 or 7 reads `Detail` at all. Card 6's field test covers `Lang` and `Input` only, on the stated ground that "the reject rows assert `Reason` alone and would not catch an unset field"; the identical argument applies to `Detail` and is not followed through, and the `Error()` tests hold `Detail` fixed so a wrong value stays invisible.
**Fix:** add `Detail` to card 6's `ParseError` field test (or a `wantDetail` column on `rejectCase`) covering at least one rune-quoting reason, one segment/component reason, `member_too_deep`, `unsupported_language`, and one blank-`Detail` reason.

### [NIT:consistency] Shared Decision's blank-line rule contradicts card 1 for doc.go
**Location:** overview `### Decision: file-level comments and godoc match the toc package`; batch 1 card 1
**Issue:** The Decision says *every* new non-test file opens with a file-level comment above `package glyph` separated by a blank line, but card 1 says `glyph/doc.go` holds only the package doc comment and the package clause — and a blank line there would detach the package doc the Decision's own rationale says doc.go owns.
**Fix:** scope the Decision's blank-line rule to the non-doc files (`glyph.go`, `errors.go`, `parse.go`, `golang.go`) and state that `doc.go` is the one file whose comment abuts the package clause.

### [NIT:consistency] No file-level comment required on the three test files
**Location:** batch 1 card 3; batch 2 cards 5, 6
**Issue:** `internal/quarryengine/toc/toc_test.go` opens with a file-level comment naming what the file covers, and the discussion cites that convention as one to follow, but the Shared Decision is scoped to non-test files and no test card asks for one.
**Fix:** state in each test card that the file opens with a file-level comment naming the tables it holds, matching `toc_test.go`.

### [NIT:consistency] Card 8 is a zero-diff card with `Commit: none`
**Location:** batch 2 card 8
**Issue:** Card 8 produces no diff and carries `Commit: none`, and its items 2 and 3 (`go vet ./...`, `golangci-lint run`) restate the overview's module-wide `verify:` that already runs at the batch boundary.
**Fix:** either declare explicitly that a zero-diff card is a supported card shape in this plan, or reduce card 8 to the two checks the verify commands do not cover (`go list -deps ./glyph` and the test-import read) and fold the rest into `verify:`.

## Verdict

REQUEST_CHANGES
Plan is sound and well sequenced; the prescribed `Detail` values are wholly untested.
MILL_REVIEW_END
