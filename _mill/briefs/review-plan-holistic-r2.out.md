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

### [BLOCKING:design] ParseError.Lang is never populated on Go rejects
**Location:** batch 2, card 4 (and card 6's reject assertions)
**Issue:** Card 4 specifies `Reason` and `Detail` for all fifteen Go reject paths but never says the `*ParseError` carries `Lang`; the prescribed signature `parseGo(input, unit, member string)` has no `lang` parameter, so an implementer can leave `Lang` at `""` for every unit/member reject while card 2 requires `Error()` to compose its message from `Reason`, `Lang` and `Input`.
**Fix:** State in card 4 that every `*ParseError` built in `parse.go` and `golang.go` sets `Lang: Go` (or `Lang: lang`) and `Input:` the original `s`, and add one card-6 assertion on those two fields, since the reject rows assert `Reason` only and would not catch it.

### [BLOCKING:consistency] `parseReject`/`goReject` field shape is unpinned and contradicted
**Location:** batch 2, Batch Scope + card 5 item 3 + card 6 reject table
**Issue:** Both slices are anonymous-struct literals defined as "the same field shape as" each other, with no field names, order or types fixed; Go type identity for anonymous structs is exact, so two independently written literals will not admit the completeness test that "ranges over both". The Batch Scope decision "Test-case structs carry a `section` field" further contradicts cards 5 and 6, which give `parseReject` and `goReject` three fields with no `section` (only `goAccept` has one).
**Fix:** Name the shared row type once in card 5 (e.g. a package-level `rejectCase` struct with explicit fields, including or explicitly excluding `section`) and have card 6 reuse that named type rather than restating a shape.

### [NIT:consistency] Card 5's internal cross-references are off by one
**Location:** batch 2, card 5 item 3
**Issue:** It cites "the `unsupported_language` cases of item 3 below, the reject-precedence case of item 4 and the `invalid_utf8` case of item 5", but those are items 4, 5 and 6; item 3 is the sentence itself.
**Fix:** Renumber the three references to 4, 5 and 6.

### [NIT:consistency] Unreachable dispatch default contradicts the plan's own rule
**Location:** batch 2, card 4 step 4
**Issue:** Step 1 already returns for `lang != Go`, so the `default:` arm returning `ReasonUnsupportedLanguage` can never fire — the exact shape the same card forbids for the unit `#` check ("a check that can never fire is worse than none") — and its `Detail` is left unspecified, unlike step 1's `string(lang)`.
**Fix:** Say explicitly that the default exists only to satisfy Go's terminating-statement rule, and fix its `Detail` to `string(lang)` so the two constructions agree.

### [NIT:consistency] File-level comments risk becoming rival package doc comments
**Location:** overview, Shared Decision "file-level comments and godoc match the toc package"
**Issue:** The decision says each non-test file opens with a comment "above the `package glyph` clause", but `internal/quarryengine/toc/types.go` separates that comment from `package toc` with a blank line; without the blank line, `glyph.go`, `errors.go`, `parse.go` and `golang.go` each become a second package doc comment competing with `doc.go`.
**Fix:** State the blank-line separation in the decision, since `doc.go` is the only file that owns the package comment.

## Verdict

REQUEST_CHANGES
Two blocking gaps: unset ParseError.Lang and an unpinned shared reject-row struct type.
MILL_REVIEW_END
